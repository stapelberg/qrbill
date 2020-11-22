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

package qrbill

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"strings"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode/encoder"
	"github.com/stapelberg/qrbill/internal/pdf"
)

// Image represents a PDF image object containing a DIN A4-sized page
// scanned with 600dpi (i.e. into 4960x7016 pixels).
type Image struct {
	pdf.Common

	Bounds image.Rectangle
}

// Objects implements Object.
func (i *Image) Objects() []pdf.Object { return []pdf.Object{i} }

// Encode implements Object.
func (i *Image) Encode(w io.Writer, ids map[string]pdf.ObjectID) error {
	_, err := fmt.Fprintf(w, `
%d 0 obj
<<
  /Subtype /Form
  /FormType 1
  /Type /XObject
  /ColorSpace /DeviceGray
  /BBox [0 0 %d %d]
  /Matrix [1 0 0 1 0 0]
  /Resources << /ProcSet [/PDF] >>
  /Length %d
>>
stream
%s
endstream
endobj`,
		int(i.Common.ID),
		i.Bounds.Max.X,
		i.Bounds.Max.Y,
		len(i.Common.Stream),
		i.Common.Stream)
	return err
}

// renderResultPDF is a copy of renderResult from
// gozxing/qrcode/qrcode_writer.go, adapted to output to PDF.
func renderResultPDF(code *encoder.QRCode, width, height, quietZone int) ([]byte, error) {
	input := code.GetMatrix()
	if input == nil {
		return nil, gozxing.NewWriterException("IllegalStateException")
	}
	inputWidth := input.GetWidth()
	inputHeight := input.GetHeight()
	qrWidth := inputWidth + (quietZone * 2)
	qrHeight := inputHeight + (quietZone * 2)
	outputWidth := qrWidth
	if outputWidth < width {
		outputWidth = width
	}
	outputHeight := qrHeight
	if outputHeight < height {
		outputHeight = height
	}

	multiple := outputWidth / qrWidth
	if h := outputHeight / qrHeight; multiple > h {
		multiple = h
	}
	// Padding includes both the quiet zone and the extra white pixels to accommodate the requested
	// dimensions. For example, if input is 25x25 the QR will be 33x33 including the quiet zone.
	// If the requested size is 200x160, the multiple will be 4, for a QR of 132x132. These will
	// handle all the padding from 100x100 (the actual QR) up to 200x160.
	leftPadding := (outputWidth - (inputWidth * multiple)) / 2
	topPadding := (outputHeight - (inputHeight * multiple)) / 2

	_, _ = leftPadding, topPadding

	var codePath strings.Builder
	codePath.WriteString("q\n")

	// Change the application coordinate system to work like the SVG one does,
	// for consistency between the different code paths. See also General
	// Coordinate System Transformation, Page 18, Encapsulated PostScript File
	// Format Specification:
	// https://www.adobe.com/content/dam/acom/en/devnet/actionscript/articles/5002.EPSF_Spec.pdf
	codePath.WriteString("1 0 0 -1 0 1265 cm\n")

	for inputY, outputY := 0, topPadding; inputY < inputHeight; inputY, outputY = inputY+1, outputY+multiple {
		// Write the contents of this row of the barcode
		for inputX, outputX := 0, leftPadding; inputX < inputWidth; inputX, outputX = inputX+1, outputX+multiple {
			if input.Get(inputX, inputY) == 1 {
				codePath.WriteString(fmt.Sprintf("%d %d %d %d re\n",
					outputX, outputY, multiple, multiple))
				//eps.WriteString(fmt.Sprintf("%d %d %d %d F\n", outputX, outputY, multiple, multiple))
			}
		}
	}
	// Fill the whole path at once.
	// This step is crucial:
	// filling individual rectangles results in rendering artifacts
	// in some PDF viewers at some zoom levels.
	// Filling the whole path seems to prevent that entirely.
	codePath.WriteString("0 g\n")
	codePath.WriteString("f\n")

	// overlay a PDF version of the swiss cross
	codePath.WriteString("1 0 0 1 549 549 cm\n")

	codePath.WriteString("1 g\n") // white
	codePath.WriteString("0 0 166 166 re\n")
	codePath.WriteString("f\n")

	codePath.WriteString("0 g\n") // black
	codePath.WriteString("12 12 142 142 re\n")
	codePath.WriteString("f\n")

	codePath.WriteString("1 g\n") // white
	codePath.WriteString("36 66 94 28 re\n")
	codePath.WriteString("f\n")
	codePath.WriteString("68 34 30 92 re\n")
	codePath.WriteString("f\n")

	codePath.WriteString("Q\n")

	kids := []pdf.Object{
		&pdf.Page{
			Common: pdf.Common{ObjectName: "page0"},
			Resources: []pdf.Object{
				&Image{
					Common: pdf.Common{
						ObjectName: "qr",
						Stream:     []byte(codePath.String()),
					},
					Bounds: image.Rect(0, 0, 1265, 1265),
				},
			},
			Parent: "pages",
			Contents: []pdf.Object{
				&pdf.Common{
					ObjectName: "content0",
					//[]byte("q 595.28 0 0 841.89 0.00 0.00 cm /code0 Do Q\n"),
					Stream: []byte(`q
0.12 0 0 0.12 0 0 cm
/qr Do
Q
`),
				},
			},
		},
	}

	doc := &pdf.Catalog{
		Common: pdf.Common{ObjectName: "catalog"},
		Pages: &pdf.Pages{
			Common: pdf.Common{ObjectName: "pages"},
			Kids:   kids,
		},
	}
	info := &pdf.DocumentInfo{
		Common:       pdf.Common{ObjectName: "info"},
		CreationDate: time.Now(),
		Producer:     "https://github.com/stapelberg/qrbill",
		Title:        "QR-Bill",
		// TODO: summarize the qr code in the subject
	}
	var buf bytes.Buffer
	pdfEnc := pdf.NewEncoder(&buf)
	if err := pdfEnc.Encode(doc, info); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
