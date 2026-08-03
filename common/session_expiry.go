package common

// LoginSessionNeverExpires keeps dashboard sessions active until an explicit
// revoke, logout, user disablement, or auth-version change.
var LoginSessionNeverExpires = false

// IsLoginSessionExpired treats zero as the explicit never-expire sentinel.
func IsLoginSessionExpired(expiresAt, now int64) bool {
	return expiresAt > 0 && expiresAt <= now
}
