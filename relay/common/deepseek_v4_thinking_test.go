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
