package ui

import (
	"fmt"
	"html"
	"net/http"

	"github.com/rrgmc/cloudcostexplorer"
)

type HTTPOutput struct {
	w   http.ResponseWriter
	err error
}

func NewHTTPOutput(w http.ResponseWriter) *HTTPOutput {
	return &HTTPOutput{w: w}
}

func (out *HTTPOutput) Write(s string) {
	if out.err != nil {
		return
	}
	_, out.err = out.w.Write([]byte(s))
}

func (out *HTTPOutput) Writeln(s string) {
	out.Write(s + "\n")
}

func (out *HTTPOutput) Writef(format string, args ...any) {
	out.Write(fmt.Sprintf(format, args...))
}

func (out *HTTPOutput) DocBegin(title string) {
	out.Writef(`<!doctype html>
<html lang="en">
<head>
	<title>%s</title>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">

	<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.0.2/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-EVSTQN3/azprG1Anm3QDgpJLIm9Nao0Yz1ztcQTwFspd3yD65VohhpuuCOmLASjC" crossorigin="anonymous">
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css">
</head>
<body style="padding-top: 8rem !important">

<div class="container container-fluid">
`, html.EscapeString(title))
}

func (out *HTTPOutput) DocEnd() {
	out.Write(`
</div>
<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.0.2/dist/js/bootstrap.bundle.min.js" integrity="sha384-MrcW6ZMFYlzcLA8Nl+NtUVF0sA7MsXsP1UyJoMp4YLEuNSfAP+JcXn/tWtIaxVXM" crossorigin="anonymous"></script>
</body></html>`)
}

func (out *HTTPOutput) NavBegin(homeLink string) {
	out.Writef(`<nav class="navbar navbar-expand-lg fixed-top navbar-light bg-light">
  <div class="container-fluid"><a class="navbar-brand" href="%s">Home</a><div class="collapse navbar-collapse">`, homeLink)
}

func (out *HTTPOutput) NavEnd() {
	out.Writeln(`</div></div></nav>`)
}

func (out *HTTPOutput) NavSearch(search string, uq *cloudcostexplorer.URLQuery) {
	sq := uq.Clone().Remove("search")

	class := ""
	if search != "" {
		class = " bg-secondary text-light"
	}

	out.Writef(`    <form class="d-flex ms-2" method="GET" action="%s">
      <input class="form-control me-2%s" name="search" value="%s" type="search" placeholder="Search" aria-label="Search">
      <button class="btn btn-outline-success" type="submit">Search</button>`, sq.Path(), class, html.EscapeString(search))
	for pn, pv := range sq.Params() {
		out.Writef(`<input type="hidden" name="%s" value="%s">`, pn, pv)
	}
	out.Writeln(`</form>`)
}

func (out *HTTPOutput) NavMenuBegin() {
	out.Writeln(`<ul class="navbar-nav me-auto mb-2 mb-lg-0">`)
}

func (out *HTTPOutput) NavMenuEnd() {
	out.Writeln(`</ul>`)
}

func (out *HTTPOutput) NavDropdownBegin(title string) {
	out.Writef(`<li class="nav-item dropdown">          
<a class="nav-link dropdown-toggle" href="#" role="button" data-bs-toggle="dropdown" aria-expanded="false">
            %s
          </a><ul class="dropdown-menu">`, title)
}

func (out *HTTPOutput) NavDropdownHeader(title string) {
	out.Writef(`<li><h6 class="dropdown-header">%s</h6></li>`, title)
}

func (out *HTTPOutput) NavDropdownDivider() {
	out.Writeln(`<li><hr class="dropdown-divider"></li>`)
}

func (out *HTTPOutput) NavDropdownItem(title string, url string) {
	out.Writef(`<li><a class="dropdown-item" href="%s">%s</a></li>`, url, title)
}

func (out *HTTPOutput) NavDropdownEnd() {
	out.Writeln(`</ul></li>`)
}

func (out *HTTPOutput) NavText(title, value string) {
	out.Writef(`<span class="navbar-text ms-2"><strong>%s:</strong> %s</span>`, title, value)
}

func (out *HTTPOutput) NavTextCustom(title, value string) {
	out.Writef(`<span class="navbar-text ms-2">%s %s</span>`, title, value)
}

func (out *HTTPOutput) BodyBegin() {
	out.Writeln(`  <div class="row">
    <div class="col">`)
}

func (out *HTTPOutput) BodyEnd() {
	out.Writeln(`</div></div>`)
}

type HTTPHandlerWithError func(http.ResponseWriter, *http.Request) error

func (h HTTPHandlerWithError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
