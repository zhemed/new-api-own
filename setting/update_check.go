package setting

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// UpdateCheckSetting 是更新检查与"立即更新"的可运行期开关。
// 注册到全局配置管理器后，面板/选项接口改完即时生效（无需重启），可随时立刻关掉外呼。
type UpdateCheckSetting struct {
	// Enabled 控制"检查更新"（面板查询 / 自动检查）。默认**开启**，使面板开箱即可显示
	// 版本与可用更新；置为 false（或 UPDATE_CHECK_ENABLED=false）可彻底关闭，关闭后服务端
	// 不发起任何外呼。离线/不可达时检查结果一律中性 unknown，不报错也不谎报，且有退避。
	Enabled bool `json:"enabled"`
	// ApplyEnabled 控制"立即更新"（下载并原地替换本机二进制）。默认**开启**，但该动作始终
	// 仅管理员可触发；置为 false 可整体关闭自更新能力。
	ApplyEnabled bool `json:"apply_enabled"`
}

var updateCheckSetting = UpdateCheckSetting{
	Enabled:      common.GetEnvOrDefaultBool("UPDATE_CHECK_ENABLED", true),
	ApplyEnabled: common.GetEnvOrDefaultBool("UPDATE_APPLY_ENABLED", true),
}

func init() {
	config.GlobalConfig.Register("update_check_setting", &updateCheckSetting)
}

// IsUpdateCheckEnabled 返回"检查更新"开关当前值。
func IsUpdateCheckEnabled() bool {
	return updateCheckSetting.Enabled
}

// IsUpdateApplyEnabled 返回"立即更新"开关当前值。
func IsUpdateApplyEnabled() bool {
	return updateCheckSetting.ApplyEnabled
}

// 更新源（默认 GitHub Releases；换成其它源时由 service 层的数据源适配点接管）。
var (
	// UpdateSourceRepository 更新源仓库（owner/repo）。
	UpdateSourceRepository = common.GetEnvOrDefaultString("UPDATE_CHECK_REPOSITORY", "zhemed/new-api-own")
	// UpdateSourceApiBaseUrl GitHub API 基址，可指向镜像/自建代理以适配受限网络。
	UpdateSourceApiBaseUrl = common.GetEnvOrDefaultString("UPDATE_CHECK_API_BASE_URL", "https://api.github.com")
	// UpdateSourceProxyUrl 可选 HTTP(S) 代理出口；不设置时沿用 Go 默认行为（HTTPS_PROXY 等）。
	UpdateSourceProxyUrl = common.GetEnvOrDefaultString("UPDATE_CHECK_PROXY_URL", "")
)

const (
	// UpdateCheckTimeout 单次元数据请求超时，避免请求卡住面板接口。
	UpdateCheckTimeout = 10 * time.Second
	// UpdateCheckCacheTTL 成功结果的缓存时长（≥1h），避免频繁外呼被上游限流。
	UpdateCheckCacheTTL = time.Hour
	// UpdateCheckFailureBackoff 失败后的退避时长，期间不再外呼。
	UpdateCheckFailureBackoff = 30 * time.Minute

	// UpdateApplyTimeout 二进制下载超时（体积较大，单独放宽）。
	UpdateApplyTimeout = 10 * time.Minute
	// UpdateApplyMaxBinaryBytes 下载体积上限，防止异常大响应写满磁盘。
	UpdateApplyMaxBinaryBytes = 256 << 20
	// UpdateApplyMaxChecksumBytes SHA256SUMS 体积上限。
	UpdateApplyMaxChecksumBytes = 64 << 10
)
