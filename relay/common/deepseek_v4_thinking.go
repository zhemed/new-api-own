package common

import (
	"fmt"
	"strings"

	appcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
)

// ApplyDeepSeekV4ClaudeThinkingSuffix maps DeepSeek V4 thinking suffixes
// (-max / -none) to Anthropic Messages thinking / output_config so pass-through
// channels (for example newapi relay) behave like the official DeepSeek channel.
func ApplyDeepSeekV4ClaudeThinkingSuffix(info *RelayInfo, request *dto.ClaudeRequest) error {
	modelName := request.Model
	if info != nil && info.ChannelMeta != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}
	baseModel, thinkingType, effort, ok := reasoning.ParseDeepSeekV4ThinkingSuffix(modelName)
	if !ok {
		return nil
	}
	request.Model = baseModel
	request.Thinking = &dto.Thinking{Type: thinkingType}
	if effort == "" {
		request.OutputConfig = nil
	} else {
		outputConfig, err := appcommon.Marshal(map[string]string{
			"effort": effort,
		})
		if err != nil {
			return fmt.Errorf("error marshalling output_config: %w", err)
		}
		request.OutputConfig = outputConfig
	}
	if info != nil {
		if info.ChannelMeta != nil {
			info.UpstreamModelName = baseModel
		}
		info.ReasoningEffort = effort
	}
	return nil
}

// ApplyDeepSeekV4OpenAIThinkingSuffix maps DeepSeek V4 thinking suffixes to
// OpenAI Chat Completions thinking / reasoning_effort fields.
func ApplyDeepSeekV4OpenAIThinkingSuffix(info *RelayInfo, request *dto.GeneralOpenAIRequest) error {
	modelName := request.Model
	if info != nil && info.ChannelMeta != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}
	baseModel, thinkingType, effort, ok := reasoning.ParseDeepSeekV4ThinkingSuffix(modelName)
	if !ok {
		applyDeepSeekV4OpenAIDefaultEffort(modelName, request)
		return nil
	}
	thinking, err := appcommon.Marshal(map[string]string{
		"type": thinkingType,
	})
	if err != nil {
		return fmt.Errorf("error marshalling thinking: %w", err)
	}
	request.Model = baseModel
	request.THINKING = thinking
	request.ReasoningEffort = effort
	if info != nil {
		if info.ChannelMeta != nil {
			info.UpstreamModelName = baseModel
		}
		info.ReasoningEffort = effort
	}
	return nil
}

// ApplyDeepSeekV4ResponsesThinkingSuffix maps DeepSeek V4 thinking suffixes to
// OpenAI Responses reasoning effort (none disables thinking mode).
func ApplyDeepSeekV4ResponsesThinkingSuffix(info *RelayInfo, request *dto.OpenAIResponsesRequest) {
	modelName := request.Model
	if info != nil && info.ChannelMeta != nil && info.UpstreamModelName != "" {
		modelName = info.UpstreamModelName
	}
	baseModel, thinkingType, effort, ok := reasoning.ParseDeepSeekV4ThinkingSuffix(modelName)
	if ok {
		if thinkingType == "disabled" {
			effort = "none"
		}
		request.Model = baseModel
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{}
		}
		request.Reasoning.Effort = effort
		if info != nil && info.ChannelMeta != nil {
			info.UpstreamModelName = baseModel
		}
	}
	if !ok && strings.HasPrefix(modelName, "deepseek-v4-") {
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{}
		}
		if request.Reasoning.Effort == "" {
			request.Reasoning.Effort = "high"
		}
	}
	if info != nil && request.Reasoning != nil {
		info.ReasoningEffort = request.Reasoning.Effort
	}
}

// applyDeepSeekV4OpenAIDefaultEffort records the default thinking effort (high)
// for suffix-less DeepSeek V4 chat requests unless thinking is explicitly disabled.
func applyDeepSeekV4OpenAIDefaultEffort(modelName string, request *dto.GeneralOpenAIRequest) {
	if request == nil || request.ReasoningEffort != "" || !strings.HasPrefix(modelName, "deepseek-v4-") {
		return
	}
	disabled := false
	if len(request.THINKING) > 0 {
		var t struct {
			Type string `json:"type"`
		}
		if err := appcommon.UnmarshalJsonStr(string(request.THINKING), &t); err == nil && t.Type == "disabled" {
			disabled = true
		}
	}
	if !disabled {
		request.ReasoningEffort = "high"
	}
}
