package reasoning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IsDeepSeekV4Model 是 V4 家族判定的唯一实现：`deepseek-v4-` 前缀，或
// `deepseek-v4.<数字>` 次版本号前缀（deepseek-v4.1-flash）。点号后必须紧跟数字，
// 所以 `deepseek-v4.` / `deepseek-v4.txt` 这类噪音名不会被误判进家族。
func TestIsDeepSeekV4Model(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{name: "v4 flash", model: "deepseek-v4-flash", want: true},
		{name: "v4 pro", model: "deepseek-v4-pro", want: true},
		{name: "v4.1 flash", model: "deepseek-v4.1-flash", want: true},
		{name: "v4.1 flash with max suffix", model: "deepseek-v4.1-flash-max", want: true},
		{name: "dot without version", model: "deepseek-v4.", want: false},
		{name: "dot with non digit", model: "deepseek-v4.txt", want: false},
		{name: "no separator", model: "deepseek-v4", want: false},
		{name: "other deepseek family", model: "deepseek-v3-flash", want: false},
		{name: "non deepseek model", model: "gpt-4o", want: false},
		{name: "empty name", model: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDeepSeekV4Model(tt.model))
		})
	}
}

// ParseDeepSeekV4ThinkingSuffix 的契约：只有 V4 家族 + 受支持的档位（-none / -max）
// 才命中；未命中（无后缀 / 非家族 / 不支持的档位）时必须 ok=false 且原样返回模型名，
// 调用方据此跳过改写。
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
			name:     "non v4 family",
			model:    "deepseek-v3-flash-max",
			wantBase: "deepseek-v3-flash-max",
		},
		{
			name:     "unsupported effort suffix",
			model:    "deepseek-v4.1-flash-high",
			wantBase: "deepseek-v4.1-flash-high",
		},
		{
			name:     "noise name with supported suffix",
			model:    "deepseek-v4.txt-max",
			wantBase: "deepseek-v4.txt-max",
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
