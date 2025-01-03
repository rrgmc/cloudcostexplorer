package cloudcostexplorer

import (
	"strings"
)

// Item contains the keys and values of a single cost explorer item, based on the query groups.
type Item struct {
	Keys   []ItemKey
	Values []float64
}

func NewItem(keys []ItemKey, periods int) *Item {
	ret := &Item{
		Keys: keys,
	}
	for range periods {
		ret.Values = append(ret.Values, 0.0)
	}
	return ret
}

// Search returns whether the search string is contained on any item key value.
func (i *Item) Search(search string) bool {
	sv := strings.ToLower(search)
	for _, key := range i.Keys {
		if kv, ok := key.Value.(string); ok {
			if strings.Contains(
				strings.ToLower(kv),
				strings.ToLower(sv),
			) {
				return true
			}
		}
	}
	return false
}

// ItemKey is the id and value of one item dimension, like "ID=service, Value=EC2".
type ItemKey struct {
	ID    string
	Value any // can be ItemValue or ValueOutput.
}

// ItemValue is the default item value, a monetary cost.
type ItemValue struct {
	Value float64
}

// DefaultItemKeysHash is the default function for hashing the list of item keys.
func DefaultItemKeysHash(keys []ItemKey) string {
	var values []string
	for _, key := range keys {
		values = append(values, key.ID)
	}
	return DefaultHash(strings.Join(values, "-"))
}
