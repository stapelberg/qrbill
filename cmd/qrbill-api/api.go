package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"

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

func logic() error {
	var listen = flag.String("listen", "localhost:9933", "[host]:port to listen on")
	flag.Parse()

	http.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {
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
			format != "html" {
			msg := fmt.Sprintf("format (%q) must be one of png, svg, txt or html", format)
			log.Printf("%s %s", prefix, msg)
			http.Error(w, msg, http.StatusBadRequest)
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
			spew.Fdump(w, qrch.Fill())

		case "html":
			debugHTML(w, r, prefix, qrch)
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, "/qr?format=html", http.StatusFound)
	})
	log.Printf("QR Bill generation URL: http://%s/qr?format=html", *listen)
	return http.ListenAndServe(*listen, nil)
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
