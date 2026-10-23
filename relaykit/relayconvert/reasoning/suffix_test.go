package reasoning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IsDeepSeekModel 的契约只有一条：模型名以 "deepseek" 开头（大小写敏感）即属家族，
// 家族成员都接受 -none/-max 档位后缀并默认思考。
func TestIsDeepSeekModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{name: "bare prefix", model: "deepseek", want: true},
		{name: "flash", model: "deepseek-flash", want: true},
		{name: "flash with max suffix", model: "deepseek-flash-max", want: true},
		{name: "v4.1 flash", model: "deepseek-v4.1-flash", want: true},
		{name: "v3 flash", model: "deepseek-v3-flash", want: true},
		{name: "uppercase first letter", model: "DeepSeek-flash", want: false},
		{name: "non deepseek model", model: "gpt-4o", want: false},
		{name: "empty name", model: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDeepSeekModel(tt.model))
		})
	}
}

// ParseDeepSeekV4ThinkingSuffix 的契约：deepseek 家族 + 受支持的档位（-none / -max）
// 才命中；未命中（无后缀 / 非家族 / 不支持的档位 / 大小写不同）时必须 ok=false
// 且原样返回模型名，调用方据此跳过改写。
func TestParseDeepSeekV4ThinkingSuffix(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantBase   string
		wantType   string
		wantEffort string
		wantOK     bool
	}{
		{
			name:       "flash max",
			model:      "deepseek-flash-max",
			wantBase:   "deepseek-flash",
			wantType:   "enabled",
			wantEffort: "max",
			wantOK:     true,
		},
		{
			name:     "flash none",
			model:    "deepseek-flash-none",
			wantBase: "deepseek-flash",
			wantType: "disabled",
			wantOK:   true,
		},
		{
			name:       "v4 max",
			model:      "deepseek-v4-flash-max",
			wantBase:   "deepseek-v4-flash",
			wantType:   "enabled",
			wantEffort: "max",
			wantOK:     true,
		},
		{
			name:     "v4 none",
			model:    "deepseek-v4-flash-none",
			wantBase: "deepseek-v4-flash",
			wantType: "disabled",
			wantOK:   true,
		},
		{
			name:       "v4.1 max",
			model:      "deepseek-v4.1-flash-max",
			wantBase:   "deepseek-v4.1-flash",
			wantType:   "enabled",
			wantEffort: "max",
			wantOK:     true,
		},
		{
			name:     "v4.1 none",
			model:    "deepseek-v4.1-flash-none",
			wantBase: "deepseek-v4.1-flash",
			wantType: "disabled",
			wantOK:   true,
		},
		{
			name:     "no suffix",
			model:    "deepseek-v4-flash",
			wantBase: "deepseek-v4-flash",
		},
		{
			name:     "non deepseek family",
			model:    "gpt-4o-max",
			wantBase: "gpt-4o-max",
		},
		{
			name:     "uppercase first letter",
			model:    "DeepSeek-flash-max",
			wantBase: "DeepSeek-flash-max",
		},
		{
			name:     "unsupported effort suffix",
			model:    "deepseek-v4.1-flash-high",
			wantBase: "deepseek-v4.1-flash-high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, thinkingType, effort, ok := ParseDeepSeekV4ThinkingSuffix(tt.model)

			// ok 决定其余三个字段是否有意义，先锁死它。
			require.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantBase, base)
			assert.Equal(t, tt.wantType, thinkingType)
			assert.Equal(t, tt.wantEffort, effort)
		})
	}
}
