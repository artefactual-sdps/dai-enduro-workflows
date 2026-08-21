package sip_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/dai-enduro-workflows/internal/sip"
)

func TestValidateName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sipName string
		want    []string
	}{
		"Accepts a valid SIP name": {
			sipName: "SIP_2025-10-20_IANUS1234_ABT",
		},
		"Accepts a leap day": {
			sipName: "SIP_2024-02-29_IANUS0001_XYZ",
		},
		"Accepts IANUS0000 and AAA": {
			sipName: "SIP_2000-01-01_IANUS0000_AAA",
		},
		"Errors when the name is empty": {
			sipName: "",
			want:    []string{"name contains invalid characters"},
		},
		"Errors when the name contains a space": {
			sipName: "SIP_2025-10-20_IANUS1234_ABT extra",
			want:    []string{"name contains invalid characters"},
		},
		"Errors when the name contains a dot": {
			sipName: "SIP_2025-10-20_IANUS1234_ABT.zip",
			want:    []string{"name contains invalid characters"},
		},
		"Errors when the name contains a slash": {
			sipName: "SIP_2025-10-20_IANUS1234_ABT/data",
			want:    []string{"name contains invalid characters"},
		},
		"Errors when there are too few sections": {
			sipName: "SIP_2025-10-20_IANUS1234",
			want:    []string{"expected 4 sections divided by '_', got: 3"},
		},
		"Errors when there are too many sections": {
			sipName: "SIP_2025-10-20_IANUS1234_ABT_X",
			want:    []string{"expected 4 sections divided by '_', got: 5"},
		},
		"Errors when dashes are used instead of underscores": {
			sipName: "SIP-2025-10-20-IANUS1234-ABT",
			want:    []string{"expected 4 sections divided by '_', got: 1"},
		},
		"Errors when the prefix is lowercase": {
			sipName: "sip_2025-10-20_IANUS1234_ABT",
			want:    []string{"expected prefix 'SIP', got: sip"},
		},
		"Errors when the prefix is not SIP": {
			sipName: "DIP_2025-10-20_IANUS1234_ABT",
			want:    []string{"expected prefix 'SIP', got: DIP"},
		},
		"Errors when the date is not a real calendar date": {
			sipName: "SIP_2025-02-29_IANUS1234_ABT",
			want: []string{
				`section 2 must be a valid date in format YYYY-MM-DD: parsing time "2025-02-29": day out of range`,
			},
		},
		"Errors when the date month is out of range": {
			sipName: "SIP_2025-13-01_IANUS1234_ABT",
			want: []string{
				`section 2 must be a valid date in format YYYY-MM-DD: parsing time "2025-13-01": month out of range`,
			},
		},
		"Errors when the date day is out of range": {
			sipName: "SIP_2025-02-30_IANUS1234_ABT",
			want: []string{
				`section 2 must be a valid date in format YYYY-MM-DD: parsing time "2025-02-30": day out of range`,
			},
		},
		"Errors when the date is not zero-padded": {
			sipName: "SIP_2025-1-20_IANUS1234_ABT",
			want: []string{
				`section 2 must be a valid date in format YYYY-MM-DD: parsing time "2025-1-20" as "2006-01-02": cannot parse "1-20" as "01"`,
			},
		},
		"Errors when the date section is empty": {
			sipName: "SIP__IANUS1234_ABT",
			want: []string{
				`section 2 must be a valid date in format YYYY-MM-DD: parsing time "" as "2006-01-02": cannot parse "" as "2006"`,
			},
		},
		"Errors when IANUS has too few digits": {
			sipName: "SIP_2025-10-20_IANUS123_ABT",
			want:    []string{"section 3 must be in format IANUS####, got: IANUS123"},
		},
		"Errors when IANUS has too many digits": {
			sipName: "SIP_2025-10-20_IANUS12345_ABT",
			want:    []string{"section 3 must be in format IANUS####, got: IANUS12345"},
		},
		"Errors when IANUS is lowercase": {
			sipName: "SIP_2025-10-20_ianus1234_ABT",
			want:    []string{"section 3 must be in format IANUS####, got: ianus1234"},
		},
		"Errors when IANUS has no digits": {
			sipName: "SIP_2025-10-20_IANUS_ABT",
			want:    []string{"section 3 must be in format IANUS####, got: IANUS"},
		},
		"Errors when the last section is lowercase": {
			sipName: "SIP_2025-10-20_IANUS1234_abt",
			want:    []string{"section 4 must be exactly 3 uppercase alphabetic characters, got: abt"},
		},
		"Errors when the last section is mixed case": {
			sipName: "SIP_2025-10-20_IANUS1234_ABt",
			want:    []string{"section 4 must be exactly 3 uppercase alphabetic characters, got: ABt"},
		},
		"Errors when the last section is too short": {
			sipName: "SIP_2025-10-20_IANUS1234_AB",
			want:    []string{"section 4 must be exactly 3 uppercase alphabetic characters, got: AB"},
		},
		"Errors when the last section is too long": {
			sipName: "SIP_2025-10-20_IANUS1234_ABTA",
			want:    []string{"section 4 must be exactly 3 uppercase alphabetic characters, got: ABTA"},
		},
		"Errors when the last section contains a digit": {
			sipName: "SIP_2025-10-20_IANUS1234_AB1",
			want:    []string{"section 4 must be exactly 3 uppercase alphabetic characters, got: AB1"},
		},
		"Errors when multiple sections are invalid": {
			sipName: "DIP_2025-13-01_ianus1234_abt",
			want: []string{
				"expected prefix 'SIP', got: DIP",
				`section 2 must be a valid date in format YYYY-MM-DD: parsing time "2025-13-01": month out of range`,
				"section 3 must be in format IANUS####, got: ianus1234",
				"section 4 must be exactly 3 uppercase alphabetic characters, got: abt",
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := sip.ValidateName(tc.sipName)
			if len(tc.want) == 0 {
				assert.Equal(t, 0, len(got))
				return
			}

			assert.DeepEqual(t, got, tc.want)
		})
	}
}
