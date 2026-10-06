package common

import (
	"fmt"
	"os"
	"strconv"
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
