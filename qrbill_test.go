package qrbill_test

import (
	"testing"

	"github.com/stapelberg/qrbill"
)

func TestAmountValidation(t *testing.T) {
	for _, tt := range []struct {
		amount     string
		wantAmount string
	}{
		{
			// ensure empty amount values are not modified
			amount:     "",
			wantAmount: "",
		},

		{
			amount:     "50",
			wantAmount: "50.00",
		},

		{
			amount:     "50.3",
			wantAmount: "50.30",
		},

		{
			amount:     "50.32",
			wantAmount: "50.32",
		},

		{
			amount:     "50.32",
			wantAmount: "50.32",
		},

		{
			amount:     "50.000",
			wantAmount: "50.00",
		},

		{
			amount:     "50.339",
			wantAmount: "50.34",
		},

		{
			amount:     "50.331",
			wantAmount: "50.33",
		},

		{
			amount:     "50.-",
			wantAmount: "0.00", // result of invalid input
		},

		{
			amount:     ".30",
			wantAmount: "0.30",
		},

		{
			amount:     ".3",
			wantAmount: "0.30",
		},

		{
			// minimum amount mentioned in the Implementation Guidelines
			amount:     "0.01",
			wantAmount: "0.01",
		},

		{
			// maximum amount mentioned in the Implementation Guidelines
			amount:     "999999999.99",
			wantAmount: "999999999.99",
		},
	} {
		t.Run(tt.amount, func(t *testing.T) {
			qrch := &qrbill.QRCH{
				CdtrInf: qrbill.QRCHCdtrInf{
					IBAN: "CH0209000000870913543",
					Cdtr: qrbill.Address{
						AdrTp:            qrbill.AddressTypeStructured,
						Name:             "Legalize it",
						StrtNmOrAdrLine1: "Quellenstrasse",
						BldgNbOrAdrLine2: "25",
						PstCd:            "8005",
						TwnNm:            "Zürich",
						Ctry:             "CH",
					},
				},
				CcyAmt: qrbill.QRCHCcyAmt{
					Amt: tt.amount,
					Ccy: "CHF",
				},
				UltmtDbtr: qrbill.Address{
					AdrTp:            qrbill.AddressTypeStructured,
					Name:             "Michael Stapelberg",
					StrtNmOrAdrLine1: "Stauffacherstr",
					BldgNbOrAdrLine2: "42",
					PstCd:            "8004",
					TwnNm:            "Zürich",
					Ctry:             "CH",
				},
				RmtInf: qrbill.QRCHRmtInf{
					Tp:  "NON", // Reference type
					Ref: "",    // Reference
					AddInf: qrbill.QRCHRmtInfAddInf{
						Ustrd: "test",
					},
				},
			}

			validated := qrch.Validate()
			if got, want := validated.CcyAmt.Amt, tt.wantAmount; got != want {
				t.Errorf("CcyAmt.Amt = %q, want %q", got, want)
			}
		})
	}
}
