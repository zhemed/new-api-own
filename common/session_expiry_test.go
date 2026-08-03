package common

import "testing"

func TestIsLoginSessionExpiredTreatsZeroAsNever(t *testing.T) {
	now := int64(100)
	if IsLoginSessionExpired(0, now) {
		t.Fatal("zero expiry must remain active")
	}
	if !IsLoginSessionExpired(now-1, now) {
		t.Fatal("past expiry must be expired")
	}
	if IsLoginSessionExpired(now+1, now) {
		t.Fatal("future expiry must remain active")
	}
}
