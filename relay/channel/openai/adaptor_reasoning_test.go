package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIRequestRecordsReasoningEffortForChatModel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeOpenAI,
			UpstreamModelName: "deepseek-v4-flash",
		},
	}
	request := &dto.GeneralOpenAIRequest{
		Model:           "deepseek-v4-flash",
		ReasoningEffort: "high",
	}

	converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err, "ConvertOpenAIRequest returned error")
	assert.Same(t, request, converted, "ConvertOpenAIRequest returned a different request pointer")
	assert.Equal(t, "high", info.ReasoningEffort, "ReasoningEffort mismatch")
}
