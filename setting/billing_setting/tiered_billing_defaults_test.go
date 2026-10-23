package billing_setting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 用户口径（2026-10-22）：新部署开箱即得 deepseek 官方人民币峰谷价，表达式逐字符固定。
// 改动这个字面量等于改价，必须是有意为之。
const expectedDeepSeekPeakOffPeakExpr = `(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))) ? tier("peak", p*2 + c*8 + cr*0.04) : tier("offpeak", p*1 + c*4 + cr*0.02)`

const deepSeekPeakCondition = `(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18)))`

func TestDefaultTieredBillingCoversDeepSeekFamily(t *testing.T) {
	for _, model := range []string{"deepseek-flash", "deepseek-v4.1-flash"} {
		t.Run(model, func(t *testing.T) {
			assert.Equal(t, BillingModeTieredExpr, GetBillingMode(model))

			expr, ok := GetBillingExpr(model)
			require.True(t, ok, "%s 必须带默认表达式", model)
			assert.Equal(t, expectedDeepSeekPeakOffPeakExpr, expr)
		})
	}

	// 默认模板只剩这两个模型；多留一个键必须是有意为之（与目录锁同口径）。
	require.Len(t, GetBillingModeCopy(), 2)
	require.Len(t, GetBillingExprCopy(), 2)

	// 目录外的模型不受影响，仍走比率计费且没有表达式。
	assert.Equal(t, BillingModeRatio, GetBillingMode("gpt-4o"))
	_, ok := GetBillingExpr("gpt-4o")
	assert.False(t, ok)
}

// 默认表达式必须能编译：编译只在请求期发生，写错不会让服务启动失败，
// 但会让该模型的每个请求都 400，所以这里兜住。
func TestDefaultDeepSeekExprCompiles(t *testing.T) {
	expr, ok := GetBillingExpr("deepseek-flash")
	require.True(t, ok)
	require.NoError(t, SmokeTestExpr(expr), "默认表达式必须能编译并通过 smoke 向量")
}

// 峰谷两个分支的金额（1M 输入 + 1M 输出，分组倍率 1）：
//
//	低谷 = 1×1 + 1×4 = 5（价格单位）→ quota = 5 × QuotaPerUnit
//	高峰 = 1×2 + 1×8 = 10（价格单位）→ quota = 10 × QuotaPerUnit
//
// 真实时间只能命中一个分支，所以把条件分别替换为 true/false 来固定断言两个分支。
func TestDefaultDeepSeekPeakOffPeakAmounts(t *testing.T) {
	expr, ok := GetBillingExpr("deepseek-flash")
	require.True(t, ok)

	peakExpr := strings.Replace(expr, deepSeekPeakCondition, "true", 1)
	offPeakExpr := strings.Replace(expr, deepSeekPeakCondition, "false", 1)
	require.NotEqual(t, expr, peakExpr, "默认表达式的条件片段变了，请同步更新本测试")
	require.NotEqual(t, expr, offPeakExpr)

	params := billingexpr.TokenParams{P: 1_000_000, C: 1_000_000, Len: 2_000_000}

	peakCost, peakTrace, err := billingexpr.RunExpr(peakExpr, params)
	require.NoError(t, err)
	assert.Equal(t, "peak", peakTrace.MatchedTier)
	assert.Equal(t, 10_000_000.0, peakCost)
	assert.Equal(t, 10*common.QuotaPerUnit, peakCost/1_000_000*common.QuotaPerUnit)

	offPeakCost, offPeakTrace, err := billingexpr.RunExpr(offPeakExpr, params)
	require.NoError(t, err)
	assert.Equal(t, "offpeak", offPeakTrace.MatchedTier)
	assert.Equal(t, 5_000_000.0, offPeakCost)
	assert.Equal(t, 5*common.QuotaPerUnit, offPeakCost/1_000_000*common.QuotaPerUnit)

	// 不带替换时按当前时间命中其一，金额必须是两个分支之一。
	actualCost, actualTrace, err := billingexpr.RunExpr(expr, params)
	require.NoError(t, err)
	assert.Contains(t, []string{"peak", "offpeak"}, actualTrace.MatchedTier)
	assert.Contains(t, []float64{5_000_000.0, 10_000_000.0}, actualCost)
}

// 与 defaultModelRatio 的一致性：低谷未命中价 ¥1.00/M 对应 0.5 倍率，
// 因此"仅输入 1M"时表达式路径与比率路径必须得到同一 quota。
func TestDefaultDeepSeekOffPeakMatchesRatioPricing(t *testing.T) {
	ratio_setting.InitRatioSettings()

	ratio, ok, _ := ratio_setting.GetModelRatio("deepseek-flash")
	require.True(t, ok)
	require.Equal(t, 0.5, ratio)

	expr, ok := GetBillingExpr("deepseek-flash")
	require.True(t, ok)
	offPeakExpr := strings.Replace(expr, deepSeekPeakCondition, "false", 1)

	const promptTokens = 1_000_000
	cost, trace, err := billingexpr.RunExpr(offPeakExpr, billingexpr.TokenParams{P: promptTokens, Len: promptTokens})
	require.NoError(t, err)
	require.Equal(t, "offpeak", trace.MatchedTier)

	exprQuota := cost / 1_000_000 * common.QuotaPerUnit
	ratioQuota := float64(promptTokens) * ratio
	assert.Equal(t, ratioQuota, exprQuota)
}
