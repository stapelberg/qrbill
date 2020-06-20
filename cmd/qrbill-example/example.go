package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/stapelberg/qrbill"

	_ "net/http/pprof"
)

func ifEmpty(s, alternative string) string {
	if s == "" {
		return alternative
	}
	return s
}

func qrchFromRequest(r *http.Request) *qrbill.QRCH {
	return &qrbill.QRCH{
		CdtrInf: qrbill.QRCHCdtrInf{
			IBAN: ifEmpty(r.FormValue("criban"), "CH0209000000870913543"),
			Cdtr: qrbill.Address{
				AdrTp:            qrbill.AddressTypeStructured,
				Name:             ifEmpty(r.FormValue("crname"), "Legalize it!"),
				StrtNmOrAdrLine1: ifEmpty(r.FormValue("craddr1"), "Quellenstrasse 25"),
				BldgNbOrAdrLine2: ifEmpty(r.FormValue("craddr2"), ""),
				PstCd:            ifEmpty(r.FormValue("crpost"), "8005"),
				TwnNm:            ifEmpty(r.FormValue("crcity"), "Zürich"),
				Ctry:             ifEmpty(r.FormValue("crcountry"), "CH"),
			},
		},
		CcyAmt: qrbill.QRCHCcyAmt{
			Amt: "",
			Ccy: "CHF",
		},
		UltmtDbtr: qrbill.Address{
			AdrTp:            qrbill.AddressTypeStructured,
			Name:             ifEmpty(r.FormValue("udname"), "Michael Stapelberg"),
			StrtNmOrAdrLine1: ifEmpty(r.FormValue("udaddr1"), "Brahmsstrasse 21"),
			BldgNbOrAdrLine2: ifEmpty(r.FormValue("udaddr2"), ""),
			PstCd:            ifEmpty(r.FormValue("udpost"), "8003"),
			TwnNm:            ifEmpty(r.FormValue("udcity"), "Zürich"),
			Ctry:             ifEmpty(r.FormValue("udcountry"), "CH"),
		},
		RmtInf: qrbill.QRCHRmtInf{
			Tp:  "NON", // Reference type
			Ref: "",    // Reference
			AddInf: qrbill.QRCHRmtInfAddInf{
				Ustrd: ifEmpty(r.FormValue("message"), "Spende 6141"),
			},
		},
	}
}

var fieldNameRe = regexp.MustCompile(`<br>(&nbsp;)*([^:]+):`)
var stringLiteralRe = regexp.MustCompile(`"([^"]*)"`)

func qrHandler(format string) http.Handler {
	if format != "png" &&
		format != "svg" &&
		format != "txt" &&
		format != "html" {
		log.Fatalf("BUG: format must be either png, svg, txt or html")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := "[" + r.RemoteAddr + "]"
		log.Printf("%s handling request for %s", prefix, r.URL.Path)
		defer log.Printf("%s done: %s", prefix, r.URL.Path)

		qrch := qrchFromRequest(r)

		bill, err := qrch.Encode()
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var b []byte
		switch format {
		case "png":
			code, err := bill.EncodeToImage()
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var buf bytes.Buffer
			if err := png.Encode(&buf, code); err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b = buf.Bytes()
			w.Header().Add("Content-Type", "image/png")

		case "svg":
			var err error
			b, err = bill.EncodeToSVG()
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Add("Content-Type", "image/svg+xml")

		case "txt":
			w.Header().Add("Content-Type", "text/plain; charset=utf-8")
			spew.Fdump(w, qrch.Fill())

		case "html":
			w.Header().Add("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<html lang="en">
<head>
  <title>QR Bill HTML Debug Page</title>
  <style type="text/css">
.fieldname { font-weight: bold; }
.stringliteral { color: blue; }
  </style>
</head>
<body style="font-family: monospace">
`)
			sp := spew.Sdump(qrch.Fill())
			sp = strings.ReplaceAll(sp, "\n", "<br>")
			sp = strings.ReplaceAll(sp, " ", "&nbsp;")
			sp = stringLiteralRe.ReplaceAllStringFunc(sp, func(stringLiteral string) string {
				return `<span class="stringliteral">` + stringLiteral + "</span>"
			})
			sp = fieldNameRe.ReplaceAllStringFunc(sp, func(fieldName string) string {
				return `<span class="fieldname">` + fieldName + "</span>"
			})
			fmt.Fprintf(w, "%s", sp)
		}

		// TODO: add cache control headers
		if _, err := io.Copy(w, bytes.NewReader(b)); err != nil {
			log.Print(err)
			return
		}
	})
}

func logic() error {
	var listen = flag.String("listen", "localhost:9933", "[host]:port to listen on")
	flag.Parse()
	http.Handle("/qr.png", qrHandler("png"))
	http.Handle("/qr.svg", qrHandler("svg"))
	http.Handle("/qr.txt", qrHandler("txt"))
	http.Handle("/qr.html", qrHandler("html"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		// TODO: add explanation for how to construt a URL
		// e.g. for usage in filemaker web view
		fmt.Fprintf(w, "<ul>")
		fmt.Fprintf(w, `<li>PNG referenz: <a href="/qr.png">qr.png</a>`+"\n")
		fmt.Fprintf(w, `<li>SVG scalable: <a href="/qr.svg">qr.svg</a>`+"\n")
		fmt.Fprintf(w, `<li>debug: <a href="/qr.txt">qr.txt</a>`+"\n")
	})
	log.Printf("listening on http://%s", *listen)
	return http.ListenAndServe(*listen, nil)
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
