package aws

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/rrgmc/cloudcostexplorer"
)

type extraOutput struct {
	usageTypeGroups *extraOutputUsageTypeGroups
	tags            *extraOutputTags
}

func (e extraOutput) Close() {
}

func (e extraOutput) ExtraOutputs() iter.Seq2[cloudcostexplorer.ValueOutput, error] {
	return func(yield func(cloudcostexplorer.ValueOutput, error) bool) {
		if e.usageTypeGroups != nil {
			yield(e.usageTypeGroups, nil)
		}
		if e.tags != nil {
			yield(e.tags, nil)
		}
	}
}

type extraOutputUsageTypeGroups struct {
	data extraDataUsageTypeGroups
}

func (e extraOutputUsageTypeGroups) Output(ctx context.Context, vctx cloudcostexplorer.ValueContext, uq *cloudcostexplorer.URLQuery) (string, error) {
	var sb strings.Builder

	if e.data.err != nil {
		_, _ = sb.WriteString(`<h3>Usage type group</h3>`)
		_, _ = sb.WriteString(fmt.Sprintf(`<p>error: %s</p>`, e.data.err.Error()))
		return sb.String(), nil
	}

	isData := false

	for _, dims := range e.data.data {
		if !isData {
			isData = true
			_, _ = sb.WriteString(`<h3>Usage type group</h3>`)
			vctx.Flush()
			_, _ = sb.WriteString(`<ul class="list-group">`)
		}

		tv := dims.Value
		if dims.Unit != "" {
			tv += fmt.Sprintf(" (unit: %s)", dims.Unit)
		}
		if dims.AttributeDump != "" {
			tv += fmt.Sprintf(" [%s]", dims.AttributeDump)
		}
		_, _ = sb.WriteString(fmt.Sprintf(`<li class="list-group-item"><a href="%s">%s</a></li>`+"\n",
			uq.Clone().Set(vctx.FilterParamName("USAGE_TYPE_GROUP"), dims.Value),
			tv))
	}
	if isData {
		_, _ = sb.WriteString(`</ul>`)
	}
	return sb.String(), nil
}

type extraOutputTags struct {
	data extraDataTags
}

func (e extraOutputTags) Output(ctx context.Context, vctx cloudcostexplorer.ValueContext, uq *cloudcostexplorer.URLQuery) (string, error) {
	var sb strings.Builder

	if e.data.err != nil {
		_, _ = sb.WriteString(`<h3>Tags</h3>`)
		_, _ = sb.WriteString(fmt.Sprintf(`<p>error: %s</p>`, e.data.err.Error()))
		return sb.String(), nil
	}

	isData := false

	// TAGS
	for _, tag := range e.data.data {
		if !isData {
			isData = true
			_, _ = sb.WriteString(`<h3>Tags</h3>`)
			vctx.Flush()

			_, _ = sb.WriteString(`<table class="table table-striped table-bordered"><tbody>`)
			_, _ = sb.WriteString(fmt.Sprintf(`<thead><th>Tag</th><th>Values</th></thead><tbody>`))
		}

		_, _ = sb.WriteString(fmt.Sprintf(`<tr><td><a href="%s">%s</a></td><td>`,
			uq.Clone().Set("group2", fmt.Sprintf("TAG%s%s", cloudcostexplorer.DataSeparator, tag.Name)),
			tag.Name))

		isCollapsed := len(tag.Values) > 10
		collapsedID := cloudcostexplorer.RandString(10)

		if isCollapsed {
			_, _ = sb.WriteString(fmt.Sprintf(`<a class="btn btn-primary" data-bs-toggle="collapse" href="#%s" role="button" aria-expanded="false" aria-controls="collapseExample">
    View %d</a>`, collapsedID, len(tag.Values)))
			_, _ = sb.WriteString(fmt.Sprintf(`<div class="collapse" id="%s">`, collapsedID))
		}

		_, _ = sb.WriteString(`<ul class="list-group">`)
		for _, tagValue := range tag.Values {
			tv := tagValue
			if tv == "" {
				tv = "[BLANK]"
			}

			_, _ = sb.WriteString(fmt.Sprintf(`<li class="list-group-item"><a href="%s">%s</a></li>`+"\n",
				uq.Clone().Set(vctx.FilterParamName("TAG"), fmt.Sprintf("%s%s%s", tag.Name, cloudcostexplorer.DataSeparator, tagValue)),
				tv))
		}
		_, _ = sb.WriteString(`</ul>`)

		if isCollapsed {
			_, _ = sb.WriteString(`</div>`)
		}

		_, _ = sb.WriteString(fmt.Sprintf(`</td></tr>`))

		vctx.Flush()
	}

	if isData {
		_, _ = sb.WriteString(`</tbody></table>`)
	}

	return sb.String(), nil
}
