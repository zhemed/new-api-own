package service

import (
	"testing"
	"time"

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

// The other periods keep their rolling-window behaviour.
func TestRankingTimeRangeOtherPeriodsStayRolling(t *testing.T) {
	now := time.Date(2026, time.October, 18, 13, 45, 0, 0, time.Local)

	for _, period := range []string{"week", "today", "year"} {
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
