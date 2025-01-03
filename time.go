package cloudcostexplorer

import (
	"fmt"
	"time"

	"cloud.google.com/go/civil"
	"github.com/invzhi/timex"
)

const (
	DefaultSkipDays = -2 // usually data is fresh only from 2 days ago
)

// FromCivilDate converts from [civil.Date] to [timex.Date].
func FromCivilDate(dt civil.Date) timex.Date {
	return timex.MustNewDate(dt.Year, int(dt.Month), dt.Day)
}

// TimeStartEnd converts 2 [timex.Date] values to 2 [time.Time] values, with the time part set to 00:00:00 and 23:59:59
// respectively.
func TimeStartEnd(start, end timex.Date) (time.Time, time.Time) {
	return time.Date(start.Year(), time.Month(start.Month()), start.Day(), 0, 0, 0, 0, time.UTC),
		time.Date(end.Year(), time.Month(end.Month()), end.Day(), 23, 59, 59, 999999, time.UTC)
}

// DateBetweenDates returns whether the passed date is between start and end.
func DateBetweenDates(dt timex.Date, start, end timex.Date) bool {
	return dt.Equal(start) || dt.Equal(end) || (dt.After(start) && dt.Before(end))
}

// GenerateQueryPeriods generates "amount" list of periods of the same number of days as the passed range,
// one before the other.
func GenerateQueryPeriods(start, end timex.Date, amount int) []QueryPeriod {
	diffDays := end.Sub(start) + 1
	var periods []QueryPeriod
	for i := amount - 1; i >= 0; i-- {
		pstart := start.Add(0, 0, -diffDays*i)
		pend := end.Add(0, 0, -diffDays*i)
		periods = append(periods, QueryPeriod{
			Start: pstart,
			End:   pend,
		})
	}
	return periods
}

// EndingOfMonth returns the last day of the month/year of the passed date.
func EndingOfMonth(date timex.Date) timex.Date {
	// return date.Add(0, 1, -date.Day()+1)
	return date.Add(0, 1, -1)
}

// FormatShortDate formats a date like "Feb 15".
func FormatShortDate(dt timex.Date) string {
	return fmt.Sprintf("%s %d", shortMonthNames[dt.Month()-1], dt.Day())
}

// ShortMonthName returns the 3-letter english month abbreviation.
func ShortMonthName(month time.Month) string {
	return shortMonthNames[month-1]
}

var shortMonthNames = []string{
	"Jan",
	"Feb",
	"Mar",
	"Apr",
	"May",
	"Jun",
	"Jul",
	"Aug",
	"Sep",
	"Oct",
	"Nov",
	"Dec",
}
