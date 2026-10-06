package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNormalizeVersion 的用例与前端 normalizeVersion 单测一一对应
// （web/src/features/system-settings/utils/__tests__/version-compare.test.ts）。
func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "strips lower-case v prefix", raw: "v0.0.6", want: "0.0.6"},
		{name: "strips upper-case v prefix", raw: "V0.0.6", want: "0.0.6"},
		{name: "trims whitespace and keeps unprefixed version", raw: " 0.0.6 ", want: "0.0.6"},
		{name: "drops build metadata", raw: "v0.0.6+build.9", want: "0.0.6"},
		{name: "keeps pre-release suffix", raw: "v0.0.6-beta.1", want: "0.0.6-beta.1"},
		{name: "empty input stays empty", raw: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, NormalizeVersion(tt.raw))
		})
	}
}

// TestCompareVersions 与前端 compareVersions 的判定保持一致：tag 的 v 前缀、
// 缺段补零、数值比较、预发布低于正式版、输入缺失为 unknown。
func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    VersionComparison
	}{
		{
			name:    "release tag differing only by v prefix is equal",
			current: "0.0.6",
			latest:  "v0.0.6",
			want:    VersionComparisonEqual,
		},
		{
			name:    "padding difference is equal",
			current: "0.0.6",
			latest:  "v0.0.6.0",
			want:    VersionComparisonEqual,
		},
		{
			name:    "published release ahead reports newer",
			current: "0.0.6",
			latest:  "v0.0.7",
			want:    VersionComparisonNewer,
		},
		{
			name:    "minor bump reports newer",
			current: "0.0.6",
			latest:  "v0.1.0",
			want:    VersionComparisonNewer,
		},
		{
			name:    "major bump reports newer",
			current: "0.0.6",
			latest:  "v1.0.0",
			want:    VersionComparisonNewer,
		},
		{
			name:    "segments compare numerically not lexicographically",
			current: "0.0.9",
			latest:  "v0.0.10",
			want:    VersionComparisonNewer,
		},
		{
			name:    "running build ahead reports older",
			current: "0.0.10",
			latest:  "v0.0.9",
			want:    VersionComparisonOlder,
		},
		{
			name:    "historical tag format still matches",
			current: "v0.0.4",
			latest:  "v0.0.4",
			want:    VersionComparisonEqual,
		},
		{
			name:    "pre-release ranks below its final release",
			current: "0.0.6-beta.1",
			latest:  "v0.0.6",
			want:    VersionComparisonNewer,
		},
		{
			name:    "final release is ahead of the running pre-release",
			current: "0.0.6",
			latest:  "v0.0.6-beta.1",
			want:    VersionComparisonOlder,
		},
		{
			name:    "two pre-releases compare by their suffix",
			current: "1.2.0-rc.1",
			latest:  "1.2.0-rc.2",
			want:    VersionComparisonNewer,
		},
		{
			name:    "non-numeric segment falls back to string compare",
			current: "1.2.x",
			latest:  "1.2.y",
			want:    VersionComparisonNewer,
		},
		{
			// 纯数字段按"去前导零后先比长度"比较，超出 int64 也不会溢出。
			name:    "oversized numeric segments still compare numerically",
			current: "1.0.99999999999999999999",
			latest:  "1.0.100000000000000000000",
			want:    VersionComparisonNewer,
		},
		{
			name:    "missing latest is unknown",
			current: "0.0.6",
			latest:  "",
			want:    VersionComparisonUnknown,
		},
		{
			name:    "missing current is unknown",
			current: "",
			latest:  "v0.0.6",
			want:    VersionComparisonUnknown,
		},
		{
			name:    "whitespace-only input is unknown",
			current: "   ",
			latest:  "v0.0.6",
			want:    VersionComparisonUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, CompareVersions(tt.current, tt.latest))
		})
	}
}
