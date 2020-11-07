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
	"image"
	"image/draw"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/common"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/makiuchi-d/gozxing/qrcode/decoder"
)

// This is a port of the Java 1.7 reference example from paymentstandards.ch:
// https://www.paymentstandards.ch/dam/downloads/qrcodegenerator.java
//
// The priority was to write idiomatic Go code first, and match the reference
// example as good as possible second.

const (
	swissCrossEdgeSidePx = 166

	swissCrossEdgeSideMm = 7

	// The edge length of the qrcode inclusive its white border.
	qrCodeEdgeSideMm = 42 + 13

	qrCodeEdgeSidePx = swissCrossEdgeSidePx / swissCrossEdgeSideMm * qrCodeEdgeSideMm
)

func generateSwissQrCode(payload string) (image.Image, error) {
	// generate the qr code from the payload
	qrCodeImage, err := generateQrCodeImage(payload)
	if err != nil {
		return nil, err
	}

	// overlay the qr code with a Swiss Cross
	return overlayWithSwissCross(qrCodeImage)
}

func qrEncodeHints() map[gozxing.EncodeHintType]interface{} {
	return map[gozxing.EncodeHintType]interface{}{
		// as per https://www.paymentstandards.ch/dam/downloads/ig-qr-bill-en.pdf, section 5.1:
		// Error correction level M (redundancy of around 15%)
		gozxing.EncodeHintType_ERROR_CORRECTION: decoder.ErrorCorrectionLevel_M,

		// Section 4.2.1: Character set:
		// UTF-8 should be used for encoding
		gozxing.EncodeHintType_CHARACTER_SET: common.CharacterSetECI_UTF8,
	}
}

func generateQrCodeImage(payload string) (image.Image, error) {
	matrix, err := qrcode.NewQRCodeWriter().Encode(
		payload,                       // contents
		gozxing.BarcodeFormat_QR_CODE, // format
		qrCodeEdgeSidePx,              // width
		qrCodeEdgeSidePx,              // height
		qrEncodeHints())               // hints
	if err != nil {
		return nil, err
	}
	return matrix, nil
}

func overlayWithSwissCross(qrCodeImage image.Image) (image.Image, error) {
	b := swisscross["third_party/swiss-cross/CH-Kreuz_7mm/CH-Kreuz_7mm.png"]
	swissCrossImage, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	combinedQrCodeImage := image.NewRGBA(qrCodeImage.Bounds())

	{
		sr := qrCodeImage.Bounds() // source rect
		destRect := image.Rectangle{image.Point{0, 0}, sr.Size()}
		draw.Draw(combinedQrCodeImage, destRect, qrCodeImage, sr.Min, draw.Src)
	}

	{
		sr := swissCrossImage.Bounds() // source rect
		const swissCrossPosition = (qrCodeEdgeSidePx / 2) - (swissCrossEdgeSidePx / 2)
		destPoint := image.Point{
			X: swissCrossPosition,
			Y: swissCrossPosition,
		}
		// Convert the source image bounds into the destination image’s coordinate
		// space:
		destRect := image.Rectangle{
			destPoint,
			destPoint.Add(sr.Size()),
		}
		draw.Draw(combinedQrCodeImage, destRect, swissCrossImage, sr.Min, draw.Src)
	}
	return combinedQrCodeImage, nil
}
