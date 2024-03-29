// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	typeInfoRe      = regexp.MustCompile(`(<br>(&nbsp;)*([^:]+):&nbsp;)[^"{]+`)
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
  <td>&criban=</td>
  <td>{{ .Criban }}</td>
</tr>

<tr>
  <td>&crname=</td>
  <td>{{ .Crname }}</td>
</tr>

<tr>
  <td>&craddr1=</td>
  <td>{{ .Craddr1 }}</td>
</tr>

<tr>
  <td>&craddr2=</td>
  <td>{{ .Craddr2 }}</td>
</tr>

<tr>
  <td>&crpost=</td>
  <td>{{ .Crpost }}</td>
</tr>

<tr>
  <td>&crcity=</td>
  <td>{{ .Crcity }}</td>
</tr>

<tr>
  <td>&crcountry=</td>
  <td>{{ .Crcountry }}</td>
</tr>


<tr>
  <td>&udname=</td>
  <td>{{ .Udname }}</td>
</tr>

<tr>
  <td>&udaddr1=</td>
  <td>{{ .Udaddr1 }}</td>
</tr>

<tr>
  <td>&udaddr2=</td>
  <td>{{ .Udaddr2 }}</td>
</tr>

<tr>
  <td>&udpost=</td>
  <td>{{ .Udpost }}</td>
</tr>

<tr>
  <td>&udcity=</td>
  <td>{{ .Udcity }}</td>
</tr>

<tr>
  <td>&udcountry=</td>
  <td>{{ .Udcountry }}</td>
</tr>

<tr>
  <td>&amount=</td>
  <td>{{ .Amount }}</td>
</tr>


<tr>
  <td>&message=</td>
  <td>{{ .Message }}</td>
</tr>

<tr>
  <td>&udaddrtype=</td>
  <td>{{ .Udaddrtype }}</td>
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

		Udname     string
		Udaddr1    string
		Udaddr2    string
		Udpost     string
		Udcity     string
		Udcountry  string
		Udaddrtype string

		Message string

		Amount string
	}{
		Criban: r.FormValue("criban"),

		Crname:    r.FormValue("crname"),
		Craddr1:   r.FormValue("craddr1"),
		Craddr2:   r.FormValue("craddr2"),
		Crpost:    r.FormValue("crpost"),
		Crcity:    r.FormValue("crcity"),
		Crcountry: r.FormValue("crcountry"),

		Udname:     r.FormValue("udname"),
		Udaddr1:    r.FormValue("udaddr1"),
		Udaddr2:    r.FormValue("udaddr2"),
		Udpost:     r.FormValue("udpost"),
		Udcity:     r.FormValue("udcity"),
		Udcountry:  r.FormValue("udcountry"),
		Udaddrtype: r.FormValue("udaddrtype"),

		Message: r.FormValue("message"),

		Amount: r.FormValue("amount"),
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
		sp = typeInfoRe.ReplaceAllString(sp, "$1")
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
