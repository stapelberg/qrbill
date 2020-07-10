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
	"flag"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/mattn/go-isatty"
	"github.com/stapelberg/qrbill"

	_ "net/http/pprof"
)

func ifEmpty(form url.Values, key, alternative string) string {
	if len(form[key]) == 0 {
		return alternative
	}
	return form[key][0]
}

func qrchFromRequest(r *http.Request) *qrbill.QRCH {
	return &qrbill.QRCH{
		CdtrInf: qrbill.QRCHCdtrInf{
			IBAN: ifEmpty(r.Form, "criban", "CH0209000000870913543"),
			Cdtr: qrbill.Address{
				AdrTp:            qrbill.AddressTypeCombined,
				Name:             ifEmpty(r.Form, "crname", "Legalize it!"),
				StrtNmOrAdrLine1: ifEmpty(r.Form, "craddr1", "Quellenstrasse 25"),
				BldgNbOrAdrLine2: ifEmpty(r.Form, "craddr2", "8005 Zürich"),
				PstCd:            ifEmpty(r.Form, "crpost", ""),
				TwnNm:            ifEmpty(r.Form, "crcity", ""),
				Ctry:             ifEmpty(r.Form, "crcountry", "CH"),
			},
		},
		CcyAmt: qrbill.QRCHCcyAmt{
			Amt: ifEmpty(r.Form, "amount", ""),
			Ccy: "CHF",
		},
		UltmtDbtr: qrbill.Address{
			AdrTp:            qrbill.AddressType(ifEmpty(r.Form, "udaddrtype", qrbill.AddressTypeCombined)),
			Name:             ifEmpty(r.Form, "udname", "Michael Stapelberg"),
			StrtNmOrAdrLine1: ifEmpty(r.Form, "udaddr1", "Stauffacherstr 42"),
			BldgNbOrAdrLine2: ifEmpty(r.Form, "udaddr2", "8004 Zürich"),
			PstCd:            ifEmpty(r.Form, "udpost", ""),
			TwnNm:            ifEmpty(r.Form, "udcity", ""),
			Ctry:             ifEmpty(r.Form, "udcountry", "CH"),
		},
		RmtInf: qrbill.QRCHRmtInf{
			Tp:  "NON", // Reference type
			Ref: "",    // Reference
			AddInf: qrbill.QRCHRmtInfAddInf{
				Ustrd: ifEmpty(r.Form, "message", "Spende 420"),
			},
		},
	}
}

// Overridden in api_gokrazy.go
var defaultListenAddress = "localhost:9933"

func logic() error {
	var listen = flag.String("listen", defaultListenAddress, "[host]:port to listen on")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {
		prefix := "[" + r.RemoteAddr + "]"
		format := r.FormValue("format")
		log.Printf("%s handling request for %s, format=%s", prefix, r.URL.Path, format)
		defer log.Printf("%s request completed (%s)", prefix, r.URL.Path)

		if format == "" {
			msg := fmt.Sprintf("no ?format= parameter specified. Try %s",
				"http://"+*listen+"/qr?format=html")
			log.Printf("%s %s", prefix, msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		if format != "png" &&
			format != "svg" &&
			format != "txt" &&
			format != "html" &&
			format != "wv" {
			msg := fmt.Sprintf("format (%q) must be one of png, svg, txt or html", format)
			log.Printf("%s %s", prefix, msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			log.Printf("%s %s", prefix, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		qrch := qrchFromRequest(r)

		bill, err := qrch.Encode()
		if err != nil {
			log.Printf("%s %s", prefix, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var b []byte
		switch format {
		case "png":
			code, err := bill.EncodeToImage()
			if err != nil {
				log.Printf("%s %s", prefix, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var buf bytes.Buffer
			if err := png.Encode(&buf, code); err != nil {
				log.Printf("%s %s", prefix, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b = buf.Bytes()
			w.Header().Add("Content-Type", "image/png")

		case "svg":
			var err error
			b, err = bill.EncodeToSVG()
			if err != nil {
				log.Printf("%s %s", prefix, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Add("Content-Type", "image/svg+xml")

		case "txt":
			w.Header().Add("Content-Type", "text/plain; charset=utf-8")
			spew.Fdump(w, qrch.Validate())

		case "html":
			debugHTML(w, r, prefix, qrch)

		case "wv":
			w.Header().Add("Content-Type", "text/html; charset=utf-8")

			r.URL.Path = "/qr"
			v := r.URL.Query()
			v.Set("format", "png")
			r.URL.RawQuery = v.Encode()

			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<style type="text/css">
img {
width: 100vw;
height: 100vh;
}
body {
margin: 0; padding: 0;
}
</style>
</head>
<body>
<img src="%s">
</body>
</html>`, r.URL.String())

		}

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
		// […] this alone is the only directive you need in preventing cached
		// responses on modern browsers.
		w.Header().Add("Cache-Control", "no-store")

		if _, err := io.Copy(w, bytes.NewReader(b)); err != nil {
			log.Printf("%s %s", prefix, err)
			return
		}
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `User-agent: *
Disallow: /
`)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, "/qr?format=html", http.StatusFound)
	})

	if flag.NArg() > 0 {
		srv := httptest.NewServer(mux)
		defer srv.Close()
		for _, arg := range flag.Args() {
			u, err := url.Parse(arg)
			if err != nil {
				return err
			}
			u.Host = strings.TrimPrefix(srv.URL, "http://")
			resp, err := srv.Client().Get(u.String())
			if err != nil {
				return err
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "text/") &&
				isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Fprintf(os.Stderr, "not writing raw image data to terminal, did you forget to redirect the output?\n")
				os.Exit(2)
			}
			if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
				return err
			}
		}
		return nil
	}

	log.Printf("QR Bill generation URL: http://%s/qr?format=html", *listen)
	return http.ListenAndServe(*listen, mux)
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
