package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	// ExecutablePath 是替换后的可执行文件路径，仅供进程内生效使用，不外发。
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

	expectedSum, err := a.fetchChecksum(ctx, checksumAsset.DownloadURL, assetName)
	if err != nil {
		return nil, err
	}

	tempPath, err := a.downloadVerifiedBinary(ctx, binaryAsset.DownloadURL, expectedSum, filepath.Dir(executablePath))
	if err != nil {
		return nil, err
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
		ExecutablePath:  executablePath,
	}, nil
}

// Activate 让刚替换的二进制生效：优先 syscall.Exec 原地重执行（不依赖外部 supervisor，
// 也不依赖容器的 restart 策略）；exec 不可用时退出进程，交给容器 restart 策略。
// 正常路径不会返回（进程镜像被替换，或进程退出）。
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

// fetchChecksum 下载 SHA256SUMS 并取出目标资产的期望校验和；取不到就不允许继续。
func (a *UpdateApplier) fetchChecksum(ctx context.Context, url string, assetName string) (string, error) {
	body, err := a.download(ctx, url, a.maxChecksumBytes, setting.UpdateCheckTimeout)
	if err != nil {
		return "", err
	}
	sum, ok := parseSHA256Sums(string(body), assetName)
	if !ok {
		return "", newUpdateApplyError("checksum_missing", http.StatusBadGateway,
			"release checksums do not cover "+assetName)
	}
	return sum, nil
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

// ActivateLatestUpdate 让新二进制生效（原地重执行或退出等容器拉起）。
func ActivateLatestUpdate(result *UpdateApplyResult) {
	if result == nil || result.ExecutablePath == "" {
		return
	}
	_ = defaultUpdateApplier().Activate(result.ExecutablePath)
}
