package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// withDisplaySettings swaps the display-currency globals for one test and
// restores them afterwards, so cases stay independent of execution order.
func withDisplaySettings(t *testing.T, displayType string, symbol string, rate float64) {
	t.Helper()
	previous := generalSetting
	t.Cleanup(func() { generalSetting = previous })

	generalSetting.QuotaDisplayType = displayType
	generalSetting.CustomCurrencySymbol = symbol
	generalSetting.CustomCurrencyExchangeRate = rate
}

// The product default is CNY in symbol-only form: CUSTOM with the ¥ symbol and a
// 1.0 rate, so every displayed amount keeps the numeric value it had in USD.
func TestDefaultQuotaDisplayIsSymbolOnlyCNY(t *testing.T) {
	def := GetGeneralSetting()

	assert.Equal(t, QuotaDisplayTypeCustom, def.QuotaDisplayType)
	assert.Equal(t, "¥", def.CustomCurrencySymbol)
	assert.Equal(t, 1.0, def.CustomCurrencyExchangeRate)
}

func TestGetCurrencySymbolPerDisplayType(t *testing.T) {
	tests := []struct {
		name        string
		displayType string
		symbol      string
		want        string
	}{
		{name: "usd keeps the canonical ascii symbol", displayType: QuotaDisplayTypeUSD, want: "$"},
		{name: "cny uses the yuan sign", displayType: QuotaDisplayTypeCNY, want: "¥"},
		{name: "custom uses the configured symbol", displayType: QuotaDisplayTypeCustom, symbol: "¥", want: "¥"},
		{name: "custom falls back when unset", displayType: QuotaDisplayTypeCustom, symbol: "", want: "¤"},
		{name: "tokens has no currency symbol", displayType: QuotaDisplayTypeTokens, symbol: "¥", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDisplaySettings(t, tt.displayType, tt.symbol, 1.0)
			assert.Equal(t, tt.want, GetCurrencySymbol())
		})
	}
}

func TestGetUsdToCurrencyRatePerDisplayType(t *testing.T) {
	const usdToCny = 7.3

	tests := []struct {
		name        string
		displayType string
		rate        float64
		want        float64
	}{
		{name: "usd is one to one", displayType: QuotaDisplayTypeUSD, want: 1},
		{name: "cny uses the usd exchange rate", displayType: QuotaDisplayTypeCNY, want: usdToCny},
		{name: "custom uses the configured rate", displayType: QuotaDisplayTypeCustom, rate: 2.5, want: 2.5},
		{name: "custom falls back to one for a zero rate", displayType: QuotaDisplayTypeCustom, rate: 0, want: 1},
		{name: "custom falls back to one for a negative rate", displayType: QuotaDisplayTypeCustom, rate: -3, want: 1},
		{name: "tokens does not convert", displayType: QuotaDisplayTypeTokens, rate: 2.5, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDisplaySettings(t, tt.displayType, "¥", tt.rate)
			assert.Equal(t, tt.want, GetUsdToCurrencyRate(usdToCny))
		})
	}
}
