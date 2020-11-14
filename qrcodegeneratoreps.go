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
	"fmt"
	"strings"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode/encoder"
)

// renderResultEPS is a copy of renderResult from
// gozxing/qrcode/qrcode_writer.go, adapted to output to SVG.
func renderResultEPS(code *encoder.QRCode, width, height, quietZone int) ([]byte, error) {
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

	// --------------------------------------------------------------------------------

	var eps strings.Builder
	// See postscript language document structuring conventions specification version 3.0
	// https://www-cdf.fnal.gov/offline/PostScript/5001.PDF

	// See Encapsulated PostScript File Format Specification
	// https://www.adobe.com/content/dam/acom/en/devnet/actionscript/articles/5002.EPSF_Spec.pdf

	// EPS files must not have lines of ASCII text that exceed 255 characters,
	// excluding line-termination characters.

	// Lines must be terminated with one of the following combinations:
	// - CR (carriage return, ASCII decimal 13)
	// - LF (line feed, ASCII decimal 10)
	// - CR LF
	// - LF CR

	// BoundingBox parameters are lower-left (llx, lly) and upper-right (urx, ury)
	eps.WriteString("%!PS-Adobe-3.0 EPSF-3.0\n")
	eps.WriteString("%%Creator: https://github.com/stapelberg/qrbill\n")
	// TODO: summarize message and recipient in title
	// TODO: is there a max length for the title?
	eps.WriteString("%%Title: QR-Bill\n")
	eps.WriteString("%%CreationDate: " + time.Now().Format("2006-01-02") + "\n")
	eps.WriteString("%%BoundingBox: 0 0 1265 1265\n")
	eps.WriteString("%%EndComments\n")
	eps.WriteString("/F { rectfill } def\n")

	// Change the application coordinate system to work like the SVG one does,
	// for consistency between the different code paths. See also General
	// Coordinate System Transformation, Page 18, Encapsulated PostScript File
	// Format Specification:
	// https://www.adobe.com/content/dam/acom/en/devnet/actionscript/articles/5002.EPSF_Spec.pdf
	eps.WriteString("0 1265 translate\n")
	eps.WriteString("1 -1 scale\n")

	// Explicitly fill the background with white:
	eps.WriteString("1 1 1 setrgbcolor\n")
	// or 1 setgray?
	eps.WriteString("0 0 1265 1265 F\n")

	// Explicitly set color to black:
	eps.WriteString("0 0 0 setrgbcolor\n")
	// or 0 setgray?

	for inputY, outputY := 0, topPadding; inputY < inputHeight; inputY, outputY = inputY+1, outputY+multiple {
		// Write the contents of this row of the barcode
		for inputX, outputX := 0, leftPadding; inputX < inputWidth; inputX, outputX = inputX+1, outputX+multiple {
			if input.Get(inputX, inputY) == 1 {
				eps.WriteString(fmt.Sprintf("%d %d %d %d F\n", outputX, outputY, multiple, multiple))
			}
		}
	}

	eps.WriteString("549 549 translate\n")

	// overlay an EPS version of the swiss cross
	eps.WriteString("1 1 1 setrgbcolor\n")
	eps.WriteString("0 0 166 166 F\n")

	eps.WriteString("0 0 0 setrgbcolor\n")
	eps.WriteString("12 12 142 142 F\n")

	eps.WriteString("1 1 1 setrgbcolor\n")
	eps.WriteString("36 66 94 28 F\n")
	eps.WriteString("68 34 30 92 F\n")

	eps.WriteString("%%EOF")
	return []byte(eps.String()), nil
}
