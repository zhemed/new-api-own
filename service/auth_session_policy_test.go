package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestLoginSessionExpiresAtSupportsNeverExpireMode(t *testing.T) {
	previous := common.LoginSessionNeverExpires
	defer func() { common.LoginSessionNeverExpires = previous }()

	common.LoginSessionNeverExpires = true
	got := loginSessionExpiresAt(100)
	assert.Equal(t, int64(0), got, "never-expire mode returned %d, want 0", got)

	common.LoginSessionNeverExpires = false
	got = loginSessionExpiresAt(100)
	assert.Greater(t, got, int64(100), "default mode returned %d, want a future expiry", got)
}
