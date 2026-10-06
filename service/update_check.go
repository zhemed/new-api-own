package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/bytedance/gopkg/util/gopool"
)

// 更新检查对外的粗粒度状态（state 字段）。
// 失败/超时/版本不可比一律归入 unknown：UI 中性显示，不报错也不谎报"有更新/已最新"。
const (
	UpdateCheckStateDisabled        = "disabled"         // 功能未开启
	UpdateCheckStateUnknown         = "unknown"          // 无可用结论（尚未检查/检查失败/版本不可比）
	UpdateCheckStateUpToDate        = "up_to_date"       // 已是最新
	UpdateCheckStateUpdateAvailable = "update_available" // 有新版本
)

const (
	// updateCheckMaxBodyBytes 限制元数据读取上限：只解析必要字段，不拉整份 release 正文。
	updateCheckMaxBodyBytes = 512 << 10
	updateCheckUserAgent    = "new-api-update-check"

	// UpdateAssetNameSHA256Sums 是更新源必须提供的校验和资产名。
	UpdateAssetNameSHA256Sums = "SHA256SUMS"
)

// UpdateAssetNameForArch 返回本机架构对应的二进制资产名（Release 资产命名约定）。
func UpdateAssetNameForArch(goarch string) string {
	return "new-api-linux-" + goarch
}

// errNoStableRelease 表示更新源当前没有正式（非预发布、非草稿）版本。
var errNoStableRelease = errors.New("update source has no stable release")

// ReleaseAsset 是 Release 资产（只保留自更新需要的少量字段）。
type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// ReleaseInfo 是更新源返回的最少字段集。
type ReleaseInfo struct {
	TagName     string         `json:"tag_name"`
	HTMLURL     string         `json:"html_url"`
	PublishedAt string         `json:"published_at"`
	Prerelease  bool           `json:"prerelease"`
	Draft       bool           `json:"draft"`
	Assets      []ReleaseAsset `json:"assets"`
}

// ReleaseFetcher 是更新源适配点：接入其它更新源（例如自建版本清单）只需实现该接口。
type ReleaseFetcher interface {
	FetchLatestRelease(ctx context.Context) (*ReleaseInfo, error)
}

// httpDoer 抽象 HTTP 调用，便于测试注入假响应（测试中不建立任何连接）。
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// UpdateCheckStatus 是对外暴露的更新检查结果。
//
// 开关字段语义（三者都直接来自服务端配置，前端不需要再猜）：
//   - CheckEnabled：更新**检查**是否开启；关闭时 state=disabled、其余结果字段保持中性。
//   - ApplyEnabled："**立即更新**"是否允许（下载并替换二进制）。它与 CheckEnabled 相互独立：
//     CheckEnabled=false + ApplyEnabled=true 时，检查结果不可用，但管理员仍可主动触发更新。
//   - Enabled：等价于 CheckEnabled 的兼容字段（旧调用方使用），新代码请用 CheckEnabled。
//
// HasUpdate 为三态：true=有更新、false=已最新、nil=未知（未开启/检查失败/版本无法比较）。
type UpdateCheckStatus struct {
	Enabled        bool   `json:"enabled"`
	CheckEnabled   bool   `json:"check_enabled"`
	ApplyEnabled   bool   `json:"apply_enabled"`
	State          string `json:"state"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      *bool  `json:"has_update"`
	ReleaseURL     string `json:"release_url,omitempty"`
	PublishedAt    string `json:"published_at,omitempty"`
	CheckedAt      int64  `json:"checked_at,omitempty"`
}

// UpdateChecker 负责更新检查的超时、缓存与失败退避；数据源由 ReleaseFetcher 注入。
type UpdateChecker struct {
	fetcher ReleaseFetcher
	now     func() time.Time
	timeout time.Duration
	ttl     time.Duration
	backoff time.Duration

	mu sync.Mutex
	// release 是最近一次成功结果；失败时保留，面板仍可展示上次已知版本。
	release   *ReleaseInfo
	checkedAt time.Time
	// nextAttempt 是下一次允许外呼的时间点（成功按 TTL、失败按退避窗口）。
	nextAttempt time.Time
}

func newUpdateChecker(fetcher ReleaseFetcher) *UpdateChecker {
	return &UpdateChecker{
		fetcher: fetcher,
		now:     time.Now,
		timeout: setting.UpdateCheckTimeout,
		ttl:     setting.UpdateCheckCacheTTL,
		backoff: setting.UpdateCheckFailureBackoff,
	}
}

// Check 返回更新状态：命中缓存或处于失败退避窗口内时直接返回上次结果，不外呼。
// 失败不会污染缓存，也不会把"查不到"当成"已是最新"：状态保持 unknown（HasUpdate 为 nil），
// 并按退避窗口节流，避免离线环境反复外呼。
func (c *UpdateChecker) Check(ctx context.Context) UpdateCheckStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if !c.nextAttempt.IsZero() && now.Before(c.nextAttempt) {
		return c.currentStatus()
	}

	fetchCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	release, err := c.fetcher.FetchLatestRelease(fetchCtx)
	if err != nil {
		c.nextAttempt = now.Add(c.backoff)
		// 只记录脱敏后的失败原因：代理地址/凭据不落日志。
		common.SysError(fmt.Sprintf("update check failed: %s", common.MaskSensitiveInfo(err.Error())))
		return c.currentStatus()
	}

	c.release = release
	c.checkedAt = now
	c.nextAttempt = now.Add(c.ttl)
	return c.currentStatus()
}

// currentStatus 由缓存结果生成对外状态；调用方需持有 c.mu。
func (c *UpdateChecker) currentStatus() UpdateCheckStatus {
	status := UpdateCheckStatus{
		Enabled:        true,
		CheckEnabled:   true,
		ApplyEnabled:   setting.IsUpdateApplyEnabled(),
		CurrentVersion: common.Version,
		HasUpdate:      nil,
	}

	if c.release == nil || !isComparableVersionTag(c.release.TagName) {
		// 无可用结论（尚未检查 / 更新源不可达 / 版本不可比）一律中性 unknown。
		status.State = UpdateCheckStateUnknown
		return status
	}

	status.LatestVersion = c.release.TagName
	// 只回显脱敏后的链接：即使更新源返回带 userinfo 的 URL，也不把凭据带给前端。
	status.ReleaseURL = sanitizePublicURL(c.release.HTMLURL)
	status.PublishedAt = c.release.PublishedAt
	status.CheckedAt = c.checkedAt.Unix()

	switch common.CompareVersions(common.Version, c.release.TagName) {
	case common.VersionComparisonNewer:
		status.State = UpdateCheckStateUpdateAvailable
		status.HasUpdate = common.GetPointer(true)
	case common.VersionComparisonEqual, common.VersionComparisonOlder:
		status.State = UpdateCheckStateUpToDate
		status.HasUpdate = common.GetPointer(false)
	default:
		// 版本不可比较：保持中性，绝不谎报"有更新"。
		status.State = UpdateCheckStateUnknown
	}
	return status
}

// isComparableVersionTag 判断 tag 归一化后是否是"以数字开头"的版本号；
// 避免更新源给出非版本文本时把状态误报成"有更新"。
func isComparableVersionTag(tag string) bool {
	normalized := common.NormalizeVersion(tag)
	if normalized == "" {
		return false
	}
	head := normalized[0]
	return head >= '0' && head <= '9'
}

// githubReleaseFetcher 是默认数据源：GitHub Releases。
// 使用 /releases/latest（GitHub 语义：最近一个非预发布、非草稿的正式版本），
// 并再次校验 prerelease/draft 标志——预发布过滤以 Release 标志为准，而不是靠版本号推导。
type githubReleaseFetcher struct {
	client  httpDoer
	repo    string
	baseURL string
}

func (f *githubReleaseFetcher) FetchLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	if strings.TrimSpace(f.repo) == "" {
		return nil, errors.New("update check repository is not configured")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, f.latestReleaseURL(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", updateCheckUserAgent)

	response, err := f.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update source returned status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, updateCheckMaxBodyBytes))
	if err != nil {
		return nil, err
	}
	release := &ReleaseInfo{}
	if err := common.Unmarshal(body, release); err != nil {
		return nil, err
	}
	if release.Draft || release.Prerelease {
		return nil, errNoStableRelease
	}
	if strings.TrimSpace(release.TagName) == "" {
		return nil, errors.New("update source returned no release tag")
	}
	release.Assets = filterUpdateAssets(release.Assets)
	return release, nil
}

func (f *githubReleaseFetcher) latestReleaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(f.baseURL), "/")
	if base == "" {
		base = "https://api.github.com"
	}
	return base + "/repos/" + f.repo + "/releases/latest"
}

// filterUpdateAssets 只保留自更新需要的资产，避免把整份资产列表带进内存与响应。
func filterUpdateAssets(assets []ReleaseAsset) []ReleaseAsset {
	kept := make([]ReleaseAsset, 0, 3)
	for _, asset := range assets {
		if asset.Name == UpdateAssetNameSHA256Sums ||
			asset.Name == UpdateAssetNameForArch("amd64") ||
			asset.Name == UpdateAssetNameForArch("arm64") {
			kept = append(kept, asset)
		}
	}
	return kept
}

// newUpdateSourceClient 构造访问更新源的客户端：克隆默认传输（保留 HTTPS_PROXY 等
// 环境变量行为），并按配置覆盖代理出口。
func newUpdateSourceClient(timeout time.Duration) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL := strings.TrimSpace(setting.UpdateSourceProxyUrl); proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil || parsed.Host == "" {
			// 不回显代理地址（可能内嵌凭据）。
			return nil, errors.New("update proxy url is invalid")
		}
		transport.Proxy = http.ProxyURL(parsed)
	}
	return &http.Client{Timeout: timeout, Transport: transport}, nil
}

// errorReleaseFetcher 让"客户端构造失败"也走正常的失败降级路径。
type errorReleaseFetcher struct{ err error }

func (f errorReleaseFetcher) FetchLatestRelease(context.Context) (*ReleaseInfo, error) {
	return nil, f.err
}

var (
	updateSourceOnce  sync.Once
	updateSource      ReleaseFetcher
	updateCheckerOnce sync.Once
	updateChecker     *UpdateChecker
)

// defaultUpdateSource 返回包级共享的数据源（含代理/base 配置）。
func defaultUpdateSource() ReleaseFetcher {
	updateSourceOnce.Do(func() {
		client, err := newUpdateSourceClient(setting.UpdateCheckTimeout)
		if err != nil {
			updateSource = errorReleaseFetcher{err: err}
			return
		}
		updateSource = &githubReleaseFetcher{
			client:  client,
			repo:    setting.UpdateSourceRepository,
			baseURL: setting.UpdateSourceApiBaseUrl,
		}
	})
	return updateSource
}

func defaultUpdateChecker() *UpdateChecker {
	updateCheckerOnce.Do(func() {
		updateChecker = newUpdateChecker(defaultUpdateSource())
	})
	return updateChecker
}

// GetUpdateStatus 返回更新检查结果。功能未开启时直接返回 disabled，绝不外呼。
// 开关字段（check_enabled / apply_enabled / enabled）始终如实回填，便于前端判断按钮可用性。
func GetUpdateStatus(ctx context.Context) UpdateCheckStatus {
	checkEnabled := setting.IsUpdateCheckEnabled()
	applyEnabled := setting.IsUpdateApplyEnabled()
	if !checkEnabled {
		return UpdateCheckStatus{
			Enabled:        false,
			CheckEnabled:   false,
			ApplyEnabled:   applyEnabled,
			State:          UpdateCheckStateDisabled,
			CurrentVersion: common.Version,
			HasUpdate:      nil,
		}
	}
	return defaultUpdateChecker().Check(ctx)
}

// sanitizePublicURL 剥掉 URL 里的 userinfo，保证对外响应/日志里不会回显任何凭据或令牌。
// 解析失败、非 http(s) 或缺 host 时返回空串：宁可不展示链接，也不回显可疑内容。
func sanitizePublicURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	parsed.User = nil
	return parsed.String()
}

// StartUpdateCheck 在服务启动阶段调用（router.SetRouter；此处日志已就绪）：
// 打一行 info 说明两个开关的默认行为、目标源与出口方式（**不含任何凭据**），
// 并在检查开启时后台预热一次，让面板首次打开就能拿到真实结果或中性 unknown。
// 预热是后台 goroutine：启动与请求路径都不会被外呼拖住。
func StartUpdateCheck() {
	egress := "direct"
	if strings.TrimSpace(setting.UpdateSourceProxyUrl) != "" {
		egress = "proxy"
	}
	common.SysLog(fmt.Sprintf(
		"update check: enabled=%t (default true), apply_enabled=%t (default true), source=%s, egress=%s, cache_ttl=%s, failure_backoff=%s",
		setting.IsUpdateCheckEnabled(), setting.IsUpdateApplyEnabled(),
		describeUpdateSource(), egress, setting.UpdateCheckCacheTTL, setting.UpdateCheckFailureBackoff,
	))

	if !setting.IsUpdateCheckEnabled() {
		return
	}
	gopool.Go(func() {
		// 后台预热绝不能让 panic 拖垮进程。
		defer func() {
			if recovered := recover(); recovered != nil {
				common.SysError(fmt.Sprintf("update check warm-up panic: %v", recovered))
			}
		}()
		GetUpdateStatus(context.Background())
	})
}

// describeUpdateSource 生成日志用的数据源描述：剥掉 URL 里的 userinfo，
// 保证日志里不会出现任何凭据/令牌。
func describeUpdateSource() string {
	base := strings.TrimRight(strings.TrimSpace(setting.UpdateSourceApiBaseUrl), "/")
	if parsed, err := url.Parse(base); err == nil {
		parsed.User = nil
		base = strings.TrimRight(parsed.String(), "/")
	}
	return base + "/repos/" + setting.UpdateSourceRepository + "/releases/latest"
}
