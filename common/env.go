package common

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetEnvOrDefault(env string, defaultValue int) int {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	num, err := strconv.Atoi(os.Getenv(env))
	if err != nil {
		SysError(fmt.Sprintf("failed to parse %s: %s, using default value: %d", env, err.Error(), defaultValue))
		return defaultValue
	}
	return num
}

func GetEnvOrDefaultString(env string, defaultValue string) string {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	return os.Getenv(env)
}

func GetEnvOrDefaultBool(env string, defaultValue bool) bool {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(os.Getenv(env))
	if err != nil {
		SysError(fmt.Sprintf("failed to parse %s: %s, using default value: %t", env, err.Error(), defaultValue))
		return defaultValue
	}
	return b
}

// GetEnvOrDefaultDuration parses a Go duration string (e.g. "10m", "1h30m").
// A value of "0" or a negative duration disables the feature it configures, so
// callers can treat a non-positive result as "off".
func GetEnvOrDefaultDuration(env string, defaultValue time.Duration) time.Duration {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(os.Getenv(env))
	if err != nil {
		SysError(fmt.Sprintf("failed to parse %s: %s, using default value: %s", env, err.Error(), defaultValue))
		return defaultValue
	}
	return d
}

// sizeUnitMultipliers maps a size suffix to its byte multiplier. Both the short
// and the "B" spellings are accepted ("200M" == "200MB"); matching is
// case-insensitive and 1024-based, matching how memory budgets are usually
// expressed.
var sizeUnitMultipliers = map[string]int64{
	"":   1,
	"B":  1,
	"K":  1 << 10,
	"KB": 1 << 10,
	"M":  1 << 20,
	"MB": 1 << 20,
	"G":  1 << 30,
	"GB": 1 << 30,
	"T":  1 << 40,
	"TB": 1 << 40,
}

// GetEnvOrDefaultSize parses a byte size such as "200MB", "512KB" or a plain
// byte count like "209715200". Units are 1024-based and optional. A value of 0
// or "" means "off" (the default is returned), which callers treat as disabled.
func GetEnvOrDefaultSize(env string, defaultValue int64) int64 {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	value, err := ParseSize(os.Getenv(env))
	if err != nil {
		SysError(fmt.Sprintf("failed to parse %s: %s, using default value: %d", env, err.Error(), defaultValue))
		return defaultValue
	}
	return value
}

// ParseSize parses a byte size with an optional 1024-based unit suffix.
func ParseSize(raw string) (int64, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(raw))
	if trimmed == "" {
		return 0, errors.New("empty size")
	}
	digits := strings.TrimRightFunc(trimmed, func(r rune) bool {
		return r < '0' || r > '9'
	})
	unit := strings.TrimSpace(trimmed[len(digits):])
	multiplier, ok := sizeUnitMultipliers[unit]
	if !ok {
		return 0, fmt.Errorf("unknown size unit %q", unit)
	}
	amount, err := strconv.ParseFloat(digits, 64)
	if err != nil {
		return 0, err
	}
	if amount < 0 {
		return 0, errors.New("negative size")
	}
	return int64(amount * float64(multiplier)), nil
}
