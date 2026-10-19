package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The "month" ranking is a calendar month to date, not a rolling 30-day window:
// the first point must be the 1st of the current month at 00:00 server time.
func TestRankingTimeRangeMonthIsCalendarMonth(t *testing.T) {
	config, err := rankingConfig("month")
	require.NoError(t, err)

	now := time.Date(2026, time.October, 18, 13, 45, 0, 0, time.Local)
	start, end := rankingTimeRange(config, now)

	wantStart := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.Local)
	assert.Equal(t, wantStart.Unix(), start)
	assert.Equal(t, now.Unix(), end)
}

func TestRankingTimeRangeMonthHandlesYearBoundary(t *testing.T) {
	config, err := rankingConfig("month")
	require.NoError(t, err)

	now := time.Date(2026, time.January, 3, 9, 0, 0, 0, time.Local)
	start, _ := rankingTimeRange(config, now)

	wantStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local)
	assert.Equal(t, wantStart.Unix(), start)
}

// The comparison period for "this month" is the whole previous calendar month.
func TestPreviousRankingTimeRangeMonthIsPreviousCalendarMonth(t *testing.T) {
	config, err := rankingConfig("month")
	require.NoError(t, err)

	currentStart := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.Local).Unix()
	previousStart, previousEnd := previousRankingTimeRange(config, currentStart)

	assert.Equal(
		t,
		time.Date(2026, time.September, 1, 0, 0, 0, 0, time.Local).Unix(),
		previousStart,
	)
	// one second before the current month starts -> last instant of September
	assert.Equal(t, currentStart-1, previousEnd)
	assert.Equal(
		t,
		time.Date(2026, time.September, 30, 23, 59, 59, 0, time.Local).Unix(),
		previousEnd,
	)
}

// The "year" ranking is a calendar year to date: the first point is January 1st.
func TestRankingTimeRangeYearIsCalendarYear(t *testing.T) {
	config, err := rankingConfig("year")
	require.NoError(t, err)

	now := time.Date(2026, time.October, 18, 13, 45, 0, 0, time.Local)
	start, end := rankingTimeRange(config, now)

	assert.Equal(t, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local).Unix(), start)
	assert.Equal(t, now.Unix(), end)
}

func TestRankingTimeRangeYearHandlesJanuaryBoundary(t *testing.T) {
	config, err := rankingConfig("year")
	require.NoError(t, err)

	now := time.Date(2026, time.January, 1, 0, 30, 0, 0, time.Local)
	start, _ := rankingTimeRange(config, now)

	assert.Equal(t, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local).Unix(), start)
}

func TestPreviousRankingTimeRangeYearIsPreviousCalendarYear(t *testing.T) {
	config, err := rankingConfig("year")
	require.NoError(t, err)

	currentStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local).Unix()
	previousStart, previousEnd := previousRankingTimeRange(config, currentStart)

	assert.Equal(t, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local).Unix(), previousStart)
	assert.Equal(t, currentStart-1, previousEnd)
	assert.Equal(
		t,
		time.Date(2025, time.December, 31, 23, 59, 59, 0, time.Local).Unix(),
		previousEnd,
	)
}

// The year buckets are monthly: roughly a dozen points instead of 365 daily
// ones, and clearly fewer than the weekly grid it replaced.
func TestYearBucketsAreMonthly(t *testing.T) {
	config, err := rankingConfig("year")
	require.NoError(t, err)

	assert.Equal(t, int64(30*24*3600), config.bucketSize)

	// A full calendar year must land in the 12–13 bucket range.
	const yearSeconds = 365 * 24 * 3600
	buckets := (yearSeconds + int(config.bucketSize) - 1) / int(config.bucketSize)
	assert.GreaterOrEqual(t, buckets, 12)
	assert.LessOrEqual(t, buckets, 13)
}

// Buckets are epoch-aligned, so a calendar range must not open on a label from
// before the range.
func TestClampRankingBucketsToRangeStartsAtTheRangeStart(t *testing.T) {
	rangeStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.Local).Unix()
	epochAligned := time.Date(2025, time.December, 22, 0, 0, 0, 0, time.Local).Unix()
	secondBucket := epochAligned + 30*24*3600

	buckets := clampRankingBucketsToRange(
		[]model.RankingQuotaBucket{
			{Bucket: epochAligned, ModelName: "gpt", Tokens: 10},
			{Bucket: secondBucket, ModelName: "gpt", Tokens: 20},
		},
		rangeStart,
	)

	require.Len(t, buckets, 2)
	assert.Equal(t, rangeStart, buckets[0].Bucket)
	assert.Equal(t, secondBucket, buckets[1].Bucket, "later buckets stay untouched")
	assert.Equal(t, int64(10), buckets[0].Tokens)
}

// The other periods keep their rolling-window behaviour.
func TestRankingTimeRangeOtherPeriodsStayRolling(t *testing.T) {
	now := time.Date(2026, time.October, 18, 13, 45, 0, 0, time.Local)

	for _, period := range []string{"week", "today"} {
		config, err := rankingConfig(period)
		require.NoError(t, err)

		start, end := rankingTimeRange(config, now)
		assert.Equal(t, now.Unix(), end, "%s: end must be now", period)
		assert.Equal(
			t,
			now.Add(-config.duration).Unix(),
			start,
			"%s: start must stay the rolling window",
			period,
		)
	}
}
