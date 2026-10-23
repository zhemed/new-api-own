package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
)

// 自更新的稳定错误码（面板/脚本按码分支，Message 只作展示）。
const (
	// UpdateApplyCodeSignatureMismatch 表示 SHA256SUMS 的签名缺失或验签失败；此时绝不替换二进制。
	UpdateApplyCodeSignatureMismatch = "signature_mismatch"
	// UpdateApplyCodeBackupFailed 表示无法生成回滚点（<exe>.prev），为安全起见中止替换。
	UpdateApplyCodeBackupFailed = "backup_failed"
	// UpdateApplyCodeNoRollbackPoint 表示本机没有可用的回滚点。
	UpdateApplyCodeNoRollbackPoint = "no_rollback_point"
	// UpdateApplyCodeRollbackPointInvalid 表示回滚点存在但不像一个可执行文件。
	UpdateApplyCodeRollbackPointInvalid = "rollback_point_invalid"
	// UpdateApplyCodeRollbackFailed 表示回滚替换失败（当前二进制保持原样）。
	UpdateApplyCodeRollbackFailed = "rollback_failed"
)

const (
	// updatePreviousBinarySuffix 是回滚点后缀：<exe>.prev（同目录、同权限）。
	updatePreviousBinarySuffix = ".prev"
	// updateInPlaceHint 是"就地替换、不持久"的说明文本，直接透给面板展示。
	updateInPlaceHint = "本次为容器内就地替换；重建容器会回退到镜像内版本"
	// updateRollbackHint 是回滚成功后的说明文本。
	updateRollbackHint = "已从本机 .prev 回滚；回滚点已被消费，且仍是容器内就地替换，重建容器会回退到镜像内版本"
)

// updateSigningPublicKeys 是内嵌的 ed25519 公钥列表，用于验证 SHA256SUMS.sig（支持轮换，可放多把）。
//
// 约定（与 CI 侧 task-45 对齐）：
//   - 公钥：raw 32 字节 ed25519 的**标准 base64**（也兼容 PEM/PKIX 写法）；
//   - 签名对象：`SHA256SUMS` 文件的**原始字节**，不做任何归一化（只对 .sig 内容 TrimSpace）；
//   - `.sig` 内容：一行标准 base64（RFC 4648，带 `=` 填充）编码的 64 字节签名 + 换行；
//     解析失败或长度不是 64 字节 → 一律 signature_mismatch（对外稳定码）。
//
// **轮换必须发版**：CI 只能校验「secret 与内置公钥一致」，发现不了"Go 内嵌公钥没换"。
// 换钥流程 = 先在本列表加入新公钥并发版 → 再用新私钥签名 → 确认无旧版本需要后删旧公钥。
// 列表为空时自更新一律 signature_mismatch（fail closed）：宁可不升级，也不接受未验签的二进制。
// 老 Release 资产没有 .sig 同样拒绝替换，不降级放行。
var updateSigningPublicKeys = []string{
	"Jpj2LFm38b+U4HM/jK+4H1jJqsdiPBqw5kLx6Ihj2r8=",
}

// UpdateApplyError 是"立即更新"的可对外错误：Code 便于面板分支，Message 已脱敏。
type UpdateApplyError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *UpdateApplyError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func newUpdateApplyError(code string, status int, message string) *UpdateApplyError {
	return &UpdateApplyError{Code: code, Message: message, HTTPStatus: status}
}

// UpdateApplyResult 是"立即更新"成功后的结果。
type UpdateApplyResult struct {
	Applied         bool   `json:"applied"`
	Version         string `json:"version"`
	PreviousVersion string `json:"previous_version"`
	// RestartMode 说明新二进制如何生效：exec=原地重执行，exit=退出进程交给容器重启策略。
	RestartMode string `json:"restart_mode"`
	// Persistent 恒为 false：替换只发生在运行中的容器内，重建容器会回到镜像内版本。
	Persistent bool `json:"persistent"`
	// Hint 是给面板直接展示的就地替换说明。
	Hint string `json:"hint"`
	// ExecutablePath 是替换后的可执行文件路径，仅供进程内生效使用，不外发。
	ExecutablePath string `json:"-"`
}

// UpdateRollbackResult 是"回滚到上一个版本"成功后的结果。
type UpdateRollbackResult struct {
	RolledBack bool `json:"rolled_back"`
	// RestartMode 与 Apply 相同：exec=原地重执行。
	RestartMode string `json:"restart_mode"`
	// Persistent 恒为 false：回滚同样只是容器内就地替换。
	Persistent bool `json:"persistent"`
	// Hint 是给面板直接展示的说明。
	Hint string `json:"hint"`
	// ExecutablePath 供进程内生效使用，不外发。
	ExecutablePath string `json:"-"`
}

// UpdateApplier 负责下载、校验并原子替换本机二进制。
//
// 注意（容器部署的已知事故）：本流程替换的是**运行中的容器内**二进制文件。
// 下一次 `docker rm` / `docker run`（或重建容器）会回到镜像内的版本，更新结果不会被保留；
// 长期升级仍应以发布新镜像 + 重建容器为准，本接口只解决"当前实例立即用上新版本"。
type UpdateApplier struct {
	fetcher    ReleaseFetcher
	downloader httpDoer
	configErr  error

	// 依赖注入点（测试用，生产走默认实现）。
	goos              string
	goarch            string
	currentExecutable func() (string, error)
	execSelf          func(executablePath string) error
	exitProcess       func(code int)

	timeout          time.Duration
	maxBinaryBytes   int64
	maxChecksumBytes int64

	mu            sync.Mutex
	applyInFlight bool
}

func newUpdateApplier(fetcher ReleaseFetcher, downloader httpDoer) *UpdateApplier {
	return &UpdateApplier{
		fetcher:           fetcher,
		downloader:        downloader,
		goos:              runtime.GOOS,
		goarch:            runtime.GOARCH,
		currentExecutable: os.Executable,
		execSelf:          execSelf,
		exitProcess:       os.Exit,
		timeout:           setting.UpdateApplyTimeout,
		maxBinaryBytes:    setting.UpdateApplyMaxBinaryBytes,
		maxChecksumBytes:  setting.UpdateApplyMaxChecksumBytes,
	}
}

// Apply 下载并校验目标版本二进制，校验通过后原子替换当前可执行文件。
// 任何一步失败都不会改动现有二进制：临时文件在**同目录**写入，成功才 rename 覆盖。
// 它只做"替换"，不改变当前进程；生效由调用方在响应写完后调用 ActivateAppliedUpdate。
func (a *UpdateApplier) Apply(ctx context.Context) (*UpdateApplyResult, error) {
	if !setting.IsUpdateApplyEnabled() {
		return nil, newUpdateApplyError("disabled", http.StatusForbidden, "update apply is disabled")
	}
	if a.configErr != nil {
		return nil, newUpdateApplyError("config_invalid", http.StatusInternalServerError, a.configErr.Error())
	}
	if !a.beginApply() {
		return nil, newUpdateApplyError("busy", http.StatusConflict, "another update is already in progress")
	}
	defer a.endApply()

	if a.goos != "linux" {
		return nil, newUpdateApplyError("unsupported_platform", http.StatusBadRequest,
			"self update is only supported on linux")
	}
	if a.goarch != "amd64" && a.goarch != "arm64" {
		return nil, newUpdateApplyError("unsupported_platform", http.StatusBadRequest,
			"no release asset for architecture "+a.goarch)
	}
	assetName := UpdateAssetNameForArch(a.goarch)

	executablePath, err := a.currentExecutable()
	if err != nil {
		return nil, newUpdateApplyError("executable_unknown", http.StatusInternalServerError, "cannot locate current executable")
	}
	executablePath, err = filepath.EvalSymlinks(executablePath)
	if err != nil {
		return nil, newUpdateApplyError("executable_unknown", http.StatusInternalServerError, "cannot resolve current executable")
	}

	fetchCtx, cancelFetch := context.WithTimeout(ctx, setting.UpdateCheckTimeout)
	release, err := a.fetcher.FetchLatestRelease(fetchCtx)
	cancelFetch()
	if err != nil {
		common.SysError(fmt.Sprintf("update apply failed to resolve release: %s", common.MaskSensitiveInfo(err.Error())))
		return nil, newUpdateApplyError("source_unavailable", http.StatusBadGateway, "update source is unavailable")
	}

	// 只允许前进：没有更新或版本无法比较时拒绝，避免误降级/误覆盖。
	if common.CompareVersions(common.Version, release.TagName) != common.VersionComparisonNewer {
		return nil, newUpdateApplyError("not_newer", http.StatusConflict,
			"running version is not older than the latest release")
	}

	binaryAsset, ok := findUpdateAsset(release.Assets, assetName)
	if !ok || strings.TrimSpace(binaryAsset.DownloadURL) == "" {
		return nil, newUpdateApplyError("asset_missing", http.StatusBadGateway,
			"release has no asset "+assetName)
	}
	checksumAsset, ok := findUpdateAsset(release.Assets, UpdateAssetNameSHA256Sums)
	if !ok || strings.TrimSpace(checksumAsset.DownloadURL) == "" {
		return nil, newUpdateApplyError("checksum_asset_missing", http.StatusBadGateway,
			"release has no "+UpdateAssetNameSHA256Sums)
	}
	signatureAsset, ok := findUpdateAsset(release.Assets, UpdateAssetNameSHA256SumsSig)
	if !ok || strings.TrimSpace(signatureAsset.DownloadURL) == "" {
		// 安全优先：没有签名资产就不再往下走，也不降级放行（老版本资产同样被拒绝）。
		return nil, newUpdateApplyError(UpdateApplyCodeSignatureMismatch, http.StatusBadGateway,
			"release has no "+UpdateAssetNameSHA256SumsSig+"; refusing to replace the binary")
	}

	sumsBody, expectedSum, err := a.fetchChecksum(ctx, checksumAsset.DownloadURL, assetName)
	if err != nil {
		return nil, err
	}
	signature, err := a.download(ctx, signatureAsset.DownloadURL, setting.UpdateApplyMaxSignatureBytes, setting.UpdateCheckTimeout)
	if err != nil {
		return nil, err
	}
	// 验签发生在下载二进制之前：签名不过关时连临时文件都不会创建，更不会触碰现有二进制。
	if err := verifySHA256SumsSignature(sumsBody, signature); err != nil {
		common.SysError(fmt.Sprintf("update apply signature verification failed: %s", common.MaskSensitiveInfo(err.Error())))
		return nil, newUpdateApplyError(UpdateApplyCodeSignatureMismatch, http.StatusBadGateway,
			"release checksums are not signed by a trusted key; refusing to replace the binary")
	}

	tempPath, err := a.downloadVerifiedBinary(ctx, binaryAsset.DownloadURL, expectedSum, filepath.Dir(executablePath))
	if err != nil {
		return nil, err
	}

	// 替换前留回滚点 <exe>.prev（复制而非改名：替换失败时原二进制必须原地保留）。
	if err := backupCurrentBinary(executablePath); err != nil {
		_ = os.Remove(tempPath)
		common.SysError(fmt.Sprintf("update apply cannot create rollback point: %s", common.MaskSensitiveInfo(err.Error())))
		return nil, newUpdateApplyError(UpdateApplyCodeBackupFailed, http.StatusInternalServerError,
			"cannot create the rollback point next to the running binary")
	}

	// 原子替换：同目录 rename。进程正在运行也不受影响（旧 inode 继续服务，
	// 也避免了直接写运行中文件导致的 ETXTBSY）。
	if err := os.Rename(tempPath, executablePath); err != nil {
		_ = os.Remove(tempPath)
		return nil, newUpdateApplyError("replace_failed", http.StatusInternalServerError, "cannot replace the running binary")
	}

	return &UpdateApplyResult{
		Applied:         true,
		Version:         release.TagName,
		PreviousVersion: common.Version,
		RestartMode:     "exec",
		Persistent:      false,
		Hint:            updateInPlaceHint,
		ExecutablePath:  executablePath,
	}, nil
}

// Activate 让刚替换的二进制生效：优先 syscall.Exec 原地重执行（不依赖外部 supervisor，
// 也不依赖容器的 restart 策略）；exec 不可用时退出进程，交给容器 restart 策略。
// 正常路径不会返回（进程镜像被替换，或进程退出）。
//
// 优雅退出（drain）评估结论：本轮**刻意不做**。要做需要把 `*http.Server` 的 Shutdown 钩子
// 从 main 注入本包（跨 main/包边界，牵动启动流程，回归面大）；而升级由管理员手动触发、
// 低频，且新进程约 1s 就绪。残余影响：**触发更新的那一刻**在途请求会随镜像替换断开。
// 后续若要补：在 main 里注册 drainBeforeExec 钩子（Shutdown 带 5-10s 超时），exec 前调用，
// 并配专项演练与超时兜底。
func (a *UpdateApplier) Activate(executablePath string) error {
	if err := a.execSelf(executablePath); err != nil {
		common.SysError(fmt.Sprintf("in-place re-exec failed, exiting for container restart: %s",
			common.MaskSensitiveInfo(err.Error())))
		a.exitProcess(0)
	}
	return nil
}

func (a *UpdateApplier) beginApply() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.applyInFlight {
		return false
	}
	a.applyInFlight = true
	return true
}

func (a *UpdateApplier) endApply() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.applyInFlight = false
}

// fetchChecksum 下载 SHA256SUMS，返回其**原始字节**（验签对象）与目标资产的期望校验和。
// 取不到就不允许继续。
func (a *UpdateApplier) fetchChecksum(ctx context.Context, url string, assetName string) ([]byte, string, error) {
	body, err := a.download(ctx, url, a.maxChecksumBytes, setting.UpdateCheckTimeout)
	if err != nil {
		return nil, "", err
	}
	sum, ok := parseSHA256Sums(string(body), assetName)
	if !ok {
		return nil, "", newUpdateApplyError("checksum_missing", http.StatusBadGateway,
			"release checksums do not cover "+assetName)
	}
	return body, sum, nil
}

// verifySHA256SumsSignature 用内嵌公钥验证 SHA256SUMS 的 ed25519 分离签名。
// 签名格式容错：base64（标准/无填充/URL）或 hex，允许前后空白与换行。
// 公钥列表为空（尚未内嵌）、签名缺失/格式错误、或没有任何公钥能验证通过 → 一律报错。
func verifySHA256SumsSignature(sumsContent []byte, signature []byte) error {
	signatureBytes, err := decodeSignature(string(signature))
	if err != nil {
		return fmt.Errorf("cannot decode %s: %w", UpdateAssetNameSHA256SumsSig, err)
	}
	if len(updateSigningPublicKeys) == 0 {
		return errors.New("no update signing public key is embedded in this build")
	}
	for _, encoded := range updateSigningPublicKeys {
		publicKey, err := decodePublicKey(encoded)
		if err != nil {
			common.SysError("update signing public key is not usable: " + common.MaskSensitiveInfo(err.Error()))
			continue
		}
		if ed25519.Verify(publicKey, sumsContent, signatureBytes) {
			return nil
		}
	}
	return errors.New("signature does not match any embedded public key")
}

// decodeSignature 解析 .sig 文本：base64（标准/无填充/URL 变体）或 hex。
func decodeSignature(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("signature is empty")
	}
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if decoded, err := encoding.DecodeString(trimmed); err == nil && len(decoded) == ed25519.SignatureSize {
			return decoded, nil
		}
	}
	if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) == ed25519.SignatureSize {
		return decoded, nil
	}
	return nil, errors.New("signature must be base64 or hex of 64 bytes")
}

// decodePublicKey 解析内嵌公钥：base64 raw 32 字节，或 PEM（PKIX/SPKI）。
func decodePublicKey(encoded string) (ed25519.PublicKey, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, errors.New("public key is empty")
	}
	if block, _ := pem.Decode([]byte(trimmed)); block != nil {
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse PEM public key: %w", err)
		}
		publicKey, ok := parsed.(ed25519.PublicKey)
		if !ok {
			return nil, errors.New("PEM public key is not ed25519")
		}
		return publicKey, nil
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding} {
		if decoded, err := encoding.DecodeString(trimmed); err == nil && len(decoded) == ed25519.PublicKeySize {
			return ed25519.PublicKey(decoded), nil
		}
	}
	return nil, errors.New("public key must be base64 of 32 raw bytes or a PEM ed25519 key")
}

// backupCurrentBinary 把当前可执行文件复制为 <exe>.prev，作为回滚点（同目录、同权限）。
// 用复制而不是改名：替换失败时原二进制必须原地保留；先写临时文件再改名，避免留下半截 .prev。
func backupCurrentBinary(executablePath string) error {
	source, err := os.Open(executablePath)
	if err != nil {
		return fmt.Errorf("cannot read the running binary: %w", err)
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat the running binary: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(executablePath), ".new-api-prev-*")
	if err != nil {
		return fmt.Errorf("cannot create the rollback point: %w", err)
	}
	tempPath := tempFile.Name()
	discard := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := io.Copy(tempFile, source); err != nil {
		discard()
		return fmt.Errorf("cannot copy the running binary: %w", err)
	}
	if err := tempFile.Chmod(info.Mode().Perm()); err != nil {
		discard()
		return fmt.Errorf("cannot preserve the binary permissions: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		discard()
		return fmt.Errorf("cannot flush the rollback point: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("cannot close the rollback point: %w", err)
	}
	if err := os.Rename(tempPath, executablePath+updatePreviousBinarySuffix); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("cannot activate the rollback point: %w", err)
	}
	return nil
}

// Rollback 用本机回滚点 <exe>.prev 原子换回上一个版本。
// 只使用本地文件：不下载任何旧版本。回滚点会被消费（rename），下一次更新才会重新生成。
// 失败时当前二进制保持原样（rename 是原子的）。
func (a *UpdateApplier) Rollback() (*UpdateRollbackResult, error) {
	if !setting.IsUpdateApplyEnabled() {
		return nil, newUpdateApplyError("disabled", http.StatusForbidden, "update apply is disabled")
	}
	if a.configErr != nil {
		return nil, newUpdateApplyError("config_invalid", http.StatusInternalServerError, a.configErr.Error())
	}
	// 与 Apply 共用同一把锁：避免替换与回滚并发。
	if !a.beginApply() {
		return nil, newUpdateApplyError("busy", http.StatusConflict, "another update is already in progress")
	}
	defer a.endApply()

	if a.goos != "linux" {
		return nil, newUpdateApplyError("unsupported_platform", http.StatusBadRequest,
			"self update is only supported on linux")
	}

	executablePath, err := a.currentExecutable()
	if err != nil {
		return nil, newUpdateApplyError("executable_unknown", http.StatusInternalServerError, "cannot locate current executable")
	}
	executablePath, err = filepath.EvalSymlinks(executablePath)
	if err != nil {
		return nil, newUpdateApplyError("executable_unknown", http.StatusInternalServerError, "cannot resolve current executable")
	}

	previousPath := executablePath + updatePreviousBinarySuffix
	info, err := os.Lstat(previousPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, newUpdateApplyError(UpdateApplyCodeNoRollbackPoint, http.StatusConflict,
			"no rollback point ("+filepath.Base(previousPath)+") on this host")
	}
	if err != nil {
		return nil, newUpdateApplyError(UpdateApplyCodeRollbackPointInvalid, http.StatusInternalServerError,
			"cannot read the rollback point")
	}
	if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > a.maxBinaryBytes {
		return nil, newUpdateApplyError(UpdateApplyCodeRollbackPointInvalid, http.StatusInternalServerError,
			"the rollback point is not a usable binary")
	}
	// 回滚点必须是本平台的 ELF 可执行文件，避免把截断/占位的文件换到 exe 上。
	if err := requireELFExecutable(previousPath); err != nil {
		return nil, newUpdateApplyError(UpdateApplyCodeRollbackPointInvalid, http.StatusInternalServerError,
			"the rollback point is not a linux executable")
	}

	if err := os.Rename(previousPath, executablePath); err != nil {
		return nil, newUpdateApplyError(UpdateApplyCodeRollbackFailed, http.StatusInternalServerError,
			"cannot restore the previous binary")
	}

	return &UpdateRollbackResult{
		RolledBack:     true,
		RestartMode:    "exec",
		Persistent:     false,
		Hint:           updateRollbackHint,
		ExecutablePath: executablePath,
	}, nil
}

// requireELFExecutable 校验文件头是 ELF（回滚点 sanity check）。
func requireELFExecutable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		return err
	}
	if header[0] != 0x7f || header[1] != 'E' || header[2] != 'L' || header[3] != 'F' {
		return errors.New("not an ELF binary")
	}
	return nil
}

// downloadVerifiedBinary 下载二进制到可执行文件同目录的临时文件并校验 SHA256。
// 校验失败会删除临时文件并返回错误，绝不触碰现有二进制。
func (a *UpdateApplier) downloadVerifiedBinary(
	ctx context.Context, url string, expectedSum string, targetDir string,
) (string, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	request, err := a.newDownloadRequest(downloadCtx, url)
	if err != nil {
		return "", err
	}
	response, err := a.downloader.Do(request)
	if err != nil {
		common.SysError(fmt.Sprintf("update apply download failed: %s", common.MaskSensitiveInfo(err.Error())))
		return "", newUpdateApplyError("download_failed", http.StatusBadGateway, "cannot download the release asset")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", newUpdateApplyError("download_failed", http.StatusBadGateway,
			fmt.Sprintf("download returned status %d", response.StatusCode))
	}

	tempFile, err := os.CreateTemp(targetDir, ".new-api-update-*")
	if err != nil {
		return "", newUpdateApplyError("temp_file_failed", http.StatusInternalServerError,
			"cannot create a temporary file next to the running binary")
	}
	tempPath := tempFile.Name()
	discardTemp := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tempFile, hasher), io.LimitReader(response.Body, a.maxBinaryBytes+1))
	if err != nil {
		discardTemp()
		common.SysError(fmt.Sprintf("update apply download failed: %s", common.MaskSensitiveInfo(err.Error())))
		return "", newUpdateApplyError("download_failed", http.StatusBadGateway, "download was interrupted")
	}
	if written > a.maxBinaryBytes {
		discardTemp()
		return "", newUpdateApplyError("binary_too_large", http.StatusRequestEntityTooLarge,
			"release asset exceeds the configured size limit")
	}
	if err := tempFile.Chmod(0o755); err != nil {
		discardTemp()
		return "", newUpdateApplyError("temp_file_failed", http.StatusInternalServerError, "cannot mark the new binary executable")
	}
	if err := tempFile.Sync(); err != nil {
		discardTemp()
		return "", newUpdateApplyError("temp_file_failed", http.StatusInternalServerError, "cannot flush the new binary to disk")
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", newUpdateApplyError("temp_file_failed", http.StatusInternalServerError, "cannot close the new binary")
	}

	if actualSum := hex.EncodeToString(hasher.Sum(nil)); !strings.EqualFold(actualSum, expectedSum) {
		_ = os.Remove(tempPath)
		return "", newUpdateApplyError("checksum_mismatch", http.StatusBadGateway,
			"downloaded asset does not match "+UpdateAssetNameSHA256Sums)
	}
	return tempPath, nil
}

// download 取一个小体积文本资源（SHA256SUMS），带体积上限。
func (a *UpdateApplier) download(ctx context.Context, url string, maxBytes int64, timeout time.Duration) ([]byte, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := a.newDownloadRequest(downloadCtx, url)
	if err != nil {
		return nil, err
	}
	response, err := a.downloader.Do(request)
	if err != nil {
		common.SysError(fmt.Sprintf("update apply checksum download failed: %s", common.MaskSensitiveInfo(err.Error())))
		return nil, newUpdateApplyError("checksum_download_failed", http.StatusBadGateway, "cannot download the release checksums")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, newUpdateApplyError("checksum_download_failed", http.StatusBadGateway,
			fmt.Sprintf("checksum download returned status %d", response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, newUpdateApplyError("checksum_download_failed", http.StatusBadGateway, "checksum download was interrupted")
	}
	if int64(len(body)) > maxBytes {
		return nil, newUpdateApplyError("checksum_download_failed", http.StatusBadGateway, "checksum file is too large")
	}
	return body, nil
}

func (a *UpdateApplier) newDownloadRequest(ctx context.Context, url string) (*http.Request, error) {
	if strings.TrimSpace(url) == "" {
		return nil, newUpdateApplyError("asset_missing", http.StatusBadGateway, "release asset has no download url")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, newUpdateApplyError("asset_missing", http.StatusBadGateway, "release asset url is invalid")
	}
	// 只读下载，不带任何凭据（也不设置 Authorization）。
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", updateCheckUserAgent)
	return request, nil
}

// parseSHA256Sums 解析 sha256sum 输出格式：`<64 位十六进制>  <文件名>`（分隔符可为空格或 " *"）。
func parseSHA256Sums(content string, assetName string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		sum := strings.ToLower(fields[0])
		name := strings.TrimPrefix(fields[1], "*")
		if name != assetName || !isSHA256Hex(sum) {
			continue
		}
		return sum, true
	}
	return "", false
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for i := 0; i < len(value); i++ {
		character := value[i]
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func findUpdateAsset(assets []ReleaseAsset, name string) (ReleaseAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return ReleaseAsset{}, false
}

// execSelf 用新二进制原地替换当前进程镜像；其余 goroutine/连接会随镜像替换结束。
func execSelf(executablePath string) error {
	return syscall.Exec(executablePath, os.Args, os.Environ())
}

var (
	updateApplierOnce sync.Once
	updateApplier     *UpdateApplier
)

func defaultUpdateApplier() *UpdateApplier {
	updateApplierOnce.Do(func() {
		client, err := newUpdateSourceClient(setting.UpdateApplyTimeout)
		if err != nil {
			applier := newUpdateApplier(defaultUpdateSource(), nil)
			applier.configErr = errors.New("update proxy url is invalid")
			updateApplier = applier
			return
		}
		updateApplier = newUpdateApplier(defaultUpdateSource(), client)
	})
	return updateApplier
}

// ApplyLatestUpdate 下载并原子替换二进制；调用方需在响应写完后调用 ActivateLatestUpdate。
func ApplyLatestUpdate(ctx context.Context) (*UpdateApplyResult, error) {
	return defaultUpdateApplier().Apply(ctx)
}

// RollbackLatestUpdate 从本机 <exe>.prev 回滚到上一个版本（不下载任何旧版本）。
func RollbackLatestUpdate() (*UpdateRollbackResult, error) {
	return defaultUpdateApplier().Rollback()
}

// ActivateLatestUpdate 让新二进制生效（原地重执行或退出等容器拉起）。
func ActivateLatestUpdate(result *UpdateApplyResult) {
	if result == nil || result.ExecutablePath == "" {
		return
	}
	_ = defaultUpdateApplier().Activate(result.ExecutablePath)
}

// ActivateLatestRollback 让回滚后的二进制生效（与 Apply 相同的激活路径）。
func ActivateLatestRollback(result *UpdateRollbackResult) {
	if result == nil || result.ExecutablePath == "" {
		return
	}
	_ = defaultUpdateApplier().Activate(result.ExecutablePath)
}
