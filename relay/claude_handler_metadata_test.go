package relay

import (
	"encoding/json"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
)

func TestSyncClaudeRequestMetadata(t *testing.T) {
	tests := []struct {
		name       string
		output     json.RawMessage
		wantEffort string
	}{
		{
			name:       "valid effort",
			output:     json.RawMessage(`{"effort":"high"}`),
			wantEffort: "high",
		},
		{
			name:       "absent output config",
			wantEffort: "",
		},
		{
			name:       "empty effort",
			output:     json.RawMessage(`{"effort":""}`),
			wantEffort: "",
		},
		{
			name:       "malformed output config",
			output:     json.RawMessage(`{"effort":`),
			wantEffort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{}
			request := &dto.ClaudeRequest{OutputConfig: tt.output}

			syncClaudeRequestMetadata(info, request)

			assert.Equal(t, tt.wantEffort, info.GetReasoningEffort())
		})
	}
}
