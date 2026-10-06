package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

const (
	testSumsURL   = "https://example.test/SHA256SUMS"
	testBinaryURL = "https://example.test/new-api-linux-amd64"
	testTag       = "v9.9.9"
)

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// newTestApplier 构造完全离线的 applier：假数据源 + 假下载器 + 临时目录里的"当前二进制"。
func newTestApplier(t *testing.T, newBinary []byte, sumsContent string) (*UpdateApplier, *fakeReleaseFetcher, string) {
	t.Helper()
	executable := filepath.Join(t.TempDir(), "new-api")
	require.NoError(t, os.WriteFile(executable, []byte("old-binary"), 0o755))

	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{
		TagName: testTag,
		Assets: []ReleaseAsset{
			{Name: UpdateAssetNameSHA256Sums, DownloadURL: testSumsURL},
			{Name: UpdateAssetNameForArch("amd64"), DownloadURL: testBinaryURL},
		},
	}}
	doer := &fakeHTTPDoer{respond: func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case testSumsURL:
			return newFakeHTTPResponse(http.StatusOK, sumsContent), nil
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
	require.Equal(t, 1, fetcher.calls)

	content, err := os.ReadFile(executable)
	require.NoError(t, err)
	require.Equal(t, newBinary, content, "替换后文件内容应为已校验的新二进制")
	info, err := os.Stat(executable)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "新二进制必须可执行")

	entries, err := os.ReadDir(filepath.Dir(executable))
	require.NoError(t, err)
	require.Len(t, entries, 1, "替换完成后不得残留临时文件")
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
	applier, _, executable := newTestApplier(t, newBinary, sums)
	applier.downloader = &fakeHTTPDoer{respond: func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == testBinaryURL {
			return nil, errors.New("connection reset by peer")
		}
		return newFakeHTTPResponse(http.StatusOK, sums), nil
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
