package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const (
	testSumsURL   = "https://example.test/SHA256SUMS"
	testSigURL    = "https://example.test/SHA256SUMS.sig"
	testBinaryURL = "https://example.test/new-api-linux-amd64"
	testTag       = "v9.9.9"
)

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// setTestSigningKey 生成 ed25519 测试密钥对并把它写成"内嵌公钥"（测试结束还原），
// 返回与 CI 约定一致的签名函数：一行标准 base64（带 `=` 填充）+ 换行。
func setTestSigningKey(t *testing.T) func(message []byte) string {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	previousKeys := updateSigningPublicKeys
	updateSigningPublicKeys = []string{base64.StdEncoding.EncodeToString(publicKey)}
	t.Cleanup(func() { updateSigningPublicKeys = previousKeys })

	return func(message []byte) string {
		return base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, message)) + "\n"
	}
}

// newTestApplierWithSignature 构造完全离线的 applier：假数据源 + 假下载器 + 临时目录里的"当前二进制"。
// signature 为 nil 时不提供 SHA256SUMS.sig 资产（用于断言"缺签名一律拒绝"）。
func newTestApplierWithSignature(
	t *testing.T, newBinary []byte, sumsContent string, signature func(message []byte) string,
) (*UpdateApplier, *fakeReleaseFetcher, string) {
	t.Helper()
	if signature == nil {
		setTestSigningKey(t) // 仍内嵌可信公钥，只是数据源不给 .sig
	}
	executable := filepath.Join(t.TempDir(), "new-api")
	require.NoError(t, os.WriteFile(executable, []byte("old-binary"), 0o755))

	assets := []ReleaseAsset{
		{Name: UpdateAssetNameSHA256Sums, DownloadURL: testSumsURL},
		{Name: UpdateAssetNameForArch("amd64"), DownloadURL: testBinaryURL},
	}
	if signature != nil {
		assets = append(assets, ReleaseAsset{Name: UpdateAssetNameSHA256SumsSig, DownloadURL: testSigURL})
	}
	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{TagName: testTag, Assets: assets}}

	doer := &fakeHTTPDoer{respond: func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case testSumsURL:
			return newFakeHTTPResponse(http.StatusOK, sumsContent), nil
		case testSigURL:
			if signature == nil {
				return newFakeHTTPResponse(http.StatusNotFound, ""), nil
			}
			return newFakeHTTPResponse(http.StatusOK, signature([]byte(sumsContent))), nil
		case testBinaryURL:
			return newFakeHTTPResponse(http.StatusOK, string(newBinary)), nil
		default:
			return newFakeHTTPResponse(http.StatusNotFound, ""), nil
		}
	}}

	applier := newUpdateApplier(fetcher, doer)
	applier.currentExecutable = func() (string, error) { return executable, nil }
	applier.goos = "linux"
	applier.goarch = "amd64"
	return applier, fetcher, executable
}

// newTestApplier 构造带**有效签名**的 applier（绝大多数用例的基线）。
func newTestApplier(t *testing.T, newBinary []byte, sumsContent string) (*UpdateApplier, *fakeReleaseFetcher, string) {
	t.Helper()
	return newTestApplierWithSignature(t, newBinary, sumsContent, setTestSigningKey(t))
}

// newRollbackTestApplier 构造只用于回滚的 applier：当前二进制在临时目录，不涉及网络。
func newRollbackTestApplier(t *testing.T) (*UpdateApplier, string) {
	t.Helper()
	executable := filepath.Join(t.TempDir(), "new-api")
	require.NoError(t, os.WriteFile(executable, []byte("current-binary"), 0o755))
	applier := newUpdateApplier(nil, nil)
	applier.currentExecutable = func() (string, error) { return executable, nil }
	applier.goos = "linux"
	return applier, executable
}

func requireUpdateApplyError(t *testing.T, err error, code string) {
	t.Helper()
	var applyErr *UpdateApplyError
	require.ErrorAs(t, err, &applyErr)
	require.Equal(t, code, applyErr.Code)
	require.NotEmpty(t, applyErr.Message)
	require.NotZero(t, applyErr.HTTPStatus)
}

// requireBinaryUntouched 断言失败的中间态可自愈：磁盘上仍是旧二进制，且没有残留临时文件。
func requireBinaryUntouched(t *testing.T, executable string) {
	t.Helper()
	content, err := os.ReadFile(executable)
	require.NoError(t, err)
	require.Equal(t, "old-binary", string(content))
	_, err = os.Stat(executable + updatePreviousBinarySuffix)
	require.True(t, errors.Is(err, os.ErrNotExist), "失败后不得生成/改动回滚点")
	entries, err := os.ReadDir(filepath.Dir(executable))
	require.NoError(t, err)
	require.Len(t, entries, 1, "失败后不得留下临时文件")
}

func TestApplyUpdateReplacesBinaryAtomically(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	applier, fetcher, executable := newTestApplier(t, newBinary, sums)

	var execCalls []string
	exitCalledWith := -1
	applier.execSelf = func(path string) error {
		execCalls = append(execCalls, path)
		return nil
	}
	applier.exitProcess = func(code int) { exitCalledWith = code }

	result, err := applier.Apply(context.Background())

	require.NoError(t, err)
	require.True(t, result.Applied)
	require.Equal(t, testTag, result.Version)
	require.Equal(t, common.Version, result.PreviousVersion)
	require.False(t, result.Persistent, "就地替换不持久：面板必须据此提示")
	require.NotEmpty(t, result.Hint, "必须给出'重建容器会回退'的提示文案")
	require.Equal(t, 1, fetcher.calls)

	content, err := os.ReadFile(executable)
	require.NoError(t, err)
	require.Equal(t, newBinary, content, "替换后文件内容应为已校验的新二进制")
	info, err := os.Stat(executable)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "新二进制必须可执行")

	// 回滚点：<exe>.prev 保留旧二进制且权限一致。
	previousInfo, err := os.Stat(executable + updatePreviousBinarySuffix)
	require.NoError(t, err, "替换成功后必须留下回滚点")
	require.Equal(t, info.Mode().Perm(), previousInfo.Mode().Perm(), "回滚点权限必须与原文件一致")
	previousContent, err := os.ReadFile(executable + updatePreviousBinarySuffix)
	require.NoError(t, err)
	require.Equal(t, "old-binary", string(previousContent), "回滚点必须是替换前的二进制")

	entries, err := os.ReadDir(filepath.Dir(executable))
	require.NoError(t, err)
	require.Len(t, entries, 2, "替换完成后只应有新二进制与回滚点，不得残留临时文件")
	require.Empty(t, execCalls, "下载/替换阶段不得触发重执行")
	require.Equal(t, -1, exitCalledWith)

	applier.Activate(result.ExecutablePath)

	require.Equal(t, []string{result.ExecutablePath}, execCalls)
	require.Equal(t, -1, exitCalledWith, "exec 可用时不应退出进程")
}

func TestApplyUpdateChecksumMismatchKeepsBinary(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	// 校验和格式合法但内容不匹配：必须拒绝替换。
	sums := strings.Repeat("a", 64) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	applier, _, executable := newTestApplier(t, newBinary, sums)
	applier.execSelf = func(string) error { t.Fatal("校验失败时绝不能生效"); return nil }

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "checksum_mismatch")
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateMissingChecksumEntryKeepsBinary(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  other-asset\n"
	applier, _, executable := newTestApplier(t, newBinary, sums)

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "checksum_missing")
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateChecksumAssetMissingKeepsBinary(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	applier, fetcher, executable := newTestApplier(t, newBinary, "")
	fetcher.release.Assets = []ReleaseAsset{
		{Name: UpdateAssetNameForArch("amd64"), DownloadURL: testBinaryURL},
	}

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "checksum_asset_missing")
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateAssetMissingKeepsBinary(t *testing.T) {
	setUpdateSetting(t, true, true)
	applier, fetcher, executable := newTestApplier(t, []byte("new-binary-content"), "")
	fetcher.release.Assets = []ReleaseAsset{
		{Name: UpdateAssetNameSHA256Sums, DownloadURL: testSumsURL},
	}

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "asset_missing")
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateDownloadFailureKeepsBinary(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	sign := setTestSigningKey(t)
	applier, _, executable := newTestApplierWithSignature(t, newBinary, sums, sign)
	applier.downloader = &fakeHTTPDoer{respond: func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case testBinaryURL:
			return nil, errors.New("connection reset by peer")
		case testSigURL:
			return newFakeHTTPResponse(http.StatusOK, sign([]byte(sums))), nil
		default:
			return newFakeHTTPResponse(http.StatusOK, sums), nil
		}
	}}

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "download_failed")
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateRejectsOversizedBinary(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	applier, _, executable := newTestApplier(t, newBinary, sums)
	applier.maxBinaryBytes = 4

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "binary_too_large")
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateUnsupportedPlatformDoesNotFetch(t *testing.T) {
	setUpdateSetting(t, true, true)
	applier, fetcher, executable := newTestApplier(t, []byte("new-binary-content"), "")
	applier.goarch = "386"

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "unsupported_platform")
	require.Equal(t, 0, fetcher.calls)
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateDisabledDoesNotFetch(t *testing.T) {
	setUpdateSetting(t, true, false)
	applier, fetcher, executable := newTestApplier(t, []byte("new-binary-content"), "")

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "disabled")
	require.Equal(t, 0, fetcher.calls)
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateRejectsNonNewerTarget(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	applier, fetcher, executable := newTestApplier(t, newBinary, sums)
	fetcher.release.TagName = common.Version

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "not_newer")
	requireBinaryUntouched(t, executable)
}

func TestApplyUpdateRejectsConcurrentApply(t *testing.T) {
	setUpdateSetting(t, true, true)
	applier, _, executable := newTestApplier(t, []byte("new-binary-content"), "")
	applier.applyInFlight = true

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, "busy")
	requireBinaryUntouched(t, executable)
}

func TestActivateFallsBackToProcessExit(t *testing.T) {
	applier := newUpdateApplier(nil, nil)
	applier.execSelf = func(string) error { return errors.New("exec not supported") }
	exitCode := -1
	applier.exitProcess = func(code int) { exitCode = code }

	err := applier.Activate("/tmp/new-api")

	require.NoError(t, err)
	require.Equal(t, 0, exitCode, "exec 不可用时交给容器 restart 策略")
}

func TestParseSHA256Sums(t *testing.T) {
	const sum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tests := []struct {
		name      string
		content   string
		assetName string
		wantSum   string
		wantOK    bool
	}{
		{
			name:      "standard sha256sum line",
			content:   sum + "  new-api-linux-amd64\n",
			assetName: "new-api-linux-amd64",
			wantSum:   sum,
			wantOK:    true,
		},
		{
			name:      "binary mode marker is tolerated",
			content:   sum + " *new-api-linux-amd64\n",
			assetName: "new-api-linux-amd64",
			wantSum:   sum,
			wantOK:    true,
		},
		{
			name:      "crlf and extra lines are tolerated",
			content:   "aaaa  other\r\n" + sum + "  new-api-linux-arm64\r\n",
			assetName: "new-api-linux-arm64",
			wantSum:   sum,
			wantOK:    true,
		},
		{
			name:      "uppercase hex is normalized",
			content:   strings.ToUpper(sum) + "  new-api-linux-amd64\n",
			assetName: "new-api-linux-amd64",
			wantSum:   sum,
			wantOK:    true,
		},
		{
			name:      "asset not covered",
			content:   sum + "  new-api-linux-arm64\n",
			assetName: "new-api-linux-amd64",
			wantOK:    false,
		},
		{
			name:      "malformed hash is not accepted",
			content:   "deadbeef  new-api-linux-amd64\n",
			assetName: "new-api-linux-amd64",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sum, ok := parseSHA256Sums(tt.content, tt.assetName)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantSum, sum)
		})
	}
}

// --- 资产签名（ed25519）-------------------------------------------------------

// 签名不过关时：拒绝替换、返回 signature_mismatch、原二进制逐字节不变、不产生回滚点。
func TestApplyUpdateRejectsForgedSignature(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"

	// 内嵌公钥是测试可信公钥，但数据源给的是另一把私钥的签名。
	setTestSigningKey(t)
	_, forgedPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	forgedSigner := func(message []byte) string {
		return base64.StdEncoding.EncodeToString(ed25519.Sign(forgedPrivateKey, message)) + "\n"
	}

	applier, _, executable := newTestApplierWithSignature(t, newBinary, sums, forgedSigner)
	before := fileSHA256(t, executable)

	_, err = applier.Apply(context.Background())

	requireUpdateApplyError(t, err, UpdateApplyCodeSignatureMismatch)
	require.Equal(t, before, fileSHA256(t, executable), "验签失败后原二进制必须逐字节不变")
	requireBinaryUntouched(t, executable)
}

// 签名有效但覆盖的不是本次提供的 SHA256SUMS（例如资产被替换）：同样拒绝。
func TestApplyUpdateRejectsSignatureOverDifferentChecksums(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	signer := setTestSigningKey(t)
	// 签名对着另一份内容（篡改后的 SHA256SUMS），与服务端返回的 sums 不匹配。
	applier, _, executable := newTestApplierWithSignature(t, newBinary, sums, func([]byte) string {
		return signer([]byte("tampered-checksums\n"))
	})

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, UpdateApplyCodeSignatureMismatch)
	requireBinaryUntouched(t, executable)
}

// 老 Release 没有 SHA256SUMS.sig：按设计拒绝替换，不降级放行。
func TestApplyUpdateRejectsMissingSignatureAsset(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	applier, _, executable := newTestApplierWithSignature(t, newBinary, sums, nil)

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, UpdateApplyCodeSignatureAssetMissing)
	requireBinaryUntouched(t, executable)
}

// 签名资产缺失是独立错误码：HTTP 502，且日志必须能指出缺的是哪个资产；
// 同时不得记录签名字节或公钥内容。
// 说明：资产名刻意不套 MaskSensitiveInfo——实测它把 "SHA256SUMS.sig" 打成 "***.sig"，会抹掉诊断价值。
func TestApplyUpdateMissingSignatureAssetReportsCodeAndLogsAssetName(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	applier, _, executable := newTestApplierWithSignature(t, newBinary, sums, nil)

	var logged strings.Builder
	previousErrorWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logged
	t.Cleanup(func() { gin.DefaultErrorWriter = previousErrorWriter })

	_, err := applier.Apply(context.Background())

	var applyErr *UpdateApplyError
	require.ErrorAs(t, err, &applyErr)
	require.Equal(t, UpdateApplyCodeSignatureAssetMissing, applyErr.Code)
	require.Equal(t, http.StatusBadGateway, applyErr.HTTPStatus)
	require.Contains(t, logged.String(), UpdateAssetNameSHA256SumsSig, "日志必须能指出缺的是哪个资产")
	require.NotContains(t, logged.String(), "public key", "日志不得记录公钥内容")
	require.NotContains(t, logged.String(), "signature does not match", "日志不得记录验签细节")
	require.NotContains(t, logged.String(), updateSigningPublicKeys[0], "日志不得回显内嵌公钥")
	requireBinaryUntouched(t, executable)
}

// TestApplyUpdateSeesSignatureAssetFromRealFetcher 是本缺陷的核心回归（fetcher → applier 直通）：
// 既有的 apply 用例都注入 fakeReleaseFetcher，绕开了 fetcher 的资产处理，才让「裁剪掉 .sig」漏网。
// 这里用本地 httptest 起一个 GitHub 风格 Release（含 SHA256SUMS.sig），经**真实 fetcher** 解析后
// 交给 applier：断言 applier 确实拿到了签名资产并进入验签环节（故意给错误签名 → signature_mismatch）。
// 关键判据是「applier 真的去下载了 .sig 资产」——修复前 fetcher 会把它裁掉，applier 连请求都不会发。
func TestApplyUpdateSeesSignatureAssetFromRealFetcher(t *testing.T) {
	setUpdateSetting(t, true, true)
	signMessage := setTestSigningKey(t)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"

	// 先声明再赋值：handler 里的闭包要引用 server.URL。
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/owner/repo/releases/latest":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"tag_name":%q,"html_url":%q,"published_at":"2026-01-02T03:04:05Z","prerelease":false,"draft":false,"assets":[`+
				`{"name":%q,"browser_download_url":%q,"size":100},`+
				`{"name":%q,"browser_download_url":%q,"size":128},`+
				`{"name":%q,"browser_download_url":%q,"size":2048}]}`,
				testTag,
				server.URL+"/releases/"+testTag,
				UpdateAssetNameSHA256Sums, server.URL+"/assets/SHA256SUMS",
				UpdateAssetNameSHA256SumsSig, server.URL+"/assets/SHA256SUMS.sig",
				UpdateAssetNameForArch("amd64"), server.URL+"/assets/new-api-linux-amd64",
			)
		case "/assets/SHA256SUMS":
			_, _ = io.WriteString(writer, sums)
		case "/assets/SHA256SUMS.sig":
			// 签名有效，但覆盖的是另一份内容 → 必须停在验签失败（T3），而不是资产缺失（T1）。
			_, _ = io.WriteString(writer, signMessage([]byte("tampered-checksums\n")))
		case "/assets/new-api-linux-amd64":
			_, _ = writer.Write(newBinary)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 真实 fetcher 打本地 server；下载器复用同一客户端（已对私有 IP/回环显式放行）并记录请求。
	httpClient := *server.Client()
	httpClient.Transport = &http.Transport{Proxy: nil, DialContext: serverDialContext}
	recorder := &recordingHTTPDoer{inner: &httpClient}
	fetcher := &githubReleaseFetcher{client: &httpClient, repo: "owner/repo", baseURL: server.URL}

	executable := filepath.Join(t.TempDir(), "new-api")
	require.NoError(t, os.WriteFile(executable, []byte("old-binary"), 0o755))
	applier := newUpdateApplier(fetcher, recorder)
	applier.currentExecutable = func() (string, error) { return executable, nil }
	applier.goos = "linux"
	applier.goarch = "amd64"

	_, err := applier.Apply(context.Background())

	// 修复前：fetcher 把 .sig 裁掉 → applier 在资产检查处直接返回，根本不会请求签名资产。
	require.Contains(t, recorder.urls(), server.URL+"/assets/SHA256SUMS.sig",
		"applier 必须拿到并请求签名资产；没有该请求说明 fetcher 仍在裁剪资产")
	var applyErr *UpdateApplyError
	require.ErrorAs(t, err, &applyErr)
	require.Equal(t, UpdateApplyCodeSignatureMismatch, applyErr.Code,
		"越过资产检查后应停在验签失败")
	require.Equal(t, http.StatusBadGateway, applyErr.HTTPStatus)
	requireBinaryUntouched(t, executable)
}

// recordingHTTPDoer 记录下载请求的 URL，用于断言 applier 实际走到了哪一步。
type recordingHTTPDoer struct {
	inner httpDoer
	mu    sync.Mutex
	seen  []string
}

func (d *recordingHTTPDoer) Do(request *http.Request) (*http.Response, error) {
	d.mu.Lock()
	d.seen = append(d.seen, request.URL.String())
	d.mu.Unlock()
	return d.inner.Do(request)
}

func (d *recordingHTTPDoer) urls() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.seen...)
}

// serverDialContext 只放行回环地址：让 httptest 客户端能连本地 server，同时杜绝测试碰外部网络。
func serverDialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("refusing non-loopback dial to %s", address)
	}
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

// .sig 不是 64 字节 base64：按 signature_mismatch 处理。
func TestApplyUpdateRejectsMalformedSignature(t *testing.T) {
	setUpdateSetting(t, true, true)
	newBinary := []byte("new-binary-content")
	sums := sha256Hex(newBinary) + "  " + UpdateAssetNameForArch("amd64") + "\n"
	applier, _, executable := newTestApplierWithSignature(t, newBinary, sums, func([]byte) string {
		return "not-a-signature\n"
	})

	_, err := applier.Apply(context.Background())

	requireUpdateApplyError(t, err, UpdateApplyCodeSignatureMismatch)
	requireBinaryUntouched(t, executable)
}

func TestDecodeSignatureFormats(t *testing.T) {
	raw := make([]byte, ed25519.SignatureSize)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	standard := base64.StdEncoding.EncodeToString(raw)

	for _, testCase := range []struct {
		name    string
		encoded string
		wantOK  bool
	}{
		{name: "standard base64 with padding and newline", encoded: standard + "\n", wantOK: true},
		{name: "standard base64 with surrounding spaces", encoded: "  " + standard + "  ", wantOK: true},
		{name: "hex is tolerated", encoded: hex.EncodeToString(raw), wantOK: true},
		{name: "empty", encoded: "", wantOK: false},
		{name: "wrong length", encoded: base64.StdEncoding.EncodeToString(raw[:32]), wantOK: false},
		{name: "not base64", encoded: "!!!not-base64!!!", wantOK: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			decoded, err := decodeSignature(testCase.encoded)
			if !testCase.wantOK {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, raw, decoded)
		})
	}
}

func TestVerifySHA256SumsSignatureRejectsEmptyKeyList(t *testing.T) {
	previousKeys := updateSigningPublicKeys
	updateSigningPublicKeys = nil
	t.Cleanup(func() { updateSigningPublicKeys = previousKeys })

	// 直接生成一对密钥：这里只验证"内嵌公钥为空 → 拒绝"，不能再把公钥写回列表。
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	message := []byte("checksums")
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, message))
	require.Error(t, verifySHA256SumsSignature(message, []byte(signature)))
}

// --- 回滚点与回滚 -------------------------------------------------------------

func TestRollbackRestoresPreviousBinary(t *testing.T) {
	setUpdateSetting(t, true, true)
	applier, executable := newRollbackTestApplier(t)

	previousBinary := append([]byte("\x7fELF"), []byte("previous-binary-content")...)
	require.NoError(t, os.WriteFile(executable+updatePreviousBinarySuffix, previousBinary, 0o755))
	currentBefore := fileSHA256(t, executable)

	result, err := applier.Rollback()

	require.NoError(t, err)
	require.True(t, result.RolledBack)
	require.False(t, result.Persistent)
	require.NotEmpty(t, result.Hint)
	require.Equal(t, executable, result.ExecutablePath)

	restored, err := os.ReadFile(executable)
	require.NoError(t, err)
	require.Equal(t, previousBinary, restored, "回滚后当前二进制应等于回滚点内容")
	require.NotEqual(t, currentBefore, fileSHA256(t, executable))
	_, err = os.Stat(executable + updatePreviousBinarySuffix)
	require.True(t, errors.Is(err, os.ErrNotExist), "回滚点会被消费")
}

func TestRollbackWithoutRollbackPointFails(t *testing.T) {
	setUpdateSetting(t, true, true)
	applier, executable := newRollbackTestApplier(t)
	before := fileSHA256(t, executable)

	_, err := applier.Rollback()

	requireUpdateApplyError(t, err, UpdateApplyCodeNoRollbackPoint)
	require.Equal(t, before, fileSHA256(t, executable))
}

func TestRollbackRejectsNonExecutableRollbackPoint(t *testing.T) {
	setUpdateSetting(t, true, true)
	applier, executable := newRollbackTestApplier(t)
	require.NoError(t, os.WriteFile(executable+updatePreviousBinarySuffix, []byte("not-an-elf"), 0o755))
	before := fileSHA256(t, executable)

	_, err := applier.Rollback()

	requireUpdateApplyError(t, err, UpdateApplyCodeRollbackPointInvalid)
	require.Equal(t, before, fileSHA256(t, executable), "回滚失败必须原样保留当前二进制")
	_, err = os.Stat(executable + updatePreviousBinarySuffix)
	require.NoError(t, err, "判定失败时回滚点不应被消费")
}

func TestRollbackDisabledDoesNotTouchFiles(t *testing.T) {
	setUpdateSetting(t, true, false)
	applier, executable := newRollbackTestApplier(t)
	previousBinary := append([]byte("\x7fELF"), []byte("previous")...)
	require.NoError(t, os.WriteFile(executable+updatePreviousBinarySuffix, previousBinary, 0o755))
	before := fileSHA256(t, executable)

	_, err := applier.Rollback()

	requireUpdateApplyError(t, err, "disabled")
	require.Equal(t, before, fileSHA256(t, executable))
	_, err = os.Stat(executable + updatePreviousBinarySuffix)
	require.NoError(t, err)
}
