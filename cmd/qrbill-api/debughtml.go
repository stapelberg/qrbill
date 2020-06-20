package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/stapelberg/qrbill"
)

var (
	fieldNameRe     = regexp.MustCompile(`<br>(&nbsp;)*([^:]+):`)
	stringLiteralRe = regexp.MustCompile(`"([^"]*)"`)
	parenRe         = regexp.MustCompile(`\(([^)]+)\)&nbsp;`)
)

var tmpl = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <title>QR Bill HTML Debug Page</title>
  <style type="text/css">

.fieldname { font-weight: bold; }
.stringliteral { color: blue; }
#spews { display: flex; }
#spews div { border: 1px solid black; margin: 1em; padding: 1em; }
.qrch { font-family: monospace; }
th { text-align: left; }
#params tr td:nth-child(2) { color: blue; }
#params tr td:nth-child(1)::before { content: "&"; }
#params tr td:nth-child(1)::after { content: "="; }

  </style>
</head>
<body>
<div id="spews">
<div class="qrch">
<h1>URL parameters</h1>
<table id="params">
<tr>
  <th>Parameter</th>
  <th>Value</th>
</tr>

<tr>
  <td>criban</td>
  <td>{{ .Criban }}</td>
</tr>

<tr>
  <td>crname</td>
  <td>{{ .Crname }}</td>
</tr>

<tr>
  <td>craddr1</td>
  <td>{{ .Craddr1 }}</td>
</tr>

<tr>
  <td>craddr2</td>
  <td>{{ .Craddr2 }}</td>
</tr>

<tr>
  <td>crpost</td>
  <td>{{ .Crpost }}</td>
</tr>

<tr>
  <td>crcity</td>
  <td>{{ .Crcity }}</td>
</tr>

<tr>
  <td>crcountry</td>
  <td>{{ .Crcountry }}</td>
</tr>


<tr>
  <td>udname</td>
  <td>{{ .Udname }}</td>
</tr>

<tr>
  <td>udaddr1</td>
  <td>{{ .Udaddr1 }}</td>
</tr>

<tr>
  <td>udaddr2</td>
  <td>{{ .Udaddr2 }}</td>
</tr>

<tr>
  <td>udpost</td>
  <td>{{ .Udpost }}</td>
</tr>

<tr>
  <td>udcity</td>
  <td>{{ .Udcity }}</td>
</tr>

<tr>
  <td>udcountry</td>
  <td>{{ .Udcountry }}</td>
</tr>

<tr>
  <td>message</td>
  <td>{{ .Message }}</td>
</tr>

</table>

</div>
`))

func debugHTML(w http.ResponseWriter, r *http.Request, prefix string, qrch *qrbill.QRCH) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, struct {
		Criban string

		Crname    string
		Craddr1   string
		Craddr2   string
		Crpost    string
		Crcity    string
		Crcountry string

		Udname    string
		Udaddr1   string
		Udaddr2   string
		Udpost    string
		Udcity    string
		Udcountry string

		Message string
	}{
		Criban: r.FormValue("criban"),

		Crname:    r.FormValue("crname"),
		Craddr1:   r.FormValue("craddr1"),
		Craddr2:   r.FormValue("craddr2"),
		Crpost:    r.FormValue("crpost"),
		Crcity:    r.FormValue("crcity"),
		Crcountry: r.FormValue("crcountry"),

		Udname:    r.FormValue("udname"),
		Udaddr1:   r.FormValue("udaddr1"),
		Udaddr2:   r.FormValue("udaddr2"),
		Udpost:    r.FormValue("udpost"),
		Udcity:    r.FormValue("udcity"),
		Udcountry: r.FormValue("udcountry"),

		Message: r.FormValue("message"),
	})
	if err != nil {
		log.Printf("%s %s", prefix, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", buf.String())

	spew := func(vars ...interface{}) string {
		sp := spew.Sdump(vars...)
		sp = strings.ReplaceAll(sp, "\n", "<br>")
		sp = strings.ReplaceAll(sp, " ", "&nbsp;")
		sp = parenRe.ReplaceAllString(sp, "")
		sp = stringLiteralRe.ReplaceAllStringFunc(sp, func(stringLiteral string) string {
			return `<span class="stringliteral">` + stringLiteral + "</span>"
		})
		sp = fieldNameRe.ReplaceAllStringFunc(sp, func(fieldName string) string {
			return `<span class="fieldname">` + fieldName + "</span>"
		})
		return sp
	}
	fmt.Fprintf(w, `<div class="qrch"><h1>input</h1>%s</div>`, spew(qrch))

	fmt.Fprintf(w, `<div class="qrch"><h1>validated</h1>%s</div>`, spew(qrch.Validate()))

	r.URL.Path = "/qr"
	v := r.URL.Query()
	v.Set("format", "png")
	r.URL.RawQuery = v.Encode()
	fmt.Fprintf(w, `<div class="qrch"><h1>QR Bill</h1><img src="%s" width="200" height="200"></div>`, r.URL.String())
}
