package main

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/invzhi/timex"
	"github.com/rrgmc/cloudcostexplorer"
	ui2 "github.com/rrgmc/cloudcostexplorer/cmd/cloudcostexplorer/ui"
)

func handlerCostExplorer(item string, cloud cloudcostexplorer.Cloud) http.Handler {
	return ui2.HTTPHandlerWithError(func(w http.ResponseWriter, r *http.Request) error {

		rootPath := fmt.Sprintf("/costexplorer/%s", url.PathEscape(item))

		uq := cloudcostexplorer.NewURLQuery(rootPath)

		// parameters
		var paramExists bool
		var limit int
		var mincost int
		var showdiff, showdiffpct bool
		var sort string
		var sortidx int
		var sortdir string
		var search string

		if limit, paramExists = HTTPQueryIntValue(r, "limit", 200); paramExists {
			uq.Set("limit", fmt.Sprintf("%d", limit))
		}
		if mincost, paramExists = HTTPQueryIntValue(r, "mincost", 1); paramExists {
			uq.Set("mincost", fmt.Sprintf("%d", mincost))
		}
		if showdiff, paramExists = HTTPQueryBoolValue(r, "showdiff", false); paramExists {
			uq.Set("showdiff", fmt.Sprintf("%t", showdiff))
		}
		if showdiffpct, paramExists = HTTPQueryBoolValue(r, "showdiffpct", false); paramExists {
			uq.Set("showdiffpct", fmt.Sprintf("%t", showdiffpct))
		}
		if sort, paramExists = HTTPQueryStringValue(r, "sort", ""); paramExists {
			uq.Set("sort", sort)
		}
		if sortidx, paramExists = HTTPQueryIntValue(r, "sortidx", -1); paramExists {
			uq.Set("sortidx", fmt.Sprintf("%d", sortidx))
		}
		if sortdir, paramExists = HTTPQueryStringValue(r, "sortdir", "D"); paramExists {
			uq.Set("sortdir", sortdir)
		}
		if search, paramExists = HTTPQueryStringValue(r, "search", ""); paramExists {
			uq.Set("search", search)
		}

		// filters
		var filters []cloudcostexplorer.QueryFilter
		var activeFilters []activeFilter

		for _, parameter := range cloud.Parameters() {
			if !parameter.IsFilter {
				continue
			}
			queryParamName := fmt.Sprintf("f%s", parameter.ID)
			queryParamValue, ok := r.URL.Query()[queryParamName]
			if !ok {
				continue
			}
			filterValue := strings.Join(queryParamValue, ",")
			uq.Set(queryParamName, filterValue)

			filters = append(filters, cloudcostexplorer.QueryFilter{
				ID:    parameter.ID,
				Value: filterValue,
			})
			activeFilters = append(activeFilters, activeFilter{
				parameter:  parameter,
				title:      cloud.ParameterTitle(parameter.ID, filterValue),
				paramNames: []string{queryParamName},
			})
		}

		// groups
		var groups []cloudcostexplorer.QueryGroup

		for groupIdx := range cloud.MaxGroupBy() {
			groupParam := fmt.Sprintf("group%d", groupIdx+1)
			if groupValue := r.URL.Query().Get(groupParam); groupValue != "" {
				parameter, ok := cloud.Parameters().FindById(groupValue)
				if !ok || !parameter.IsGroup {
					return fmt.Errorf("invalid group '%s'", groupValue)
				}

				uq.Set(groupParam, groupValue)

				var groupData string
				if parameter.HasData {
					groupValue, groupData, ok = strings.Cut(groupValue, cloudcostexplorer.DataSeparator)
					if !ok && parameter.DataRequired {
						return fmt.Errorf("group '%s' requires a data value", groupValue)
					}
				}

				groups = append(groups, cloudcostexplorer.QueryGroup{
					ID:   parameter.ID,
					Data: groupData,
				})
			} else {
				break
			}
		}

		if len(groups) == 0 {
			uq.Set("group1", cloud.Parameters().DefaultGroup().ID)
			groups = append(groups, cloudcostexplorer.QueryGroup{
				ID: cloud.Parameters().DefaultGroup().ID,
			})
		}

		periodList, periodDesc, err := ParsePeriod(r)
		if err != nil {
			return err
		}

		if period := r.URL.Query().Get("period"); period != "" {
			uq.Set("period", period)
		}
		var periodParams []string
		if period2 := r.URL.Query().Get("period2"); period2 != "" {
			uq.Set("period2", period2)
			periodParams = append(periodParams, "period2")
		}
		maxPeriodParam := 3
		for {
			curpparam := fmt.Sprintf("period%d", maxPeriodParam)
			if xp := r.URL.Query().Get(curpparam); xp != "" {
				periodParams = append(periodParams, curpparam)
				uq.Set(curpparam, xp)
			} else {
				break
			}
			maxPeriodParam++
		}

		var periodMatchErrors []error

		queryData, err := cloudcostexplorer.QueryHandler(r.Context(), cloud,
			cloudcostexplorer.WithQueryHandlerFilters(filters...),
			cloudcostexplorer.WithQueryHandlerGroups(groups...),
			cloudcostexplorer.WithQueryHandlerPeriodLists(periodList...),
			cloudcostexplorer.WithQueryHandlerOnPeriodMatchError(func(item cloudcostexplorer.CloudQueryItem, matchCount int) error {
				periodMatchErrors = append(periodMatchErrors, fmt.Errorf("period '%s' should match 1 but matched %d", item.Date.String(), matchCount))
				return nil
			}),
		)
		if err != nil {
			return err
		}

		if sort != "" && sortidx == -1 {
			sortidx = len(queryData.Periods) - 1
		}

		if queryData.ExtraOutput != nil {
			defer queryData.ExtraOutput.Close()
		}

		out := ui2.NewHTTPOutput(w)

		out.DocBegin(fmt.Sprintf("%s - CloudCostExplorer", item))

		out.NavBegin(rootPath)
		out.NavMenuBegin()

		// GROUPS BEGIN
		maxGroupBy := min(len(groups)+1, cloud.MaxGroupBy())
		for i := range maxGroupBy {
			keyI := i + 1
			isNewGroup := keyI > len(groups)
			isLast := keyI >= len(groups)
			keyName := fmt.Sprintf("group%d", keyI)
			keydesc := fmt.Sprintf(" %d", keyI)

			out.NavDropdownBegin(fmt.Sprintf("Group by%s", keydesc))

			popUQ := uq.Clone()
			clearUQ := uq.Clone()

			for ck := keyI; ck <= maxGroupBy; ck++ {
				popUQ.Move(fmt.Sprintf("group%d", ck+1), fmt.Sprintf("group%d", ck))
				clearUQ.Remove(fmt.Sprintf("group%d", ck))
			}

			if !isNewGroup {
				out.NavDropdownHeader(queryData.Groups[i].Title(true))
				out.NavDropdownDivider()
			}

			if !isNewGroup && !isLast {
				out.NavDropdownItem("POP", popUQ.String())

				out.NavDropdownItem(fmt.Sprintf("INVERT WITH GROUP %d", keyI+1), uq.Clone().
					Swap(fmt.Sprintf("group%d", keyI), fmt.Sprintf("group%d", keyI+1)).String())
			}
			if !isNewGroup {
				out.NavDropdownItem("CLEAR", clearUQ.String())
			}

			for _, parameter := range cloud.Parameters() {
				if !parameter.IsGroup || parameter.DataRequired {
					continue
				}
				mname := parameter.Name
				if parameter.MenuTitle != "" {
					mname = parameter.MenuTitle
				}
				out.NavDropdownItem(mname, uq.Clone().Set(keyName, parameter.ID).String())
			}

			out.NavDropdownEnd()
		}

		// GROUPS END

		// PERIOD BEGIN

		yesterday := timex.Today(time.UTC).AddDays(-1)
		for p := range 3 {
			if p > 1 && !IsPeriod2(r) {
				break
			}

			periodDesc := "Period"
			periodParam := "period"
			if p > 0 {
				periodDesc = "Previous Period"
				if p > 1 {
					periodDesc += fmt.Sprintf(" %d", p)
				}
				periodParam = fmt.Sprintf("period%d", p+1)
			}

			out.NavDropdownBegin(periodDesc)
			if p > 0 {
				clearUQ := uq.Clone()

				for ck := p + 1; ck <= maxPeriodParam; ck++ {
					clearUQ.Remove(fmt.Sprintf("period%d", ck))
				}

				out.NavDropdownItem("CLEAR", clearUQ.String())
				out.NavDropdownDivider()
			}
			if p == 1 {
				out.NavDropdownItem("REPEAT 2", uq.Clone().Set(periodParam, "R2").String())
				out.NavDropdownItem("REPEAT 3", uq.Clone().Set(periodParam, "R3").String())
				out.NavDropdownItem("REPEAT 7", uq.Clone().Set(periodParam, "R7").String())
				out.NavDropdownItem("REPEAT 30", uq.Clone().Set(periodParam, "R30").String())
				out.NavDropdownDivider()
			}
			out.NavDropdownItem("Yesterday", uq.Clone().Set(periodParam, fmt.Sprintf("T%s", yesterday.Format("YYYY-MM-DD"))).String())
			out.NavDropdownItem("1 day", uq.Clone().Set(periodParam, "d1").String())
			out.NavDropdownItem("5 days", uq.Clone().Set(periodParam, "d5").String())
			out.NavDropdownItem("14 days", uq.Clone().Set(periodParam, "d14").String())
			out.NavDropdownItem("1 month", uq.Clone().Set(periodParam, "m1").String())
			out.NavDropdownItem("2 months", uq.Clone().Set(periodParam, "m2").String())
			out.NavDropdownItem("3 months", uq.Clone().Set(periodParam, "m3").String())
			out.NavDropdownItem("6 months", uq.Clone().Set(periodParam, "m6").String())
			out.NavDropdownItem("12 months", uq.Clone().Set(periodParam, "m12").String())
			curYear, curMonth, curDay := time.Now().Date()
			for dct := range 8 {
				out.NavDropdownItem(fmt.Sprintf("%s/%04d", cloudcostexplorer.ShortMonthName(curMonth), curYear),
					uq.Clone().Set(periodParam, fmt.Sprintf("M%04d%02d", curYear, curMonth)).String())
				if dct < 3 && curDay > 2 {
					out.NavDropdownItem(fmt.Sprintf("%s/%04d to day", cloudcostexplorer.ShortMonthName(curMonth), curYear),
						uq.Clone().Set(periodParam, fmt.Sprintf("T%04d-%02d-01|%04d-%02d-%02d", curYear, curMonth, curYear, curMonth, curDay-2)).String())
				}
				curMonth -= 1
				if curMonth < 1 {
					curMonth = 12
					curYear -= 1
				}
			}
			out.NavDropdownEnd()
		}

		out.NavDropdownBegin("Config")
		out.NavDropdownItem("Default cost limit", uq.Clone().Remove("mincost").String())
		out.NavDropdownItem("Remove cost limit", uq.Clone().Set("mincost", "0").String())
		out.NavDropdownDivider()
		scdq := uq.Clone().
			Set("showdiff", "1").
			Set("showdiffpct", "1")
		if sort == "" {
			scdq.
				Set("sort", "diff")
		}
		out.NavDropdownItem("Show cost difference", scdq.String())
		out.NavDropdownItem("Hide cost difference", uq.Clone().Remove("showdiff", "showdiffpct").String())
		hcdv := uq.Clone()
		hcdp := uq.Clone()
		if showdiff {
			hcdv.Remove("showdiff")
		} else {
			hcdv.Set("showdiff", "1")
		}
		if showdiffpct {
			hcdp.Remove("showdiffpct")
		} else {
			hcdp.Set("showdiffpct", "1")
		}
		out.NavDropdownItem("Toggle cost difference value", hcdv.String())
		out.NavDropdownItem("Toggle cost difference %", hcdp.String())
		out.NavDropdownEnd()

		// PERIOD END

		out.NavMenuEnd()

		out.NavTextCustom(`<span class="badge bg-secondary">Period</span>`, periodDesc)

		// FILTERS BEGIN

		for _, actiteFilter := range activeFilters {
			out.NavTextCustom(fmt.Sprintf(`<span class="badge bg-secondary">%s <a href="%s"><i class="bi bi-trash text-white"></i></a></span>`,
				actiteFilter.parameter.Name,
				uq.Clone().Remove(actiteFilter.paramNames...)),
				cloudcostexplorer.EllipticalTruncate(actiteFilter.title, 32))
		}

		// FILTERS END

		out.NavSearch(search, uq)

		out.NavEnd()

		out.BodyBegin()

		// DATA
		out.Writeln(`<table class="table table-striped table-bordered table-sm">`)

		// HEADER BEGIN
		out.Writef(`<thead><tr>`)
		out.Writeln(`<th></th>`)
		for gidx, currentgroup := range queryData.Groups {
			out.Writef(`<th>%s (%d)</th>`, currentgroup.Title(true), gidx+1)
		}
		for periodIdx, period := range queryData.Periods {
			if periodIdx > 0 && showdiff {
				out.Writef(`<th>Diff&nbsp;%s</th>`,
					ui2.SortIcon(sort == "diff" && sortidx == periodIdx, sortdir,
						uq.Clone().Set("sort", "diff").
							Set("sortidx", fmt.Sprintf("%d", periodIdx))))
			}
			if periodIdx > 0 && showdiffpct {
				out.Writef(`<th>Diff%%&nbsp;%s</th>`,
					ui2.SortIcon(sort == "diffpct" && sortidx == periodIdx, sortdir,
						uq.Clone().Set("sort", "diffpct").
							Set("sortidx", fmt.Sprintf("%d", periodIdx))))
			}
			periodIcon := ""
			if periodIdx == len(queryData.Periods)-1 {
				periodIcon = fmt.Sprintf(`&nbsp;%s`,
					ui2.SortIcon(sort == "", "", uq.Clone().Remove("sort", "sortidx", "sortdir")))
			} else {
				periodIcon = fmt.Sprintf(`&nbsp;<a title="Filter only this period" class="link-secondary" href="%s"><i class="bi bi-filter-circle"></i></a>`,
					uq.Clone().Remove(periodParams...).Set("period", period.StringFilter()))
			}
			out.Writef(`<th>%s%s</th>`, period.StringWithDuration(!queryData.PeriodsSameDuration), periodIcon)
		}
		out.Writeln(`</tr></thead>`)
		// HEADER END

		out.Writeln(`<tbody>`)

		// TOTAL BEGIN
		out.Writef("<tr><td align=\"center\">%s</td><td colspan=\"%d\"><strong>TOTAL</strong></td>",
			humanize.Comma(int64(len(queryData.Items))),
			len(queryData.Groups))
		for periodIdx, period := range queryData.Periods {
			costClass := ""
			if periodIdx > 0 {
				costDiff, pctCostDiff := periodCostDiffGet(periodIdx, queryData.Periods)

				costClass = "text-danger"
				if costDiff <= 0 {
					costClass = "text-success"
				}
				if showdiff || showdiffpct {
					if showdiff {
						out.Writef(`<td class="%s" align="right">%s</td>`, costClass, cloudcostexplorer.FormatMoney(costDiff))
					}
					if showdiffpct {
						out.Writef(`<td class="%s" align="right">%s%%</td>`, costClass, humanize.CommafWithDigits(pctCostDiff, 2))
					}
					costClass = ""
				}
			}

			out.Writef(`<td class="%s" align="right"><strong>%s</strong></td>`,
				costClass, cloudcostexplorer.FormatMoney(period.TotalValue))
		}
		out.Writeln("</tr>")
		// TOTAL END

		// DATA BEGIN
		isLimit := false
		ct := 1
		var skipMinCost int
		var skipSearch int
		totalCols := 1 + len(queryData.Groups) + len(queryData.Periods)
		if showdiff {
			totalCols += len(queryData.Periods) - 1
		}
		if showdiffpct {
			totalCols += len(queryData.Periods) - 1
		}

		slices.SortFunc(queryData.Items, func(a, b *cloudcostexplorer.Item) int {
			if sort == "diff" || sort == "diffpct" {
				if sortidx > 0 && sortidx < len(queryData.Periods) {
					if sort == "diff" {
						return compare(math.Abs(itemCostDiff(sortidx, a)), math.Abs(itemCostDiff(sortidx, b)), sortdir != "A")
					}
					return compare(math.Abs(itemCostDiffPct(sortidx, a)), math.Abs(itemCostDiffPct(sortidx, b)), sortdir != "A")
				}
			}
			return compare(a.Values[len(b.Values)-1], b.Values[len(b.Values)-1], true)
		})
		for _, item := range queryData.Items {
			if search != "" && !item.Search(search) {
				skipSearch++
				continue
			}

			var maxCostValue float64
			for _, periodValue := range item.Values {
				if math.Abs(periodValue) > maxCostValue {
					maxCostValue = math.Abs(periodValue)
				}
			}
			if mincost > 0 && maxCostValue < float64(mincost) {
				skipMinCost++
				continue
			}

			out.Writeln(`<tr>`)

			out.Writef(`<td align="center">%d</td>`, ct)

			for groupIdx, group := range item.Keys {
				switch gv := group.Value.(type) {
				case cloudcostexplorer.ValueOutput:
					ov, err := gv.Output(r.Context(), newValueContext(fmt.Sprintf("group%d", groupIdx+1), w), uq.Clone())
					if err != nil {
						return fmt.Errorf("error handling custom value: %w", err)
					}
					out.Writef(`<td>%s</td>`, ov)
				default:
					if queryData.Groups[groupIdx].IsGroupFilter {
						gq := uq.Clone().Set(fmt.Sprintf("f%s", queryData.Groups[groupIdx].ID), group.ID)
						// if only one group and filtering by one of its values, change the group to the one with the next priority.
						if len(queryData.Groups) == 1 && queryData.Groups[groupIdx].DefaultPriority > 0 {
							gf, ok := cloud.Parameters().FindByGroupDefaultPriority(queryData.Groups[groupIdx].DefaultPriority + 1)
							if ok {
								gq.Set("group1", gf.ID)
							}
						}
						out.Writef(`<td><a href="%s">%s</a></td>`, gq, group.Value)
					} else {
						out.Writef(`<td>%s</td>`, group.Value)
					}
				}
			}
			for periodIdx, periodValue := range item.Values {
				costClass := ""
				if periodIdx > 0 {
					costDiff, pctCostDiff := itemCostDiffGet(periodIdx, item)

					costClass = "text-danger"
					if costDiff <= 0 {
						costClass = "text-success"
					}
					if showdiff || showdiffpct {
						if showdiff {
							out.Writef(`<td class="%s" align="right">%s</td>`, costClass, cloudcostexplorer.FormatMoney(costDiff))
						}
						if showdiffpct {
							out.Writef(`<td class="%s" align="right">%s%%</td>`, costClass, humanize.CommafWithDigits(pctCostDiff, 2))
						}
						costClass = ""
					}
				}
				out.Writef(`<td class="%s" align="right">%s</td>`, costClass, cloudcostexplorer.FormatMoney(periodValue))
			}

			out.Writeln(`</tr>`)

			ct++
			if limit > 0 && ct > limit {
				isLimit = true
				break
			}
		}

		// DATA END

		if skipSearch > 0 {
			out.Writeln(`<tr>`)
			out.Writef(`<td colspan="%d" align="center">Skipped %s items because of search term '%s'</td>`,
				totalCols,
				humanize.Comma(int64(skipSearch)),
				search)
			out.Writeln(`</tr>`)
		}
		if skipMinCost > 0 {
			out.Writeln(`<tr>`)
			out.Writef(`<td colspan="%d" align="center">Skipped %d rows with absolute cost less than %s [<a href="%s">remove limit</a>]</td>`,
				totalCols,
				skipMinCost, cloudcostexplorer.FormatMoney(float64(mincost)),
				uq.Clone().Set("mincost", "0"))
			out.Writeln(`</tr>`)
		}
		if isLimit {
			out.Writeln(`<tr>`)
			out.Writef(`<td colspan="%d" align="center">Stopped after reaching limit of %s (total was %s) [use "&amp;limit=5000" to increase limit]</td>`,
				totalCols,
				humanize.Comma(int64(limit)),
				humanize.Comma(int64(len(queryData.Items))))
			out.Writeln(`</tr>`)
		}

		out.Writeln(`</tbody></table>`)

		// extra data

		if queryData.ExtraOutput != nil {
			for eo, err := range queryData.ExtraOutput.ExtraOutputs() {
				if err != nil {
					return fmt.Errorf("error handling custom value: %w", err)
				}

				value, err := eo.Output(r.Context(), newValueContext("", w), uq.Clone())
				if err != nil {
					return fmt.Errorf("error handling custom value: %w", err)
				}

				out.Writeln(value)
			}
		}

		if len(periodMatchErrors) > 0 {
			out.Writeln(`<h3>Errors</h3>`)

			out.Writeln(`<ul class="list-group">`)
			for _, perr := range periodMatchErrors {
				out.Writef(`<li class="list-group-item">%s</li>`+"\n", perr.Error())
			}
			out.Writeln(`</ul>`)
		}

		out.BodyEnd()

		out.DocEnd()
		return nil
	})
}
