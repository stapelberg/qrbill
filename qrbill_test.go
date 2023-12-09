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
						AdrTp:            qrbill.AddressTypeCombined,
						Name:             "Legalize it",
						StrtNmOrAdrLine1: "Quellenstrasse 25",
						BldgNbOrAdrLine2: "8005 Zürich",
						Ctry:             "CH",
					},
				},
				CcyAmt: qrbill.QRCHCcyAmt{
					Amt: tt.amount,
					Ccy: "CHF",
				},
				UltmtDbtr: qrbill.Address{
					AdrTp:            qrbill.AddressTypeCombined,
					Name:             "Michael Stapelberg",
					StrtNmOrAdrLine1: "Stauffacherstr 42",
					BldgNbOrAdrLine2: "8004 Zürich",
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
