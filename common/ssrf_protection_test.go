package common

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSRFProtectionRejectsLiteralPrivateAndReservedIPs(t *testing.T) {
	protection := &SSRFProtection{
		AllowPrivateIp:   false,
		DomainFilterMode: false,
		IpFilterMode:     false,
	}

	tests := []string{
		"127.0.0.1",
		"10.0.0.1",
		"169.254.169.254",
		"fc00::1",
		"::ffff:127.0.0.1",
	}
	for _, host := range tests {
		t.Run(host, func(t *testing.T) {
			require.Error(t, protection.ValidateNetworkTarget(host, 80))
		})
	}
}

func TestSSRFProtectionAllowsPrivateIPWhenExplicitlyEnabled(t *testing.T) {
	protection := &SSRFProtection{
		AllowPrivateIp:   true,
		DomainFilterMode: false,
		IpFilterMode:     false,
	}

	require.NoError(t, protection.ValidateNetworkTarget("10.0.0.1", 80))
}

func TestSSRFProtectionRejectsResolvedPrivateIP(t *testing.T) {
	protection := &SSRFProtection{
		AllowPrivateIp:         false,
		DomainFilterMode:       false,
		IpFilterMode:           false,
		ApplyIPFilterForDomain: true,
	}

	require.NoError(t, protection.ValidateNetworkTarget("example.com", 80))
	require.Error(t, protection.ValidateResolvedIP("example.com", net.ParseIP("169.254.169.254")))
}

func TestNewSSRFProtectionFromFetchSettingParsesPortRanges(t *testing.T) {
	protection, err := NewSSRFProtectionFromFetchSetting(false, false, false, nil, nil, []string{"80", "8000-8001"}, true)
	require.NoError(t, err)

	require.NoError(t, protection.ValidateNetworkTarget("example.com", 8001))
	require.Error(t, protection.ValidateNetworkTarget("example.com", 9000))
}

// TestIsPrivateIPClassifiesSpecialPurposeAddresses 是 SSRF 私网判定的表驱动契约：
// 判定只在 IsPrivateIP 一处维护，SSRF 校验（ValidateNetworkTarget / ValidateResolvedIP）
// 的行为由本表间接锁定。
func TestIsPrivateIPClassifiesSpecialPurposeAddresses(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		// IPv4 公网
		{"ipv4 public 8.8.8.8", net.ParseIP("8.8.8.8"), false},
		{"ipv4 public 1.1.1.1", net.ParseIP("1.1.1.1"), false},
		{"ipv4 public 172.32.0.1 (outside 172.16/12)", net.ParseIP("172.32.0.1"), false},
		// IPv4 私网/CGNAT 网段的边界外侧（防止 CIDR mask 写错把公网判成私网或反之）
		{"ipv4 public 9.255.255.255 (below 10/8)", net.ParseIP("9.255.255.255"), false},
		{"ipv4 public 11.0.0.0 (above 10/8)", net.ParseIP("11.0.0.0"), false},
		{"ipv4 public 172.15.255.255 (below 172.16/12)", net.ParseIP("172.15.255.255"), false},
		{"ipv4 public 172.32.0.0 (above 172.16/12)", net.ParseIP("172.32.0.0"), false},
		{"ipv4 public 192.167.255.255 (below 192.168/16)", net.ParseIP("192.167.255.255"), false},
		{"ipv4 public 192.169.0.0 (above 192.168/16)", net.ParseIP("192.169.0.0"), false},
		{"ipv4 public 100.63.255.255 (below CGNAT 100.64/10)", net.ParseIP("100.63.255.255"), false},
		{"ipv4 public 100.128.0.0 (above CGNAT 100.64/10)", net.ParseIP("100.128.0.0"), false},
		// IPv4 私网 RFC1918
		{"ipv4 private 10.0.0.1", net.ParseIP("10.0.0.1"), true},
		{"ipv4 private 10.255.255.255", net.ParseIP("10.255.255.255"), true},
		{"ipv4 private 172.16.0.1", net.ParseIP("172.16.0.1"), true},
		{"ipv4 private 172.31.255.255", net.ParseIP("172.31.255.255"), true},
		{"ipv4 private 192.168.1.1", net.ParseIP("192.168.1.1"), true},
		// IPv4 特殊用途
		{"ipv4 loopback 127.0.0.1", net.ParseIP("127.0.0.1"), true},
		{"ipv4 loopback 127.255.255.254", net.ParseIP("127.255.255.254"), true},
		{"ipv4 link-local 169.254.169.254", net.ParseIP("169.254.169.254"), true},
		{"ipv4 unspecified 0.0.0.0", net.ParseIP("0.0.0.0"), true},
		{"ipv4 cgnat 100.64.0.1", net.ParseIP("100.64.0.1"), true},
		{"ipv4 test-net-1 192.0.2.1", net.ParseIP("192.0.2.1"), true},
		{"ipv4 benchmarking 198.18.0.1", net.ParseIP("198.18.0.1"), true},
		{"ipv4 multicast 224.0.0.1", net.ParseIP("224.0.0.1"), true},
		{"ipv4 reserved 240.0.0.1", net.ParseIP("240.0.0.1"), true},
		{"ipv4 broadcast 255.255.255.255", net.ParseIP("255.255.255.255"), true},
		{"ipv4-mapped loopback ::ffff:127.0.0.1", net.ParseIP("::ffff:127.0.0.1"), true},
		{"ipv4-mapped private ::ffff:10.0.0.1", net.ParseIP("::ffff:10.0.0.1"), true},
		{"ipv4-mapped public ::ffff:8.8.8.8", net.ParseIP("::ffff:8.8.8.8"), false},
		// IPv6 公网
		{"ipv6 public 2001:4860:4860::8888", net.ParseIP("2001:4860:4860::8888"), false},
		{"ipv6 public 2606:4700:4700::1111", net.ParseIP("2606:4700:4700::1111"), false},
		{"ipv6 public fbff::1 (below fc00::/7)", net.ParseIP("fbff::1"), false},
		// IPv6 特殊用途
		{"ipv6 ula fc00::1", net.ParseIP("fc00::1"), true},
		{"ipv6 ula fd12:3456:789a::1", net.ParseIP("fd12:3456:789a::1"), true},
		{"ipv6 ula fdff::1 (upper edge of fc00::/7)", net.ParseIP("fdff::1"), true},
		{"ipv6 loopback ::1", net.ParseIP("::1"), true},
		{"ipv6 link-local fe80::1", net.ParseIP("fe80::1"), true},
		{"ipv6 unspecified ::", net.ParseIP("::"), true},
		{"ipv6 documentation 2001:db8::1", net.ParseIP("2001:db8::1"), true},
		{"ipv6 multicast ff02::1", net.ParseIP("ff02::1"), true},
		// 未解析/nil：不可信
		{"nil", nil, true},
		// 空但非 nil 的 IP 是无效输入；当前实现只把 nil 当作不可信，空切片按"非私网"处理
		// （行为与抽取前逐字一致；生产调用点只会传 net.ParseIP/解析器结果，不会传空切片）。
		{"empty non-nil", net.IP{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, IsPrivateIP(tc.ip))
		})
	}
}
