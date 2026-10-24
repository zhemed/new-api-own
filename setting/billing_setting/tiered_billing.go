package billing_setting

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/samber/lo"
)

const (
	BillingModeRatio      = "ratio"
	BillingModeTieredExpr = "tiered_expr"
	BillingModeField      = "billing_mode"
	BillingExprField      = "billing_expr"
)

// BillingSetting is managed by config.GlobalConfig.Register.
// DB keys: billing_setting.billing_mode, billing_setting.billing_expr
type BillingSetting struct {
	BillingMode map[string]string `json:"billing_mode"`
	BillingExpr map[string]string `json:"billing_expr"`
}

// defaultDeepSeekPeakOffPeakExpr 是 deepseek 家族的默认峰谷定价表达式（2026-10-22 官方人民币价）。
// 计价语义（详见 pkg/billingexpr/expr.md）：
//   - 系数是"每 1M tokens 的价格"（文档写作 $/1M），换算 quota 用
//     quota = 系数 × tokens / 1_000_000 × QuotaPerUnit × 分组倍率
//     （relay/helper/price.go 的 tiered 分支）；
//   - 显示值（每 1M）= 系数 × 显示汇率。因此下面的 ¥1.00 / ¥4.00 / ¥0.02 只在
//     "显示汇率 = 1.0"（当前默认：CUSTOM + ¥ + 1.0，见 setting/operation_setting/general_setting.go）
//     时与官方人民币价一一对应；若把显示汇率改成 7.3，同一表达式显示会放大 7.3 倍。
//   - 高峰窗口按官方说明取北京时间工作日 09:00-12:00 与 14:00-18:00，缓存命中/未命中/输出全部 ×2。
//   - 低谷：p*1（未命中 ¥1.00/M）、c*4（输出 ¥4.00/M）、cr*0.02（命中 ¥0.02/M）。
const defaultDeepSeekPeakOffPeakExpr = `(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))) ? tier("peak", p*2 + c*8 + cr*0.04) : tier("offpeak", p*1 + c*4 + cr*0.02)`

// defaultTieredBillingModels 默认启用峰谷表达式计费的模型，与 defaultModelRatio 的默认模板一致。
var defaultTieredBillingModels = []string{"deepseek-flash", "deepseek-v4.1-flash"}

func defaultBillingSetting() BillingSetting {
	modes := make(map[string]string, len(defaultTieredBillingModels))
	exprs := make(map[string]string, len(defaultTieredBillingModels))
	for _, model := range defaultTieredBillingModels {
		modes[model] = BillingModeTieredExpr
		exprs[model] = defaultDeepSeekPeakOffPeakExpr
	}
	return BillingSetting{BillingMode: modes, BillingExpr: exprs}
}

// DefaultBillingMode2JSONString 返回计费模式的**代码默认** JSON（两个默认模板均为 tiered_expr）。
// 供「重置定价」恢复出厂值使用，避免在别处复制默认值字面量。
func DefaultBillingMode2JSONString() string {
	return marshalDefault(defaultBillingSetting().BillingMode, BillingModeField)
}

// DefaultBillingExpr2JSONString 返回计费表达式的代码默认 JSON（两个默认模板逐字符相同）。
func DefaultBillingExpr2JSONString() string {
	return marshalDefault(defaultBillingSetting().BillingExpr, BillingExprField)
}

func marshalDefault(value map[string]string, field string) string {
	jsonBytes, err := common.Marshal(value)
	if err != nil {
		common.SysError("error marshalling default " + field + ": " + err.Error())
	}
	return string(jsonBytes)
}

var billingSetting = defaultBillingSetting()

func init() {
	config.GlobalConfig.Register("billing_setting", &billingSetting)
}

// ---------------------------------------------------------------------------
// Read accessors (hot path, must be fast)
// ---------------------------------------------------------------------------

func GetBillingMode(model string) string {
	if mode, ok := billingSetting.BillingMode[model]; ok {
		return mode
	}
	return BillingModeRatio
}

func GetBillingExpr(model string) (string, bool) {
	expr, ok := billingSetting.BillingExpr[model]
	return expr, ok
}

func GetBillingModeCopy() map[string]string {
	return lo.Assign(billingSetting.BillingMode)
}

func GetBillingExprCopy() map[string]string {
	return lo.Assign(billingSetting.BillingExpr)
}

func GetPricingSyncData(base map[string]any) map[string]any {
	extra := make(map[string]any, 2)
	if modes := GetBillingModeCopy(); len(modes) > 0 {
		extra[BillingModeField] = modes
	}
	if exprs := GetBillingExprCopy(); len(exprs) > 0 {
		extra[BillingExprField] = exprs
	}
	return lo.Assign(base, extra)
}

// ---------------------------------------------------------------------------
// Smoke test (called externally for validation before save)
// ---------------------------------------------------------------------------

func SmokeTestExpr(exprStr string) error {
	return smokeTestExpr(exprStr)
}

func smokeTestExpr(exprStr string) error {
	vectors := []billingexpr.TokenParams{
		{P: 0, C: 0, Len: 0},
		{P: 1000, C: 1000, Len: 1000},
		{P: 100000, C: 100000, Len: 100000},
		{P: 1000000, C: 1000000, Len: 1000000},
	}
	requests := []billingexpr.RequestInput{
		{},
		{
			Headers: map[string]string{
				"anthropic-beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast","stream_options":{"include_usage":true},"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	}

	for _, v := range vectors {
		for _, request := range requests {
			result, _, err := billingexpr.RunExprWithRequest(exprStr, v, request)
			if err != nil {
				return fmt.Errorf("vector {p=%g, c=%g}: run failed: %w", v.P, v.C, err)
			}
			if result < 0 {
				return fmt.Errorf("vector {p=%g, c=%g}: result %f < 0", v.P, v.C, result)
			}
		}
	}
	return nil
}
