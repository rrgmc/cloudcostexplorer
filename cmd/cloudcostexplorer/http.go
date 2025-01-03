package main

import (
	"net/http"
	"strconv"
)

func HTTPQueryValueGet(r *http.Request, param string) (string, bool) {
	if v, ok := r.URL.Query()[param]; ok && len(v) > 0 {
		return v[0], true
	}
	return "", false
}

func HTTPQueryStringValue(r *http.Request, param, def string) (string, bool) {
	if v, ok := HTTPQueryValueGet(r, param); ok {
		return v, true
	}
	return def, false
}

func HTTPQueryIntValue(r *http.Request, param string, def int) (int, bool) {
	if v, ok := HTTPQueryValueGet(r, param); ok {
		v2, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return int(v2), true
		}
	}
	return def, false
}

func HTTPQueryBoolValue(r *http.Request, param string, def bool) (value bool, paramExists bool) {
	if v, ok := HTTPQueryValueGet(r, param); ok {
		v2, err := strconv.ParseBool(v)
		if err == nil {
			return v2, true
		}
	}
	return def, false
}
