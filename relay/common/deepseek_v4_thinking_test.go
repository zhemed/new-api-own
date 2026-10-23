package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

func TestApplyDeepSeekV4OpenAIThinkingSuffixDefaultsHighForSuffixlessV4(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			UpstreamModelName: "deepseek-v4-flash",
		},
	}
	request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-flash"}

	err := ApplyDeepSeekV4OpenAIThinkingSuffix(info, request)
	require.NoError(t, err)

	// The defaulted high effort must be visible on both the upstream request
	// and the relay info so usage logs record it (regression for the missing
	// info sync).
	assert.Equal(t, "high", request.ReasoningEffort)
	assert.Equal(t, "high", info.ReasoningEffort)
}

func TestApplyDeepSeekV4OpenAIThinkingSuffixSkipsExplicitEffort(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			UpstreamModelName: "deepseek-v4-flash",
		},
	}
	request := &dto.GeneralOpenAIRequest{
		Model:           "deepseek-v4-flash",
		ReasoningEffort: "low",
	}

	err := ApplyDeepSeekV4OpenAIThinkingSuffix(info, request)
	require.NoError(t, err)

	assert.Equal(t, "low", request.ReasoningEffort)
	assert.Equal(t, "", info.ReasoningEffort)
}

func TestApplyDeepSeekV4OpenAIThinkingSuffixRespectsThinkingDisabled(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			UpstreamModelName: "deepseek-v4-flash",
		},
	}
	thinking, err := common.Marshal(map[string]string{"type": "disabled"})
	require.NoError(t, err)
	request := &dto.GeneralOpenAIRequest{
		Model:    "deepseek-v4-flash",
		THINKING: thinking,
	}

	err = ApplyDeepSeekV4OpenAIThinkingSuffix(info, request)
	require.NoError(t, err)

	assert.Equal(t, "", request.ReasoningEffort)
	assert.Equal(t, "", info.ReasoningEffort)
}

func TestApplyDeepSeekV4OpenAIThinkingSuffixIgnoresNonV4Models(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			UpstreamModelName: "gpt-4o",
		},
	}
	request := &dto.GeneralOpenAIRequest{Model: "gpt-4o"}

	err := ApplyDeepSeekV4OpenAIThinkingSuffix(info, request)
	require.NoError(t, err)

	assert.Equal(t, "", request.ReasoningEffort)
	assert.Equal(t, "", info.ReasoningEffort)
}

func TestApplyDeepSeekV4OpenAIThinkingSuffixMapsMaxSuffix(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			UpstreamModelName: "deepseek-v4-flash-max",
		},
	}
	request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-flash-max"}

	err := ApplyDeepSeekV4OpenAIThinkingSuffix(info, request)
	require.NoError(t, err)

	assert.Equal(t, "deepseek-v4-flash", request.Model)
	assert.Equal(t, "max", request.ReasoningEffort)
	assert.Equal(t, "max", info.ReasoningEffort)
	assert.Equal(t, "deepseek-v4-flash", info.UpstreamModelName)
}

// deepseek 家族（任何 deepseek 前缀）都必须支持 -none / -max 档位与无后缀默认 high：
// deepseek-flash 是用户挂在渠道上的名字，deepseek-v4.1-flash 是默认模板，
// deepseek-v4-flash 作为既有参照，三者同后缀结果必须逐项一致。
func TestApplyDeepSeekV4OpenAIThinkingSuffixCoversDeepSeekFamily(t *testing.T) {
	cases := []struct {
		suffix        string
		thinkingType  string
		requestEffort string
		infoEffort    string
	}{
		{suffix: "", thinkingType: "", requestEffort: "high", infoEffort: "high"},
		{suffix: "-max", thinkingType: "enabled", requestEffort: "max", infoEffort: "max"},
		{suffix: "-none", thinkingType: "disabled", requestEffort: "", infoEffort: "none"},
	}

	models := []string{"deepseek-flash", "deepseek-v4.1-flash", "deepseek-v4-flash"}
	for _, model := range models {
		for _, tc := range cases {
			t.Run(model+tc.suffix, func(t *testing.T) {
				info := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: model + tc.suffix}}
				request := &dto.GeneralOpenAIRequest{Model: model + tc.suffix}
				require.NoError(t, ApplyDeepSeekV4OpenAIThinkingSuffix(info, request))

				// 档位后缀被剥掉后落到基名；无后缀时保持原名。
				assert.Equal(t, model, request.Model)
				assert.Equal(t, model, info.UpstreamModelName)
				assert.Equal(t, tc.requestEffort, request.ReasoningEffort)
				assert.Equal(t, tc.infoEffort, info.ReasoningEffort)
				if tc.thinkingType == "" {
					assert.Empty(t, request.THINKING)
				} else {
					assert.Equal(t, `{"type":"`+tc.thinkingType+`"}`, string(request.THINKING))
				}

				// 与既有 v4-flash 的同后缀结果一致（只是基名不同）。
				refInfo := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: "deepseek-v4-flash" + tc.suffix}}
				refRequest := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-flash" + tc.suffix}
				require.NoError(t, ApplyDeepSeekV4OpenAIThinkingSuffix(refInfo, refRequest))

				assert.Equal(t, refRequest.ReasoningEffort, request.ReasoningEffort)
				assert.Equal(t, refInfo.ReasoningEffort, info.ReasoningEffort)
				assert.Equal(t, string(refRequest.THINKING), string(request.THINKING))
			})
		}
	}
}
