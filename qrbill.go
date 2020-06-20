// Package qrbill implements the Swiss QR-bill standard.
//
// More specifically, the most recent standard version at the time of writing
// was the Swiss Payment Standards 2019 Swiss Implementation Guidelines QR-bill
// Version 2.1, to be found at:
//
// https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-en.pdf (English)
// https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-de.pdf (German)
//
// Note
//
// QRR and SCOR references are not yet implemented.
package qrbill

import (
	"bytes"
	"image"
	"regexp"
	"strings"

	"github.com/aaronarduino/goqrsvg"
	svg "github.com/ajstarks/svgo"
	"github.com/boombuler/barcode/qr"

	// We currently read the swiss cross PNG version.
	_ "image/png"
)

// As per section 4.1: In general:
// Oriented upon the Swiss Implementation Guidelines for Credit Transfers for
// the ISO 20022 Customer Credit Transfer Initiation message (pain.001).

// see also:
// https://github.com/codebude/QRCoder/wiki/Advanced-usage---Payload-generators#317-swissqrcode-iso-20022

const (
	// QRType is an unambiguous indicator for the Swiss QR Code. Fixed value.
	QRType = "SPC" // Swiss Payments Code

	// Version contains the version of the specifications (Implementation
	// Guidelines) in use on the date on which the Swiss QR Code was
	// created. The first two positions indicate the main version, the following
	// two positions the sub-version. Fixed value.
	Version = "0200" // Version 2.0

	// CodingType is the character set code. Fixed value.
	CodingType = "1" // UTF-8 restricted to the Latin character set

	// Trailer is an unambiguous indicator for the end of payment data. Fixed
	// value.
	Trailer = "EPD" // End Payment Data
)

// AddressType corresponds to AdrTp in ISO20022.
type AddressType string

const (
	AddressTypeStructured AddressType = "S"
	AddressTypeCombined               = "K"
)

type Address struct {
	AdrTp            AddressType
	Name             string // Name, max 70. chars, first name + last name, or company name
	StrtNmOrAdrLine1 string // Street or address line 1
	BldgNbOrAdrLine2 string // Building number or address line 2
	PstCd            string // Postal code, max 16 chars, must be provided without a country code prefix
	TwnNm            string // Town, max. 35 chars
	Ctry             string // Country, two-digit country code according to ISO 3166-1
}

func (a Address) Validate() Address {
	c := a

	if v := c.Name; len(v) > 70 {
		c.Name = v[:70]
	}

	if v := c.StrtNmOrAdrLine1; len(v) > 70 {
		c.StrtNmOrAdrLine1 = v[:70]
	}

	if v := c.BldgNbOrAdrLine2; len(v) > 16 {
		c.BldgNbOrAdrLine2 = v[:16]
	}

	if v := c.PstCd; len(v) > 16 {
		c.PstCd = v[:16]
	}

	if v := c.TwnNm; len(v) > 35 {
		c.TwnNm = v[:35]
	}

	return c
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

var (
	nonNumericRe      = regexp.MustCompile(`[^0-9]`)
	nonAlphanumericRe = regexp.MustCompile(`[^A-Za-z0-9]`)
	nonDecimalRe      = regexp.MustCompile(`[^0-9.]`)
)

func (q *QRCH) Validate() *QRCH {
	clone := &QRCH{}
	*clone = *q

	// Fill in all fixed values:
	clone.Header.QRType = QRType
	clone.Header.Version = Version
	clone.Header.Coding = CodingType
	clone.RmtInf.AddInf.Trailer = Trailer

	// TODO(spec): strictly speaking, we need to restrict ourselves only to
	// permitted characters (see below). But, even the example from SIX does not
	// do that (Monatspr_ä_mie):
	// https://www.moneytoday.ch/lexikon/qr-rechnung/

	// 4.3.2 Permitted characters
	// general: only the latin character set is permitted. UTF-8 should be used for encoding
	// numeric: 0-9
	// alphanumeric: A-Z a-z 0-9
	// decimal: 0-9 plus decimal separator .

	// Character Set as per PAIN.001.001.03:
	// https://businessbanking.bankofireland.com/app/uploads/2018/11/OMI015017-Credit-Transfer-PAIN.001.001.03-DIGITALFINAL-VERSION.pdf

	// a b c d e f g h i j k l m n o p q r s t u v w x y z
	// A B C D E F G H I J K L M N O P Q R S T U V W X Y Z
	// / - ? : ( ) . , ' +
	// <space>

	// Enforce all field constraints:

	clone.CdtrInf.IBAN = nonAlphanumericRe.ReplaceAllString(clone.CdtrInf.IBAN, "")

	clone.CdtrInf.Cdtr = clone.CdtrInf.Cdtr.Validate()

	clone.UltmtCdtr = clone.UltmtCdtr.Validate()

	clone.RmtInf.Tp = nonAlphanumericRe.ReplaceAllString(clone.RmtInf.Tp, "")
	if v := clone.RmtInf.Tp; len(v) > 4 {
		clone.RmtInf.Tp = v[:4]
	}

	clone.RmtInf.Ref = nonAlphanumericRe.ReplaceAllString(clone.RmtInf.Ref, "")
	if v := clone.RmtInf.Ref; len(v) > 27 {
		clone.RmtInf.Ref = v[:27]
	}

	if v := clone.RmtInf.AddInf.Ustrd; len(v) > 140 {
		clone.RmtInf.AddInf.Ustrd = v[:140]
	}

	return clone
}

func (q *QRCH) Encode() (*Bill, error) {
	f := q.Validate()
	return &Bill{
		qrcontents: strings.Join([]string{
			f.Header.QRType,
			f.Header.Version,
			f.Header.Coding,

			f.CdtrInf.IBAN,

			string(f.CdtrInf.Cdtr.AdrTp),
			f.CdtrInf.Cdtr.Name,
			f.CdtrInf.Cdtr.StrtNmOrAdrLine1,
			f.CdtrInf.Cdtr.BldgNbOrAdrLine2,
			f.CdtrInf.Cdtr.PstCd,
			f.CdtrInf.Cdtr.TwnNm,
			f.CdtrInf.Cdtr.Ctry,

			string(f.UltmtCdtr.AdrTp),
			f.UltmtCdtr.Name,
			f.UltmtCdtr.StrtNmOrAdrLine1,
			f.UltmtCdtr.BldgNbOrAdrLine2,
			f.UltmtCdtr.PstCd,
			f.UltmtCdtr.TwnNm,
			f.UltmtCdtr.Ctry,

			f.CcyAmt.Amt,
			f.CcyAmt.Ccy,

			string(f.UltmtDbtr.AdrTp),
			f.UltmtDbtr.Name,
			f.UltmtDbtr.StrtNmOrAdrLine1,
			f.UltmtDbtr.BldgNbOrAdrLine2,
			f.UltmtDbtr.PstCd,
			f.UltmtDbtr.TwnNm,
			f.UltmtDbtr.Ctry,

			f.RmtInf.Tp,
			f.RmtInf.Ref,
			f.RmtInf.AddInf.Ustrd,
			f.RmtInf.AddInf.Trailer,
		}, "\n"),
	}, nil
}

type Bill struct {
	qrcontents string
}

func (b *Bill) EncodeToSVG() ([]byte, error) {
	// as per https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-en.pdf, section 5.1:
	// Error correction level M (redundancy of around 15%)

	// Section 4.2.1: Character set:
	// UTF-8 should be used for encoding

	code, err := qr.Encode(b.qrcontents, qr.M, qr.Unicode)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	s := svg.New(&buf)
	qrsvg := goqrsvg.NewQrSVG(code, 5)
	qrsvg.StartQrSVG(s)
	if err := qrsvg.WriteQrSVG(s); err != nil {
		return nil, err
	}

	// TODO: overlay the swiss cross logo!

	s.End()
	return buf.Bytes(), nil
}

func (b *Bill) EncodeToImage() (image.Image, error) {
	return generateSwissQrCode(b.qrcontents)
}
