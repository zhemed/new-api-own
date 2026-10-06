package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/require"
)

// fakeReleaseFetcher 是纯内存数据源：测试中不建立任何连接。
type fakeReleaseFetcher struct {
	calls          int
	release        *ReleaseInfo
	err            error
	blockOnContext bool
}

func (f *fakeReleaseFetcher) FetchLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	f.calls++
	if f.blockOnContext {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.release, nil
}

// setUpdateSetting 走生产同款的选项更新路径切换开关（同时验证"改完即时生效"）。
func setUpdateSetting(t *testing.T, checkEnabled bool, applyEnabled bool) {
	t.Helper()
	configObject := config.GlobalConfig.Get("update_check_setting")
	require.NotNil(t, configObject)
	require.NoError(t, config.UpdateConfigFromMap(configObject, map[string]string{
		"enabled":       strconv.FormatBool(checkEnabled),
		"apply_enabled": strconv.FormatBool(applyEnabled),
	}))
	t.Cleanup(func() {
		_ = config.UpdateConfigFromMap(configObject, map[string]string{
			"enabled":       "false",
			"apply_enabled": "true",
		})
	})
}

// installFakeUpdateChecker 把包级单例替换为注入的假数据源，并保持替换（测试二进制内
// 永不回落到真实更新源）。
func installFakeUpdateChecker(fetcher ReleaseFetcher) *UpdateChecker {
	checker := newUpdateChecker(fetcher)
	updateChecker = checker
	updateCheckerOnce.Do(func() {})
	return checker
}

func TestGetUpdateStatusDisabledDoesNotFetch(t *testing.T) {
	setUpdateSetting(t, false, true)
	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{TagName: "v9.9.9"}}
	installFakeUpdateChecker(fetcher)

	status := GetUpdateStatus(context.Background())

	require.False(t, status.Enabled)
	require.Equal(t, UpdateCheckStateDisabled, status.State)
	require.Equal(t, common.Version, status.CurrentVersion)
	require.Nil(t, status.HasUpdate)
	require.Equal(t, 0, fetcher.calls, "关闭状态绝不能外呼")
}

func TestUpdateCheckerReportsUpdateAndCachesResult(t *testing.T) {
	setUpdateSetting(t, true, true)
	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{
		TagName:     "v9.9.9",
		HTMLURL:     "https://example.test/releases/v9.9.9",
		PublishedAt: "2026-01-02T03:04:05Z",
	}}
	checker := installFakeUpdateChecker(fetcher)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	checker.now = func() time.Time { return now }

	status := GetUpdateStatus(context.Background())
	require.True(t, status.Enabled)
	require.Equal(t, UpdateCheckStateUpdateAvailable, status.State)
	require.Equal(t, "v9.9.9", status.LatestVersion)
	require.Equal(t, "https://example.test/releases/v9.9.9", status.ReleaseURL)
	require.Equal(t, "2026-01-02T03:04:05Z", status.PublishedAt)
	require.NotNil(t, status.HasUpdate)
	require.True(t, *status.HasUpdate)
	require.Equal(t, 1, fetcher.calls)

	// 缓存命中：TTL 内不再外呼。
	now = now.Add(setting.UpdateCheckCacheTTL / 2)
	require.Equal(t, UpdateCheckStateUpdateAvailable, GetUpdateStatus(context.Background()).State)
	require.Equal(t, 1, fetcher.calls)

	// 超过 TTL 后重新检查。
	now = now.Add(setting.UpdateCheckCacheTTL)
	GetUpdateStatus(context.Background())
	require.Equal(t, 2, fetcher.calls)
}

// TestGetUpdateStatusReportsBothSwitches 覆盖三种开关组合，并保证前端不需要靠 /api/option/ 猜状态：
// 检查关（应用仍可开）/ 检查开+应用关 / 两者都开。
func TestGetUpdateStatusReportsBothSwitches(t *testing.T) {
	tests := []struct {
		name          string
		checkEnabled  bool
		applyEnabled  bool
		wantState     string
		wantHasUpdate *bool
	}{
		{
			name:         "check off, apply still allowed",
			checkEnabled: false,
			applyEnabled: true,
			wantState:    UpdateCheckStateDisabled,
		},
		{
			name:          "check on, apply off",
			checkEnabled:  true,
			applyEnabled:  false,
			wantState:     UpdateCheckStateUpdateAvailable,
			wantHasUpdate: common.GetPointer(true),
		},
		{
			name:          "both on",
			checkEnabled:  true,
			applyEnabled:  true,
			wantState:     UpdateCheckStateUpdateAvailable,
			wantHasUpdate: common.GetPointer(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setUpdateSetting(t, tt.checkEnabled, tt.applyEnabled)
			fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{TagName: "v9.9.9"}}
			installFakeUpdateChecker(fetcher)

			status := GetUpdateStatus(context.Background())

			require.Equal(t, tt.checkEnabled, status.CheckEnabled)
			require.Equal(t, tt.applyEnabled, status.ApplyEnabled)
			require.Equal(t, tt.checkEnabled, status.Enabled, "enabled 是 check_enabled 的兼容别名")
			require.Equal(t, tt.wantState, status.State)
			require.Equal(t, tt.wantHasUpdate, status.HasUpdate)
			if tt.checkEnabled {
				require.Equal(t, 1, fetcher.calls)
			} else {
				require.Equal(t, 0, fetcher.calls, "检查关闭时不得外呼")
			}
		})
	}
}

// TestSanitizePublicURL 断言对外字段不回显凭据/令牌。
func TestSanitizePublicURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "userinfo is stripped",
			raw:  "https://user:secret-token@mirror.test/releases/v9.9.9",
			want: "https://mirror.test/releases/v9.9.9",
		},
		{
			name: "plain https url is kept",
			raw:  "https://github.com/owner/repo/releases/tag/v9.9.9",
			want: "https://github.com/owner/repo/releases/tag/v9.9.9",
		},
		{name: "non http scheme is dropped", raw: "javascript:alert(1)", want: ""},
		{name: "plain text is dropped", raw: "not a url", want: ""},
		{name: "empty stays empty", raw: "  ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, sanitizePublicURL(tt.raw))
		})
	}
}

func TestUpdateCheckerStripsCredentialsFromReleaseURL(t *testing.T) {
	setUpdateSetting(t, true, true)
	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{
		TagName: "v9.9.9",
		HTMLURL: "https://user:secret-token@mirror.test/releases/v9.9.9",
	}}
	installFakeUpdateChecker(fetcher)

	status := GetUpdateStatus(context.Background())

	require.Equal(t, "https://mirror.test/releases/v9.9.9", status.ReleaseURL)
	require.NotContains(t, status.ReleaseURL, "secret-token")
	require.NotContains(t, status.ReleaseURL, "user:")
}

func TestUpdateCheckerReportsUpToDateForSameVersion(t *testing.T) {
	setUpdateSetting(t, true, true)
	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{TagName: common.Version}}
	installFakeUpdateChecker(fetcher)

	status := GetUpdateStatus(context.Background())

	require.Equal(t, UpdateCheckStateUpToDate, status.State)
	require.NotNil(t, status.HasUpdate)
	require.False(t, *status.HasUpdate)
}

func TestUpdateCheckerFailureIsUnknownAndBacksOff(t *testing.T) {
	setUpdateSetting(t, true, true)
	fetcher := &fakeReleaseFetcher{err: errors.New("dial tcp: connection refused")}
	checker := installFakeUpdateChecker(fetcher)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	checker.now = func() time.Time { return now }

	status := GetUpdateStatus(context.Background())
	require.Equal(t, UpdateCheckStateUnknown, status.State)
	require.Nil(t, status.HasUpdate, "查不到不能当成已是最新")
	require.Empty(t, status.LatestVersion)
	require.Equal(t, 1, fetcher.calls)

	// 退避窗口内不再重试。
	now = now.Add(setting.UpdateCheckFailureBackoff / 2)
	require.Equal(t, UpdateCheckStateUnknown, GetUpdateStatus(context.Background()).State)
	require.Equal(t, 1, fetcher.calls)

	// 退避结束后恢复检查；这次成功后状态变正常。
	now = now.Add(setting.UpdateCheckFailureBackoff)
	fetcher.err = nil
	fetcher.release = &ReleaseInfo{TagName: "v9.9.9"}
	require.Equal(t, UpdateCheckStateUpdateAvailable, GetUpdateStatus(context.Background()).State)
	require.Equal(t, 2, fetcher.calls)
}

func TestUpdateCheckerTimeoutDegradesToUnknown(t *testing.T) {
	setUpdateSetting(t, true, true)
	fetcher := &fakeReleaseFetcher{blockOnContext: true}
	checker := installFakeUpdateChecker(fetcher)
	checker.timeout = 20 * time.Millisecond

	status := GetUpdateStatus(context.Background())

	require.Equal(t, UpdateCheckStateUnknown, status.State)
	require.Nil(t, status.HasUpdate)
	require.Equal(t, 1, fetcher.calls)
}

// TestDescribeUpdateSourceStripsCredentials 保证启动日志里不会出现任何凭据/令牌。
func TestDescribeUpdateSourceStripsCredentials(t *testing.T) {
	previousBaseURL := setting.UpdateSourceApiBaseUrl
	previousRepository := setting.UpdateSourceRepository
	setting.UpdateSourceApiBaseUrl = "https://user:secret-token@mirror.test/api/github/"
	setting.UpdateSourceRepository = "owner/repo"
	t.Cleanup(func() {
		setting.UpdateSourceApiBaseUrl = previousBaseURL
		setting.UpdateSourceRepository = previousRepository
	})

	description := describeUpdateSource()

	require.Contains(t, description, "mirror.test")
	require.Contains(t, description, "owner/repo/releases/latest")
	require.NotContains(t, description, "secret-token")
	require.NotContains(t, description, "user:")
}

func TestUpdateCheckerKeepsLastKnownReleaseAfterFailure(t *testing.T) {
	setUpdateSetting(t, true, true)
	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{TagName: "v9.9.9"}}
	checker := installFakeUpdateChecker(fetcher)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	checker.now = func() time.Time { return now }

	require.Equal(t, UpdateCheckStateUpdateAvailable, GetUpdateStatus(context.Background()).State)

	// 过期后再查失败：保留上次已知版本，不把"查不到"说成"已是最新"。
	now = now.Add(2 * setting.UpdateCheckCacheTTL)
	fetcher.err = errors.New("dial tcp: i/o timeout")
	status := GetUpdateStatus(context.Background())
	require.Equal(t, UpdateCheckStateUpdateAvailable, status.State)
	require.Equal(t, "v9.9.9", status.LatestVersion)
	require.NotNil(t, status.HasUpdate)
	require.True(t, *status.HasUpdate)
	require.Equal(t, 2, fetcher.calls)
}

func TestUpdateCheckerNonVersionTagIsUnknown(t *testing.T) {
	setUpdateSetting(t, true, true)
	fetcher := &fakeReleaseFetcher{release: &ReleaseInfo{TagName: "release-latest"}}
	installFakeUpdateChecker(fetcher)

	status := GetUpdateStatus(context.Background())

	require.Equal(t, UpdateCheckStateUnknown, status.State)
	require.Nil(t, status.HasUpdate, "非版本号不得谎报有更新")
	require.Empty(t, status.LatestVersion)
}

// fakeHTTPDoer 把 HTTP 调用替换为内存响应。
type fakeHTTPDoer struct {
	respond  func(req *http.Request) (*http.Response, error)
	requests []*http.Request
}

func (d *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	d.requests = append(d.requests, req)
	return d.respond(req)
}

func newFakeHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}

const stableReleasePayload = `{
  "tag_name": "v0.0.7",
  "html_url": "https://example.test/releases/v0.0.7",
  "published_at": "2026-01-02T03:04:05Z",
  "prerelease": false,
  "draft": false,
  "body": "release notes",
  "assets": [
    {"name": "SHA256SUMS", "browser_download_url": "https://example.test/SHA256SUMS", "size": 100},
    {"name": "new-api-linux-amd64", "browser_download_url": "https://example.test/new-api-linux-amd64", "size": 2048},
    {"name": "notes.txt", "browser_download_url": "https://example.test/notes.txt", "size": 10}
  ]
}`

func TestGithubReleaseFetcherParsesStableRelease(t *testing.T) {
	doer := &fakeHTTPDoer{respond: func(req *http.Request) (*http.Response, error) {
		return newFakeHTTPResponse(http.StatusOK, stableReleasePayload), nil
	}}
	fetcher := &githubReleaseFetcher{client: doer, repo: "owner/repo"}

	release, err := fetcher.FetchLatestRelease(context.Background())

	require.NoError(t, err)
	require.Equal(t, "v0.0.7", release.TagName)
	require.Equal(t, "https://example.test/releases/v0.0.7", release.HTMLURL)
	require.Len(t, doer.requests, 1)
	require.Equal(t, "https://api.github.com/repos/owner/repo/releases/latest", doer.requests[0].URL.String())
	require.Equal(t, "application/vnd.github+json", doer.requests[0].Header.Get("Accept"))
	require.NotEmpty(t, doer.requests[0].Header.Get("User-Agent"), "GitHub 强制要求 User-Agent")
	// 只保留自更新需要的资产。
	require.Len(t, release.Assets, 2)
	require.Equal(t, UpdateAssetNameSHA256Sums, release.Assets[0].Name)
	require.Equal(t, UpdateAssetNameForArch("amd64"), release.Assets[1].Name)
}

func TestGithubReleaseFetcherHonorsBaseURLOverride(t *testing.T) {
	doer := &fakeHTTPDoer{respond: func(req *http.Request) (*http.Response, error) {
		return newFakeHTTPResponse(http.StatusOK, stableReleasePayload), nil
	}}
	fetcher := &githubReleaseFetcher{client: doer, repo: "owner/repo", baseURL: "https://mirror.test/api/github/"}

	_, err := fetcher.FetchLatestRelease(context.Background())

	require.NoError(t, err)
	require.Equal(t, "https://mirror.test/api/github/repos/owner/repo/releases/latest", doer.requests[0].URL.String())
}

func TestGithubReleaseFetcherRejectsPrereleaseAndErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		doerErr    error
		wantErrMsg string
	}{
		{
			name:       "prerelease flag is respected",
			status:     http.StatusOK,
			body:       `{"tag_name":"v9.9.9-rc1","prerelease":true}`,
			wantErrMsg: "no stable release",
		},
		{
			name:       "draft flag is respected",
			status:     http.StatusOK,
			body:       `{"tag_name":"v9.9.9","draft":true}`,
			wantErrMsg: "no stable release",
		},
		{
			name:       "missing tag is rejected",
			status:     http.StatusOK,
			body:       `{"html_url":"https://example.test"}`,
			wantErrMsg: "no release tag",
		},
		{
			name:       "non 200 status is rejected",
			status:     http.StatusNotFound,
			body:       `{}`,
			wantErrMsg: "status 404",
		},
		{
			name:       "transport error is propagated",
			doerErr:    errors.New("dial tcp: connection refused"),
			wantErrMsg: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doer := &fakeHTTPDoer{respond: func(req *http.Request) (*http.Response, error) {
				if tt.doerErr != nil {
					return nil, tt.doerErr
				}
				return newFakeHTTPResponse(tt.status, tt.body), nil
			}}
			fetcher := &githubReleaseFetcher{client: doer, repo: "owner/repo"}

			_, err := fetcher.FetchLatestRelease(context.Background())

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}
