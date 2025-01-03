package cloudcostexplorer

import (
	"iter"
	"maps"
	"net/url"
)

// URLQuery is a URL query builder.
type URLQuery struct {
	path   string
	params map[string]string
}

func NewURLQuery(path string) *URLQuery {
	return &URLQuery{
		path:   path,
		params: make(map[string]string),
	}
}

// Clone clones the query to a new instance.
func (q *URLQuery) Clone() *URLQuery {
	return &URLQuery{
		path:   q.path,
		params: maps.Clone(q.params),
	}
}

func (q *URLQuery) Path() string {
	return q.path
}

func (q *URLQuery) SetPath(path string) *URLQuery {
	q.path = path
	return q
}

func (q *URLQuery) Set(key, value string) *URLQuery {
	q.params[key] = value
	return q
}

func (q *URLQuery) SetFromQuery(query url.Values, keys ...string) *URLQuery {
	for _, key := range keys {
		_ = q.Set(key, query.Get(key))
	}
	return q
}

func (q *URLQuery) Copy(keyFrom, keyTo string) *URLQuery {
	if _, ok := q.params[keyFrom]; ok {
		q.params[keyTo] = q.params[keyFrom]
	}
	return q
}

func (q *URLQuery) Move(keyFrom, keyTo string) *URLQuery {
	if _, ok := q.params[keyFrom]; ok {
		q.params[keyTo] = q.params[keyFrom]
	}
	delete(q.params, keyFrom)
	return q
}

func (q *URLQuery) Swap(key1, key2 string) *URLQuery {
	key1Value, key1Ok := q.params[key1]
	key2Value, key2Ok := q.params[key2]
	if !key1Ok || !key2Ok {
		return q
	}
	q.params[key1] = key2Value
	q.params[key2] = key1Value
	return q
}

func (q *URLQuery) Get(key string) string {
	return q.params[key]
}

func (q *URLQuery) Remove(keys ...string) *URLQuery {
	for _, key := range keys {
		delete(q.params, key)
	}
	return q
}

func (q *URLQuery) String() string {
	if len(q.params) == 0 {
		return q.path + "?"
	}

	values := url.Values{}
	for k, v := range q.params {
		if v == "" {
			continue
		}
		values.Set(k, v)
	}
	return q.path + "?" + values.Encode()
}

func (q *URLQuery) Params() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, v := range q.params {
			if v == "" {
				continue
			}
			if !yield(k, v) {
				return
			}
		}
	}
}
