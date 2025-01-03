package main

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/invzhi/timex"
	"github.com/rrgmc/cloudcostexplorer"
)

func IsPeriod2(r *http.Request) bool {
	period2 := r.URL.Query().Get("period2")
	if period2 == "" || strings.HasPrefix(period2, "R") {
		return false
	}
	return true
}

func ParsePeriod(r *http.Request) ([]cloudcostexplorer.QueryPeriodList, string, error) {
	period := r.URL.Query().Get("period")
	period2 := r.URL.Query().Get("period2")
	if period == "" {
		period = "d14"
	}

	var list []cloudcostexplorer.QueryPeriodList

	currentDate := initialDateWithSkipDays(r.URL.Query().Get("skipdays"))

	start, end, desc, err := ParsePeriodValue(period, currentDate)
	if err != nil {
		return nil, "", err
	}

	if strings.HasPrefix(period2, "R") {
		prepeat, perr := strconv.Atoi(strings.TrimPrefix(period2, "R"))
		if perr != nil {
			return nil, "", fmt.Errorf("could not parse 'repeat' value '%s': %w", period2, perr)
		}

		list = append(list, cloudcostexplorer.QueryPeriodList{
			Periods: cloudcostexplorer.GenerateQueryPeriods(start, end, prepeat),
		})

		desc = desc + fmt.Sprintf(" (repeat %d)", prepeat)
		period2 = ""
	} else {
		list = append(list, cloudcostexplorer.QueryPeriodList{
			Periods: cloudcostexplorer.GenerateQueryPeriods(start, end, 1),
		})
	}

	if period2 != "" {
		pi := 2
		for {
			curperiod := r.URL.Query().Get(fmt.Sprintf("period%d", pi))
			if curperiod == "" {
				break
			}
			start2, end2, _, err := ParsePeriodValue(curperiod, start.AddDays(-1))
			if err != nil {
				return nil, "", err
			}

			list = slices.Insert(list, 0, cloudcostexplorer.QueryPeriodList{
				Periods: cloudcostexplorer.GenerateQueryPeriods(start2, end2, 1),
			})
			pi++
		}
	}

	return list, desc, nil
}

func ParsePeriodValue(period string, currentDate timex.Date) (timex.Date, timex.Date, string, error) {
	var start, end timex.Date
	desc := "selected dates"

	if strings.HasPrefix(period, "d") {
		pperiod, perr := strconv.Atoi(strings.TrimPrefix(period, "d"))
		if perr != nil {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'days' value '%s': %w", period, perr)
		}
		startDate := currentDate.AddDays(-pperiod + 1)
		start = startDate
		end = currentDate
		desc = fmt.Sprintf("%d days", pperiod)
	} else if strings.HasPrefix(period, "m") {
		pperiod, perr := strconv.Atoi(strings.TrimPrefix(period, "m"))
		if perr != nil {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'months' value '%s': %w", period, perr)
		}
		startDate := currentDate.Add(0, -pperiod, 1)
		start = startDate
		end = currentDate
		desc = fmt.Sprintf("%d months", pperiod)
	} else if strings.HasPrefix(period, "M") {
		if len(period) != 7 {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'month' value '%s'", period)
		}

		pyear, pyearerr := strconv.Atoi(period[1:5])
		pmonth, pmontherr := strconv.Atoi(period[5:7])
		if pyearerr != nil {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'month' value '%s': %w", period, pyearerr)
		}
		if pmontherr != nil {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'month' value '%s': %w", period, pmontherr)
		}
		startDate, err := timex.NewDate(pyear, pmonth, 1)
		if err != nil {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("invalid month value: %s", period)
		}
		start = startDate
		end = cloudcostexplorer.EndingOfMonth(startDate)
		desc = fmt.Sprintf("%s/%d", cloudcostexplorer.ShortMonthName(time.Month(pmonth)), pyear)
	} else if strings.HasPrefix(period, "T") {
		if len(period) != 11 && len(period) != 22 {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'date range' value '%s'", period)
		}

		startDate, serr := timex.ParseDate("YYYY-MM-DD", period[1:11])
		if serr != nil {
			return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'date range' value '%s': %w", period, serr)
		}
		var endDate timex.Date
		if len(period) == 22 {
			endDate, serr = timex.ParseDate("YYYY-MM-DD", period[12:22])
			if serr != nil {
				return timex.Date{}, timex.Date{}, "", fmt.Errorf("could not parse 'date range' value '%s': %w", period, serr)
			}
		} else {
			endDate = startDate
		}

		start = startDate
		end = endDate
		if start.Equal(end) {
			desc = start.Format("DD/MMM/YYYY")
		} else {
			desc = fmt.Sprintf("%s to %s", start.Format("DD/MMM/YYYY"), end.Format("DD/MMM/YYYY"))
		}
	} else {
		return timex.Date{}, timex.Date{}, "", fmt.Errorf("unknown period value '%s'", period)
	}

	return start, end, desc, nil
}

func initialDateWithSkipDays(skipDays string) timex.Date {
	ret := timex.Today(time.UTC).AddDays(cloudcostexplorer.DefaultSkipDays) // usually data is fresh only from 2 days ago
	if skipDays != "" {
		days, err := strconv.Atoi(skipDays)
		if err != nil {
			fmt.Printf("error parsing skip days: %v\n", err)
		} else {
			ret = ret.AddDays(-days)
		}
	}
	return ret
}
