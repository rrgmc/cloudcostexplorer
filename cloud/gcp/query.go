package gcp

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/invzhi/timex"
	"github.com/rrgmc/cloudcostexplorer"
	"google.golang.org/api/iterator"
)

func (c *Cloud) Query(ctx context.Context, options ...cloudcostexplorer.QueryOption) iter.Seq2[cloudcostexplorer.CloudQueryItem, error] {
	return func(yield func(cloudcostexplorer.CloudQueryItem, error) bool) {
		optns, err := cloudcostexplorer.ParseQueryOptions(options...)
		if err != nil {
			yield(cloudcostexplorer.CloudQueryItem{}, err)
			return
		}

		var useResourceTable bool

		fieldsAdd := ""
		joinAdd := ""
		whereAdd := ""
		havingAdd := ""
		labelqueryadd := "LEFT "
		// if labelexclusive {
		// 	labelqueryadd = " "
		// }
		var groupFieldsAdd []string

		nstart, nend := cloudcostexplorer.TimeStartEnd(optns.Start, optns.End)

		queryParameters := []bigquery.QueryParameter{
			{Name: "start", Value: nstart.Format(time.RFC3339)},
			{Name: "end", Value: nend.Format(time.RFC3339)},
		}

		for _, filter := range optns.Filters {
			switch filter.ID {
			case "SERVICE":
				whereAdd += " AND service.id = @service"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "service",
					Value: filter.Value,
				})
			case "REGION":
				whereAdd += " AND location.region = @region"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "region",
					Value: filter.Value,
				})
			case "PROJECT":
				whereAdd += " AND project.id = @project"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "project",
					Value: filter.Value,
				})
			case "SKU":
				whereAdd += " AND sku.id = @sku"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "sku",
					Value: filter.Value,
				})
			case "RESOURCE":
				whereAdd += " AND resource.global_name = @resource"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "resource",
					Value: filter.Value,
				})
			case "LABEL":
				useResourceTable = true
				lkey, lval, _ := strings.Cut(filter.Value, cloudcostexplorer.DataSeparator)
				joinAdd += fmt.Sprintf(` JOIN UNNEST(labels) as filter_labels ON filter_labels.key = "%s"`, lkey)
				whereAdd += " AND filter_labels.value = @label_value"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "label_value",
					Value: lval,
				})
			case "SYSLABEL":
				useResourceTable = true
				lkey, lval, _ := strings.Cut(filter.Value, cloudcostexplorer.DataSeparator)
				joinAdd += fmt.Sprintf(` JOIN UNNEST(system_labels) as filter_system_labels ON filter_system_labels.key = "%s"`, lkey)
				whereAdd += " AND filter_system_labels.value = @system_label_value"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "system_label_value",
					Value: lval,
				})
			case "TAGS":
				useResourceTable = true
				lkey, lval, _ := strings.Cut(filter.Value, cloudcostexplorer.DataSeparator)
				joinAdd += fmt.Sprintf(` JOIN UNNEST(tags) as filter_tags ON filter_tags.key = "%s"`, lkey)
				whereAdd += " AND filter_tags.value = @tags_value"
				queryParameters = append(queryParameters, bigquery.QueryParameter{
					Name:  "tags_value",
					Value: lval,
				})
			default:
				yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("unknown filter: %s", filter.ID))
				return
			}
		}

		if optns.GroupByDate {
			fieldsAdd += ", DATE(usage_start_time, 'UTC') as usage_date"
			groupFieldsAdd = append(groupFieldsAdd, "usage_date")
		}

		for gidx, group := range optns.Groups {
			kgroup, kok := c.parameters.FindById(group.ID)
			if !kok || !kgroup.IsGroup {
				yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("invalid group '%s'", group.ID))
				return
			}

			kname := fmt.Sprintf("key%d", gidx+1)
			kdescname := fmt.Sprintf("key%ddesc", gidx+1)

			switch group.ID {
			case "SERVICE":
				fieldsAdd += fmt.Sprintf(", service.id AS %s, service.description as %s", kname, kdescname)
				groupFieldsAdd = append(groupFieldsAdd, "service.id, service.description")
			case "REGION":
				fieldsAdd += fmt.Sprintf(", location.region AS %s, location.region as %s", kname, kdescname)
				groupFieldsAdd = append(groupFieldsAdd, "location.region, location.region")
			case "PROJECT":
				fieldsAdd += fmt.Sprintf(", project.id as %s, project.name as %s", kname, kdescname)
				groupFieldsAdd = append(groupFieldsAdd, "project.id, project.name")
			case "SKU":
				fieldsAdd += fmt.Sprintf(", sku.id as %s, sku.description as %s", kname, kdescname)
				groupFieldsAdd = append(groupFieldsAdd, "sku.id, sku.description")
			case "COSTTYPE":
				fieldsAdd += fmt.Sprintf(", cost_type as %s, cost_type as %s", kname, kdescname)
				groupFieldsAdd = append(groupFieldsAdd, "cost_type")
			case "LABEL":
				useResourceTable = true
				if group.Data == "" {
					fieldsAdd += fmt.Sprintf(", '' as %s, TO_JSON_STRING(labels) as %s", kname, kdescname)
					groupFieldsAdd = append(groupFieldsAdd, "TO_JSON_STRING(labels)")
				} else {
					joinAdd += fmt.Sprintf(` %sJOIN UNNEST(labels) as group_labels ON group_labels.key = "%s"`, labelqueryadd, group.Data)
					fieldsAdd += fmt.Sprintf(", group_labels.value as %s, group_labels.value as %s", kname, kdescname)
					groupFieldsAdd = append(groupFieldsAdd, "group_labels.value")
				}
			case "SYSLABEL":
				useResourceTable = true
				if group.Data == "" {
					fieldsAdd += fmt.Sprintf(", '' as %s, TO_JSON_STRING(system_labels) as %s", kname, kdescname)
					groupFieldsAdd = append(groupFieldsAdd, "TO_JSON_STRING(system_labels)")
				} else {
					joinAdd += fmt.Sprintf(` %sJOIN UNNEST(system_labels) as group_system_labels ON group_system_labels.key = "%s"`, labelqueryadd, group.Data)
					fieldsAdd += fmt.Sprintf(", group_system_labels.value as %s, group_system_labels.value as %s", kname, kdescname)
					groupFieldsAdd = append(groupFieldsAdd, "group_system_labels.value")
				}
			case "TAGS":
				useResourceTable = true
				if group.Data == "" {
					fieldsAdd += fmt.Sprintf(", '' as %s, TO_JSON_STRING(tags) as %s", kname, kdescname)
					groupFieldsAdd = append(groupFieldsAdd, "TO_JSON_STRING(tags)")
				} else {
					joinAdd += fmt.Sprintf(` %sJOIN UNNEST(tags) as group_tags ON group_tags.key = "%s"`, labelqueryadd, group.Data)
					fieldsAdd += fmt.Sprintf(", group_tags.value as %s, group_tags.value as %s", kname, kdescname)
					groupFieldsAdd = append(groupFieldsAdd, "group_tags.value")
				}
			case "RESOURCE":
				useResourceTable = true
				fieldsAdd += fmt.Sprintf(", resource.global_name as %s, resource.name as %s", kname, kdescname)
				groupFieldsAdd = append(groupFieldsAdd, "resource.global_name, resource.name")
			default:
				yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("unknown filter: %s", fmt.Errorf("unknown group: %s", group)))
				return
			}
		}

		tableName := c.defaultTableName
		if useResourceTable {
			tableName = c.resourceTableName
		}

		query := fmt.Sprintf(`select
    SUM(cost)
    + SUM(IFNULL((SELECT SUM(c.amount)
                  FROM UNNEST(credits) c), 0))
    AS total %s
FROM 
	%s
	%s
WHERE
    usage_start_time >= @start AND usage_start_time <= @end
	%s
GROUP BY %s
%s
`, fieldsAdd, tableName, joinAdd, whereAdd, strings.Join(groupFieldsAdd, ", "), havingAdd)

		servicesQuery := c.bigQueryClient.Query(query)

		servicesQuery.Parameters = queryParameters

		servicesIter, err := servicesQuery.Read(ctx)
		if err != nil {
			yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("error querying BigQuery: %w", err))
			return
		}

		var row map[string]bigquery.Value
		for {
			err := servicesIter.Next(&row)
			if err == iterator.Done {
				break
			}
			if err != nil {
				yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("error iterating BigQuery: %w", err))
				return
			}

			cost := row["total"].(float64)

			var itemKeys []cloudcostexplorer.ItemKey
			for groupIdx, group := range optns.Groups {
				idFieldname := fmt.Sprintf("key%d", groupIdx+1)
				descFieldname := fmt.Sprintf("key%ddesc", groupIdx+1)

				keyValue := bigQueryStringValue(row, descFieldname)

				key := cloudcostexplorer.ItemKey{
					ID:    bigQueryStringValue(row, idFieldname),
					Value: keyValue,
				}
				if key.ID == "" {
					key.ID = keyValue
				}
				keyWasBlank := false
				if key.ID == "" {
					keyWasBlank = true
					key.ID = c.blankKeyValue
					key.Value = cloudcostexplorer.EmptyValue{}
				}

				switch group.ID {
				case "LABEL", "SYSLABEL", "TAGS":
					// if there is a group filter only one value is shown
					if group.Data == "" {
						key.Value = NewLabelValue(group.ID, fmt.Sprint(key.Value))
					}
				case "RESOURCE":
					if !keyWasBlank && keyValue == "" {
						key.Value = key.ID
					}
				}

				itemKeys = append(itemKeys, key)

				if ks, ok := key.Value.(string); ok {
					c.setParameterValue(group.ID, key.ID, ks)
				}
			}

			var itemDate timex.Date
			if optns.GroupByDate {
				itemDate = cloudcostexplorer.FromCivilDate(row["usage_date"].(civil.Date))
			}

			if !yield(cloudcostexplorer.CloudQueryItem{
				Date:  itemDate,
				Keys:  itemKeys,
				Value: cost,
			}, nil) {
				return
			}
		}
	}
}

func (c *Cloud) QueryExtraOutput(ctx context.Context, extraData []cloudcostexplorer.QueryExtraData) cloudcostexplorer.QueryExtraOutput {
	return nil
}
