package ui

import (
	"fmt"

	"github.com/rrgmc/cloudcostexplorer"
)

func SortIcon(fieldIsSorted bool, sortDir string, uq *cloudcostexplorer.URLQuery) string {
	sortIcon := "bi-sort-down"
	sortColor := "link-secondary"
	newSortDir := "D"

	if fieldIsSorted {
		sortColor = "link-primary"
		if sortDir == "A" {
			sortIcon = "bi-sort-up"
			newSortDir = "D"
		} else {
			newSortDir = "A"
		}
	}

	if sortDir != "" {
		uq.Set("sortdir", newSortDir)
	}

	return fmt.Sprintf(`<a class="%s" href="%s"><i class="bi %s"></i></a>`,
		sortColor, uq, sortIcon)
}
