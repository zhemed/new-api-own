package common

import (
	"strings"
)

// VersionComparison 是版本比较结果，取值与前端保持一致
// （web/src/features/system-settings/utils/version-compare.ts）。
type VersionComparison string

const (
	// VersionComparisonEqual 两侧版本等价。
	VersionComparisonEqual VersionComparison = "equal"
	// VersionComparisonNewer latest 高于 current，存在可用更新。
	VersionComparisonNewer VersionComparison = "newer"
	// VersionComparisonOlder latest 低于 current，当前构建更超前。
	VersionComparisonOlder VersionComparison = "older"
	// VersionComparisonUnknown 输入缺失或无法比较。
	VersionComparisonUnknown VersionComparison = "unknown"
)

// NormalizeVersion 去掉首尾空白、一个前导 v/V 前缀与构建元数据（"+" 之后），
// 保留预发布后缀；与前端 normalizeVersion 语义一致。
func NormalizeVersion(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	if idx := strings.Index(trimmed, "+"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(trimmed)
}

// CompareVersions 比较 current 与 latest，语义与前端 compareVersions 对齐：
// 逐段数值比较（缺段按 0），段内非纯数字时退化为字符串比较，
// 同一核心版本下预发布低于正式版，两侧任一为空则返回 unknown。
func CompareVersions(current, latest string) VersionComparison {
	currentNormalized := NormalizeVersion(current)
	latestNormalized := NormalizeVersion(latest)
	if currentNormalized == "" || latestNormalized == "" {
		return VersionComparisonUnknown
	}

	currentCore, currentPreRelease := splitVersionCore(currentNormalized)
	latestCore, latestPreRelease := splitVersionCore(latestNormalized)

	// 核心分段更大的一侧版本更高。
	if coreResult := compareVersionCores(currentCore, latestCore); coreResult != 0 {
		if coreResult > 0 {
			return VersionComparisonOlder
		}
		return VersionComparisonNewer
	}

	if currentPreRelease == latestPreRelease {
		return VersionComparisonEqual
	}
	// 核心相同时，没有预发布后缀的一侧是正式版，版本更高。
	if currentPreRelease == "" {
		return VersionComparisonOlder
	}
	if latestPreRelease == "" {
		return VersionComparisonNewer
	}
	if latestPreRelease > currentPreRelease {
		return VersionComparisonNewer
	}
	return VersionComparisonOlder
}

// splitVersionCore 把 "1.2.3-beta.1" 拆成核心分段与预发布后缀（只按第一个 "-" 切分）。
func splitVersionCore(value string) ([]string, string) {
	core, preRelease, _ := strings.Cut(value, "-")
	return strings.Split(core, "."), preRelease
}

func compareVersionCores(left, right []string) int {
	length := len(left)
	if len(right) > length {
		length = len(right)
	}
	for i := 0; i < length; i++ {
		leftSegment := "0"
		if i < len(left) {
			leftSegment = left[i]
		}
		rightSegment := "0"
		if i < len(right) {
			rightSegment = right[i]
		}
		if leftSegment == rightSegment {
			continue
		}
		if isNumericSegment(leftSegment) && isNumericSegment(rightSegment) {
			if result := compareNumericSegments(leftSegment, rightSegment); result != 0 {
				return result
			}
			continue
		}
		if leftSegment > rightSegment {
			return 1
		}
		return -1
	}
	return 0
}

// isNumericSegment 判断分段是否为纯数字段（与前端 /^\d+$/ 等价）。
func isNumericSegment(segment string) bool {
	if segment == "" {
		return false
	}
	for i := 0; i < len(segment); i++ {
		if segment[i] < '0' || segment[i] > '9' {
			return false
		}
	}
	return true
}

// compareNumericSegments 按数值比较两个纯数字段；用"去前导零 + 先比长度"避免整数溢出。
func compareNumericSegments(left, right string) int {
	left = strings.TrimLeft(left, "0")
	right = strings.TrimLeft(right, "0")
	if len(left) != len(right) {
		if len(left) > len(right) {
			return 1
		}
		return -1
	}
	return strings.Compare(left, right)
}
