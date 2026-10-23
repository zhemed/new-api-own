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

// deepseek-v4.1-flash 与 deepseek-v4-flash 同族：带次版本号的基名必须同样被识别，
// -max / -none 都归一到基名，且除基名本身外与既有 v4-flash 行为逐项一致。
func TestApplyDeepSeekV4OpenAIThinkingSuffixCoversV41Family(t *testing.T) {
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

	for _, tc := range cases {
		t.Run("deepseek-v4.1-flash"+tc.suffix, func(t *testing.T) {
			v41Info := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: "deepseek-v4.1-flash" + tc.suffix}}
			v41Request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4.1-flash" + tc.suffix}
			require.NoError(t, ApplyDeepSeekV4OpenAIThinkingSuffix(v41Info, v41Request))

			assert.Equal(t, "deepseek-v4.1-flash", v41Request.Model)
			assert.Equal(t, "deepseek-v4.1-flash", v41Info.UpstreamModelName)
			assert.Equal(t, tc.requestEffort, v41Request.ReasoningEffort)
			assert.Equal(t, tc.infoEffort, v41Info.ReasoningEffort)
			if tc.thinkingType == "" {
				assert.Empty(t, v41Request.THINKING)
			} else {
				assert.Equal(t, `{"type":"`+tc.thinkingType+`"}`, string(v41Request.THINKING))
			}

			// 与既有 v4-flash 的结果一致（只是基名不同）。
			v4Info := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: "deepseek-v4-flash" + tc.suffix}}
			v4Request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-flash" + tc.suffix}
			require.NoError(t, ApplyDeepSeekV4OpenAIThinkingSuffix(v4Info, v4Request))

			assert.Equal(t, v4Request.ReasoningEffort, v41Request.ReasoningEffort)
			assert.Equal(t, v4Info.ReasoningEffort, v41Info.ReasoningEffort)
			assert.Equal(t, string(v4Request.THINKING), string(v41Request.THINKING))
		})
	}
}
