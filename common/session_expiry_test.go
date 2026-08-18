package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLoginSessionExpiredTreatsZeroAsNever(t *testing.T) {
	now := int64(100)
	assert.False(t, IsLoginSessionExpired(0, now), "zero expiry must remain active")
	assert.True(t, IsLoginSessionExpired(now-1, now), "past expiry must be expired")
	assert.False(t, IsLoginSessionExpired(now+1, now), "future expiry must remain active")
}
