package logger

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withDisplaySettings swaps the display-currency globals for one test and
// restores them afterwards, so cases stay independent of execution order.
func withDisplaySettings(t *testing.T, displayType string, symbol string, rate float64, usdExchangeRate float64) {
	t.Helper()

	general := operation_setting.GetGeneralSetting()
	previousGeneral := *general
	previousRate := operation_setting.USDExchangeRate
	t.Cleanup(func() {
		*general = previousGeneral
		operation_setting.USDExchangeRate = previousRate
	})

	general.QuotaDisplayType = displayType
	general.CustomCurrencySymbol = symbol
	general.CustomCurrencyExchangeRate = rate
	operation_setting.USDExchangeRate = usdExchangeRate
}

// The shipped default must render the yuan sign while leaving every numeric
// value untouched: one unit of quota is still exactly 1.000000.
func TestDefaultDisplayShowsYuanWithoutChangingValues(t *testing.T) {
	general := operation_setting.GetGeneralSetting()
	require.Equal(t, operation_setting.QuotaDisplayTypeCustom, operation_setting.GetQuotaDisplayType())
	require.Equal(t, "¥", general.CustomCurrencySymbol)
	require.Equal(t, 1.0, general.CustomCurrencyExchangeRate)

	assert.Equal(t, "¥1.000000", FormatQuota(int(common.QuotaPerUnit)))
	assert.Equal(t, "¥0.500000", FormatQuota(int(common.QuotaPerUnit)/2))
	assert.Equal(t, "¥1.000000 额度", LogQuota(int(common.QuotaPerUnit)))
}

// FormatQuota/LogQuota must keep one output shape per display type; these exact
// strings are what show up in client-visible log content, error messages and
// quota notifications.
func TestQuotaFormattingPerDisplayType(t *testing.T) {
	oneUnit := int(common.QuotaPerUnit)

	tests := []struct {
		name            string
		displayType     string
		symbol          string
		rate            float64
		usdExchangeRate float64
		wantFormat      string
		wantLog         string
	}{
		{
			name:        "usd divides only by quota per unit",
			displayType: operation_setting.QuotaDisplayTypeUSD,
			wantFormat:  "＄1.000000",
			wantLog:     "＄1.000000 额度",
		},
		{
			name:            "cny converts with the usd exchange rate",
			displayType:     operation_setting.QuotaDisplayTypeCNY,
			usdExchangeRate: 7.3,
			wantFormat:      "¥7.300000",
			wantLog:         "¥7.300000 额度",
		},
		{
			name:        "custom at rate one is symbol only",
			displayType: operation_setting.QuotaDisplayTypeCustom,
			symbol:      "¥",
			rate:        1,
			wantFormat:  "¥1.000000",
			wantLog:     "¥1.000000 额度",
		},
		{
			name:        "custom applies its own rate",
			displayType: operation_setting.QuotaDisplayTypeCustom,
			symbol:      "¥",
			rate:        2,
			wantFormat:  "¥2.000000",
			wantLog:     "¥2.000000 额度",
		},
		{
			name:        "custom falls back to the generic sign when the symbol is empty",
			displayType: operation_setting.QuotaDisplayTypeCustom,
			symbol:      "",
			rate:        1,
			wantFormat:  "¤1.000000",
			wantLog:     "¤1.000000 额度",
		},
		{
			name:        "custom falls back to rate one for a non-positive rate",
			displayType: operation_setting.QuotaDisplayTypeCustom,
			symbol:      "¥",
			rate:        0,
			wantFormat:  "¥1.000000",
			wantLog:     "¥1.000000 额度",
		},
		{
			name:        "tokens keeps the raw quota",
			displayType: operation_setting.QuotaDisplayTypeTokens,
			symbol:      "¥",
			rate:        1,
			wantFormat:  "500000",
			wantLog:     "500000 点额度",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDisplaySettings(t, tt.displayType, tt.symbol, tt.rate, tt.usdExchangeRate)

			assert.Equal(t, tt.wantFormat, FormatQuota(oneUnit))
			assert.Equal(t, tt.wantLog, LogQuota(oneUnit))
		})
	}
}
