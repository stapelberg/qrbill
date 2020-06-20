// Package qrbill implements the Swiss QR-bill standard.
//
// More specifically, the most recent standard version at the time of writing
// was the Swiss Payment Standards 2019 Swiss Implementation Guidelines QR-bill
// Version 2.1, to be found at:
//
// https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-en.pdf (English)
// https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-de.pdf (German)
package qrbill

import (
	"image"
	"log"
	"os"
	"strings"

	"github.com/aaronarduino/goqrsvg"
	svg "github.com/ajstarks/svgo"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	// We currently read the swiss cross PNG version.
	_ "image/png"
)

// As per section 4.1: In general:
// Oriented upon the Swiss Implementation Guidelines for Credit Transfers for
// the ISO 20022 Customer Credit Transfer Initiation message (pain.001).

// Section 4.2.1: Character set:
// UTF-8 should be used for encoding

// see also:
// https://github.com/codebude/QRCoder/wiki/Advanced-usage---Payload-generators#317-swissqrcode-iso-20022

const (
	// QRType is an unambiguous indicator for the Swiss QR Code. Fixed value
	// "SPC".
	QRType = "SPC" // Swiss Payments Code

	// Version contains the version of the specifications (Implementation
	// Guidelines) in use on the date on which the Swiss QR Code was
	// created. The first two positions indicate the main version, the following
	// two positions the sub-version. Fixed value of "0200" for Version 2.0.
	Version = "0200" // Version 2.0

	// CodingType is the character set code. Fixed value "1".
	CodingType = "1" // UTF-8 restricted to the Latin character set
)

// AddressType corresponds to AdrTp in ISO20022.
type AddressType string

const (
	AddressTypeStructured AddressType = "S"
	AddressTypeCombined               = "K"
)

// - fixed length: 21 alphanumeric characters
// - only IBANs with CH or LI country code permitted
var iban = "CH0209000000870913543"

type Address struct {
	AdrTp            AddressType
	Name             string // Name, max 70. chars, first name + last name, or company name
	StrtNmOrAdrLine1 string // Street or address line 1
	BldgNbOrAdrLine2 string // Building number or address line 2
	PstCd            string // Postal code, max 16 chars, must be provided without a country code prefix
	TwnNm            string // Town, max. 35 chars
	Ctry             string // Country, two-digit country code according to ISO 3166-1
}

type QRCHHeader struct {
	QRType  string
	Version string
	Coding  string
}

type QRCHCdtrInf struct {
	IBAN string
	Cdtr Address // Creditor
}

type QRCHCcyAmt struct {
	Amt string // Amount
	Ccy string // Currency
}

type QRCHRmtInfAddInf struct {
	Ustrd   string // Unstructured message
	Trailer string // Trailer
}

type QRCHRmtInf struct {
	Tp     string           // Reference type
	Ref    string           // Reference
	AddInf QRCHRmtInfAddInf // Additional information
}

type QRCH struct {
	Header    QRCHHeader  // Header
	CdtrInf   QRCHCdtrInf // Creditor information (Account / Payable to)
	UltmtCdtr Address     // (must not be filled in, for Future Use)
	CcyAmt    QRCHCcyAmt  // Paymount amount information
	UltmtDbtr Address     // Ultimate Debtor
	RmtInf    QRCHRmtInf  // Payment reference
}

func (q *QRCH) QRContents() string {
	return strings.Join([]string{
		q.Header.QRType,
		q.Header.Version,
		q.Header.Coding,

		q.CdtrInf.IBAN,

		string(q.CdtrInf.Cdtr.AdrTp),
		q.CdtrInf.Cdtr.Name,
		q.CdtrInf.Cdtr.StrtNmOrAdrLine1,
		q.CdtrInf.Cdtr.BldgNbOrAdrLine2,
		q.CdtrInf.Cdtr.PstCd,
		q.CdtrInf.Cdtr.TwnNm,
		q.CdtrInf.Cdtr.Ctry,

		string(q.UltmtCdtr.AdrTp),
		q.UltmtCdtr.Name,
		q.UltmtCdtr.StrtNmOrAdrLine1,
		q.UltmtCdtr.BldgNbOrAdrLine2,
		q.UltmtCdtr.PstCd,
		q.UltmtCdtr.TwnNm,
		q.UltmtCdtr.Ctry,

		q.CcyAmt.Amt,
		q.CcyAmt.Ccy,

		string(q.UltmtDbtr.AdrTp),
		q.UltmtDbtr.Name,
		q.UltmtDbtr.StrtNmOrAdrLine1,
		q.UltmtDbtr.BldgNbOrAdrLine2,
		q.UltmtDbtr.PstCd,
		q.UltmtDbtr.TwnNm,
		q.UltmtDbtr.Ctry,

		q.RmtInf.Tp,
		q.RmtInf.Ref,
		q.RmtInf.AddInf.Ustrd,
		q.RmtInf.AddInf.Trailer,
	}, "\n")
}

// https://www.paymentstandards.ch/dam/downloads/qrcodegenerator.java
func generateSwissQrCode(content string) error {
	// generate the qr code from the payload
	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return err
	}

	// overlay the qr code with a Swiss Cross
	combined, err := overlayWithSwissCross(code)
	if err != nil {
		return err
	}
	_ = combined

	return nil
}

func overlayWithSwissCross(code barcode.Barcode) (image.Image, error) {
	// TODO: bundle a swiss cross image
	const swissCrossPath = "/home/michael/go/src/github.com/stapelberg/qrbill/third_party/swiss-cross/CH-Kreuz_7mm/CH-Kreuz_7mm.png"

	// TODO: read swiss cross image
	f, err := os.Open(swissCrossPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	log.Printf("bounds: %+v", m.Bounds())
	return nil, nil
}

func Generate() error {
	log.Printf("hey!")

	content := (&QRCH{
		Header: QRCHHeader{
			QRType:  QRType,
			Version: Version,
			Coding:  CodingType,
		},
		CdtrInf: QRCHCdtrInf{
			IBAN: iban,
			Cdtr: Address{
				AdrTp:            AddressTypeStructured, // CR AddressTyp
				Name:             "Legalize it!",        // CR Name
				StrtNmOrAdrLine1: "Quellenstrasse 25",   // CR Street or address line 1
				BldgNbOrAdrLine2: "",                    // CR Building number or address line 2
				PstCd:            "8005",                // CR Postal code
				TwnNm:            "Zürich",              // CR City
				Ctry:             "CH",                  // CR Country
			},
		},
		CcyAmt: QRCHCcyAmt{
			Amt: "",
			Ccy: "CHF",
		},
		UltmtDbtr: Address{
			"S",                  // UD AddressTyp
			"Michael Stapelberg", // UD Name
			"Brahmsstrasse 21",   // UD Street or address line 1
			"",                   // UD Building number or address line 2
			"8003",               // Postal code
			"Zürich",             // City
			"CH",                 // Country
		},
		RmtInf: QRCHRmtInf{
			Tp:  "NON", // Reference type
			Ref: "",    // Reference
			AddInf: QRCHRmtInfAddInf{
				Ustrd:   "Spende 6141",
				Trailer: "EPD",
			},
		},
	}).QRContents()

	// as per https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-en.pdf, section 5.1:
	// Error correction level M (redundancy of around 15%)

	// TODO: data content must be no more than 997 characters

	// https://www.PaymentStandards.CH/FAQ)
	// TODO: auf version 24 (46mm x 46mm) skalieren

	// version 25 with 117 x 117 modules

	// minimum module size of 0.4mm (recommended for printing)

	// TODO: overlay the swiss cross logo!
	// TODO: verify dimensions when printed

	// TODO: ensure UTF-8
	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return err
	}
	f, err := os.Create("/tmp/code.svg")
	if err != nil {
		return err
	}
	defer f.Close()
	s := svg.New(f)
	qrsvg := goqrsvg.NewQrSVG(code, 5)
	qrsvg.StartQrSVG(s)
	if err := qrsvg.WriteQrSVG(s); err != nil {
		return err
	}
	s.End()
	if err := f.Close(); err != nil {
		return err
	}

	if err := generateSwissQrCode(content); err != nil {
		return err
	}

	/*
		code, err := qrcode.NewWithForcedVersion(content, 25, qrcode.Medium)
		if err != nil {
			return err
		}
		log.Printf("code: %v", code)
		const pixelsPerMillimeter = 10
		return code.WriteFile(-2, "/tmp/code.png")
	*/
	return nil
}
