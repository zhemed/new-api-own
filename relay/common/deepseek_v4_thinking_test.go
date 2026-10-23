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

// 三条路径的统一锚点（这里是最先落地的 OpenAI Chat 路径）：deepseek-flash（实际挂在渠道上
// 的名字）与 deepseek-v4.1-flash（默认模板）× 三档（无后缀 / -max / -none）都必须与既有
// deepseek-v4-flash 同后缀逐项一致；每一格同时另锁一份绝对期望值——只对照另一个模型
// 会掩盖"两边一起错"。
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

	for _, model := range []string{"deepseek-flash", "deepseek-v4.1-flash"} {
		for _, tc := range cases {
			t.Run(model+tc.suffix, func(t *testing.T) {
				info := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: model + tc.suffix}}
				request := &dto.GeneralOpenAIRequest{Model: model + tc.suffix}
				require.NoError(t, ApplyDeepSeekV4OpenAIThinkingSuffix(info, request))

				// 绝对期望值：档位后缀被剥掉后落到基名；无后缀时保持原名。
				assert.Equal(t, model, request.Model)
				assert.Equal(t, model, info.UpstreamModelName)
				assert.Equal(t, tc.requestEffort, request.ReasoningEffort)
				assert.Equal(t, tc.infoEffort, info.ReasoningEffort)
				if tc.thinkingType == "" {
					assert.Empty(t, request.THINKING)
				} else {
					assert.Equal(t, `{"type":"`+tc.thinkingType+`"}`, string(request.THINKING))
				}

				// 参照模型 deepseek-v4-flash 跑同一格：先确认参照自身归一正确，再逐项对照。
				refInfo := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: "deepseek-v4-flash" + tc.suffix}}
				refRequest := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-flash" + tc.suffix}
				require.NoError(t, ApplyDeepSeekV4OpenAIThinkingSuffix(refInfo, refRequest))

				assert.Equal(t, "deepseek-v4-flash", refRequest.Model)
				assert.Equal(t, "deepseek-v4-flash", refInfo.UpstreamModelName)
				assert.Equal(t, refRequest.ReasoningEffort, request.ReasoningEffort)
				assert.Equal(t, refInfo.ReasoningEffort, info.ReasoningEffort)
				assert.Equal(t, string(refRequest.THINKING), string(request.THINKING))
			})
		}
	}
}

// Claude 路径的同一套锚点：ApplyDeepSeekV4ClaudeThinkingSuffix 只做"档位后缀 →
// thinking / output_config"改写，不负责补默认档位——无后缀时补 high 由
// relay/claude_handler.go:131 的守卫负责，所以本函数在无后缀格必须保持 Thinking=nil、
// OutputConfig 为空、info.ReasoningEffort 为空。
func TestApplyDeepSeekV4ClaudeThinkingSuffixCoversDeepSeekFamily(t *testing.T) {
	cases := []struct {
		suffix           string
		wantThinkingType string
		wantOutputConfig string
		wantInfoEffort   string
	}{
		{suffix: "", wantThinkingType: "", wantOutputConfig: "", wantInfoEffort: ""},
		{suffix: "-max", wantThinkingType: "enabled", wantOutputConfig: `{"effort":"max"}`, wantInfoEffort: "max"},
		// -none 禁用 thinking；本函数把 info.ReasoningEffort 留空（OpenAI/Responses 路径记 "none"）。
		// 这是既有行为，本测试固化现状，不做"顺手统一"。
		{suffix: "-none", wantThinkingType: "disabled", wantOutputConfig: "", wantInfoEffort: ""},
	}

	for _, model := range []string{"deepseek-flash", "deepseek-v4.1-flash"} {
		for _, tc := range cases {
			t.Run(model+tc.suffix, func(t *testing.T) {
				info := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: model + tc.suffix}}
				request := &dto.ClaudeRequest{Model: model + tc.suffix}
				require.NoError(t, ApplyDeepSeekV4ClaudeThinkingSuffix(info, request))

				// 绝对期望值。
				assert.Equal(t, model, request.Model)
				assert.Equal(t, model, info.UpstreamModelName)
				assert.Equal(t, tc.wantInfoEffort, info.ReasoningEffort)
				if tc.wantThinkingType == "" {
					assert.Nil(t, request.Thinking, "无后缀不得凭空写 thinking；补 high 由 claude_handler 的守卫负责")
				} else {
					require.NotNil(t, request.Thinking)
					assert.Equal(t, tc.wantThinkingType, request.Thinking.Type)
				}
				assert.Equal(t, tc.wantOutputConfig, string(request.OutputConfig))

				// 参照模型 deepseek-v4-flash 跑同一格：先确认参照自身归一正确，再逐项对照。
				refInfo := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: "deepseek-v4-flash" + tc.suffix}}
				refRequest := &dto.ClaudeRequest{Model: "deepseek-v4-flash" + tc.suffix}
				require.NoError(t, ApplyDeepSeekV4ClaudeThinkingSuffix(refInfo, refRequest))

				assert.Equal(t, "deepseek-v4-flash", refRequest.Model)
				assert.Equal(t, "deepseek-v4-flash", refInfo.UpstreamModelName)
				assert.Equal(t, refInfo.ReasoningEffort, info.ReasoningEffort)
				assert.Equal(t, string(refRequest.OutputConfig), string(request.OutputConfig))
				if refRequest.Thinking == nil {
					assert.Nil(t, request.Thinking)
				} else {
					require.NotNil(t, request.Thinking)
					assert.Equal(t, refRequest.Thinking.Type, request.Thinking.Type)
				}
			})
		}
	}
}

// Responses 路径的同一套锚点：无后缀补 high、-max 传 max、-none 传 none（禁用思考），
// 模型名同样在后缀存在时归一到基名。
func TestApplyDeepSeekV4ResponsesThinkingSuffixCoversDeepSeekFamily(t *testing.T) {
	cases := []struct {
		suffix         string
		wantEffort     string
		wantInfoEffort string
	}{
		{suffix: "", wantEffort: "high", wantInfoEffort: "high"},
		{suffix: "-max", wantEffort: "max", wantInfoEffort: "max"},
		{suffix: "-none", wantEffort: "none", wantInfoEffort: "none"},
	}

	for _, model := range []string{"deepseek-flash", "deepseek-v4.1-flash"} {
		for _, tc := range cases {
			t.Run(model+tc.suffix, func(t *testing.T) {
				info := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: model + tc.suffix}}
				request := &dto.OpenAIResponsesRequest{Model: model + tc.suffix}
				ApplyDeepSeekV4ResponsesThinkingSuffix(info, request)

				// 绝对期望值：档位后缀被剥掉后落到基名；无后缀时保持原名。
				assert.Equal(t, model, request.Model)
				assert.Equal(t, model, info.UpstreamModelName)
				require.NotNil(t, request.Reasoning)
				assert.Equal(t, tc.wantEffort, request.Reasoning.Effort)
				assert.Equal(t, tc.wantInfoEffort, info.ReasoningEffort)

				// 参照模型 deepseek-v4-flash 跑同一格：先确认参照自身归一正确，再逐项对照。
				refInfo := &RelayInfo{ChannelMeta: &ChannelMeta{UpstreamModelName: "deepseek-v4-flash" + tc.suffix}}
				refRequest := &dto.OpenAIResponsesRequest{Model: "deepseek-v4-flash" + tc.suffix}
				ApplyDeepSeekV4ResponsesThinkingSuffix(refInfo, refRequest)

				assert.Equal(t, "deepseek-v4-flash", refRequest.Model)
				assert.Equal(t, "deepseek-v4-flash", refInfo.UpstreamModelName)
				require.NotNil(t, refRequest.Reasoning)
				assert.Equal(t, refRequest.Reasoning.Effort, request.Reasoning.Effort)
				assert.Equal(t, refInfo.ReasoningEffort, info.ReasoningEffort)
			})
		}
	}
}
