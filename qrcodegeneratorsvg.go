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

	svg "github.com/ajstarks/svgo"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode/encoder"
)

// renderResultSVG is a copy of renderResult from
// gozxing/qrcode/qrcode_writer.go, adapted to output to SVG.
func renderResultSVG(code *encoder.QRCode, width, height, quietZone int) ([]byte, error) {
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

	var buf bytes.Buffer
	s := svg.New(&buf)
	s.Start(outputWidth, outputHeight)
	s.Rect(0, 0, outputWidth, outputHeight, "fill:white;stroke:white")

	s.Group(`shape-rendering="crispEdges"`)

	for inputY, outputY := 0, topPadding; inputY < inputHeight; inputY, outputY = inputY+1, outputY+multiple {
		// Write the contents of this row of the barcode
		for inputX, outputX := 0, leftPadding; inputX < inputWidth; inputX, outputX = inputX+1, outputX+multiple {
			if input.Get(inputX, inputY) == 1 {
				s.Rect(outputX, outputY, multiple, multiple, "fill:black;stroke:none;")
			}
		}
	}

	s.Gend()

	s.End()
	return buf.Bytes(), nil
}
