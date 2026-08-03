package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestLoginSessionExpiresAtSupportsNeverExpireMode(t *testing.T) {
	previous := common.LoginSessionNeverExpires
	defer func() { common.LoginSessionNeverExpires = previous }()

	common.LoginSessionNeverExpires = true
	if got := loginSessionExpiresAt(100); got != 0 {
		t.Fatalf("never-expire mode returned %d, want 0", got)
	}

	common.LoginSessionNeverExpires = false
	if got := loginSessionExpiresAt(100); got <= 100 {
		t.Fatalf("default mode returned %d, want a future expiry", got)
	}
}
