package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 默认目录（2026-10-22 起）只保留 deepseek-flash 及其别名，加两个由引擎注入的音频默认模型。
// 价格按 DeepSeek 官方人民币低谷价：缓存命中 ¥0.02 / 未命中 ¥1.00 / 输出 ¥4.00（每 1M tokens）。
// 换算：显示值（每 1M）= 2 × model_ratio × 显示汇率，默认显示 ¥ + 汇率 1.0。
const (
	deepSeekOffPeakModelRatio      = 0.5
	deepSeekOffPeakCompletionRatio = 4.0
	deepSeekOffPeakCacheRatio      = 0.02
)

func TestDeepSeekFlashDefaultPricing(t *testing.T) {
	InitRatioSettings()

	for _, name := range []string{"deepseek-flash", "deepseek-v4-flash", "deepseek-chat", "deepseek-reasoner"} {
		t.Run(name, func(t *testing.T) {
			modelRatio, ok, _ := GetModelRatio(name)
			require.True(t, ok, "%s 必须带默认倍率", name)
			assert.Equal(t, deepSeekOffPeakModelRatio, modelRatio)

			assert.Equal(t, deepSeekOffPeakCompletionRatio, GetCompletionRatio(name))

			cacheRatio, ok := GetCacheRatio(name)
			require.True(t, ok, "%s 必须带默认缓存倍率", name)
			assert.Equal(t, deepSeekOffPeakCacheRatio, cacheRatio)
		})
	}

	// deepseek 渠道的模型列表带 -none / -max 思考档位后缀（relay/channel/deepseek/constants.go），
	// FormatMatchingModelName 会把它归一到 deepseek-v4-flash，价格必须仍然命中。
	for _, name := range []string{"deepseek-v4-flash-none", "deepseek-v4-flash-max"} {
		modelRatio, ok, _ := GetModelRatio(name)
		require.True(t, ok, "%s 必须能归一到别名倍率", name)
		assert.Equal(t, deepSeekOffPeakModelRatio, modelRatio)
	}
}

// 被裁掉的目录条目不能再带默认计费配置，否则新装实例仍会为它们定价。
func TestDefaultCatalogNoLongerPricesFormerModels(t *testing.T) {
	InitRatioSettings()

	for _, name := range []string{
		"gpt-4o", "gpt-4", "gpt-5", "o3", "claude-sonnet-4-20250514", "gemini-2.5-pro",
		"ERNIE-4.0-8K", "glm-4", "qwen-plus", "grok-3-beta", "command-r",
		"deepseek-v4-pro", "deepseek-coder", "deepseek-ai/DeepSeek-V3.1",
		"dall-e-3", "gpt-image-1", "sora-2", "gpt-4o-mini-tts",
	} {
		t.Run(name, func(t *testing.T) {
			_, ok, _ := GetModelRatio(name)
			assert.False(t, ok, "%s 不应再有默认倍率", name)

			_, found := GetModelPrice(name, false)
			assert.False(t, found, "%s 不应再有默认固定价", name)

			_, cached := GetCacheRatio(name)
			assert.False(t, cached, "%s 不应再有默认缓存倍率", name)
		})
	}
}

// 引擎自身会注入的默认模型（音频端点）与功能动作计费键必须保留，否则这些端点会直接失败。
func TestDefaultCatalogKeepsEngineInjectedDefaults(t *testing.T) {
	InitRatioSettings()

	speechRatio, ok, _ := GetModelRatio("tts-1") // middleware/distributor.go 为 /v1/audio/speech 注入
	require.True(t, ok)
	assert.Equal(t, 7.5, speechRatio)

	transcribeRatio, ok, _ := GetModelRatio("whisper-1") // 为 /v1/audio/transcriptions|translations 注入
	require.True(t, ok)
	assert.Equal(t, 15.0, transcribeRatio)

	midjourneyPrice, ok := GetModelPrice("mj_imagine", false) // Midjourney 动作计费键
	require.True(t, ok)
	assert.Equal(t, 0.1, midjourneyPrice)
}

// 目录规模被显式锁定：新增默认模型必须同时更新这里，避免又长回成"上百个默认模型"。
func TestDefaultCatalogSizeStaysPruned(t *testing.T) {
	InitRatioSettings()

	assert.Len(t, GetModelRatioCopy(), 9)
	assert.Len(t, GetModelPriceCopy(), 20)
	assert.Len(t, GetCompletionRatioCopy(), 4)
	assert.Len(t, GetCacheRatioCopy(), 4)
	assert.Empty(t, GetCreateCacheRatioCopy())
	assert.Empty(t, GetImageRatioCopy())
	assert.Empty(t, GetAudioRatioCopy())
}
