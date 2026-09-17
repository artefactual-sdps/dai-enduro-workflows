package sip

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const timeFormat = "2006-01-02" // YYYY-MM-DD

var (
	// SIP names MUST only use allowed characters: a-z; A-Z; 0-9; dash (-) and underscore (_)
	alphaNumericRX = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	ianusRX        = regexp.MustCompile(`(?i)^[a-z0-9]{4}$`) // exactly 4 alphanumeric characters, case insensitive
	lastSectionRX  = regexp.MustCompile(`^[A-Z]{3}$`)
)

// ValidateName checks that name matches SIP_YYYY-MM-DD_####_@@@
// where #### is exactly 4 alphanumeric characters (case insensitive)
// and @@@ is three uppercase letters.
// Example: SIP_2025-10-20_A1B2_ABT
func ValidateName(name string) []string {
	validationErrors := []string{}
	if !alphaNumericRX.MatchString(name) {
		validationErrors = append(validationErrors, "name contains invalid characters")
		return validationErrors
	}

	sections := strings.Split(name, "_")
	if len(sections) != 4 {
		msg := fmt.Sprintf("expected 4 sections divided by '_', got: %d", len(sections))
		validationErrors = append(validationErrors, msg)
		return validationErrors
	}

	section1 := sections[0]
	if section1 != "SIP" {
		msg := fmt.Sprintf("expected prefix 'SIP', got: %s", section1)
		validationErrors = append(validationErrors, msg)
	}

	section2 := sections[1]
	if _, err := time.Parse(timeFormat, section2); err != nil {
		msg := fmt.Sprintf("section 2 must be a valid date in format YYYY-MM-DD: %s", err.Error())
		validationErrors = append(validationErrors, msg)
	}

	section3 := sections[2]
	if !ianusRX.MatchString(section3) {
		msg := fmt.Sprintf("section 3 must be exactly 4 alphanumeric characters, got: %s", section3)
		validationErrors = append(validationErrors, msg)
	}

	section4 := sections[3]
	if !lastSectionRX.MatchString(section4) {
		msg := fmt.Sprintf("section 4 must be exactly 3 uppercase alphabetic characters, got: %s", section4)
		validationErrors = append(validationErrors, msg)
	}

	return validationErrors
}
