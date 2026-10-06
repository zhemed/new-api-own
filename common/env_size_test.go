package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseSize pins the size syntax accepted by LOG_MEMORY_MAX_BYTES and any
// future byte budget: plain byte counts and 1024-based unit suffixes.
func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int64
		wantErr bool
	}{
		{name: "plain byte count", raw: "209715200", want: 209715200},
		{name: "megabytes", raw: "200MB", want: 200 * 1024 * 1024},
		{name: "megabytes lower case", raw: "200mb", want: 200 * 1024 * 1024},
		{name: "short megabyte suffix", raw: "200M", want: 200 * 1024 * 1024},
		{name: "space before unit", raw: " 200 MB ", want: 200 * 1024 * 1024},
		{name: "kilobytes", raw: "512KB", want: 512 * 1024},
		{name: "gigabytes", raw: "1GB", want: 1024 * 1024 * 1024},
		{name: "fractional gigabytes", raw: "1.5GB", want: int64(1.5 * 1024 * 1024 * 1024)},
		{name: "zero", raw: "0", want: 0},
		{name: "bare bytes suffix", raw: "4096B", want: 4096},
		{name: "unknown unit", raw: "200XB", wantErr: true},
		{name: "not a number", raw: "lots", wantErr: true},
		{name: "empty", raw: "   ", wantErr: true},
		{name: "negative", raw: "-1MB", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSize(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestGetEnvOrDefaultSize pins the env-facing wrapper: unset/empty falls back to
// the default ("off" for a budget), a configured value is parsed, and an
// unparsable value falls back instead of failing the caller.
func TestGetEnvOrDefaultSize(t *testing.T) {
	const env = "TEST_LOG_MEMORY_MAX_BYTES"

	t.Run("empty falls back to default", func(t *testing.T) {
		t.Setenv(env, "")
		require.Zero(t, GetEnvOrDefaultSize(env, 0))
		require.Equal(t, int64(7), GetEnvOrDefaultSize(env, 7))
	})

	t.Run("configured value is parsed", func(t *testing.T) {
		t.Setenv(env, "200MB")
		require.Equal(t, int64(209715200), GetEnvOrDefaultSize(env, 0))
	})

	t.Run("plain byte count is parsed", func(t *testing.T) {
		t.Setenv(env, "209715200")
		require.Equal(t, int64(209715200), GetEnvOrDefaultSize(env, 0))
	})

	t.Run("invalid value falls back to default", func(t *testing.T) {
		t.Setenv(env, "banana")
		require.Equal(t, int64(42), GetEnvOrDefaultSize(env, 42))
	})
}
