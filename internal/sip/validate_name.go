package sip

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const timeFormat = "2006-01-02" // YYYY-MM-DD

var (
	// SIP names MUST only use allowed characters: a-z; A-Z; 0-9; dash (-) and underscore (_)
	alphaNumericRX = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	ianusRX        = regexp.MustCompile(`^IANUS\d{4}$`) // IANUS####
	lastSectionRX  = regexp.MustCompile(`^[A-Z]{3}$`)
)

// ValidateName checks that name matches SIP_YYYY-MM-DD_IANUS####_@@@
// where #### is four digits and @@@ is three uppercase letters.
// Example: SIP_2025-10-20_IANUS1234_ABT
func ValidateName(name string) error {
	if !alphaNumericRX.MatchString(name) {
		return errors.New("name contains invalid characters")
	}

	sections := strings.Split(name, "_")
	if len(sections) != 4 {
		return fmt.Errorf("expected 4 sections divided by '_', got: %d", len(sections))
	}

	section1 := sections[0]
	if section1 != "SIP" {
		return fmt.Errorf("expected prefix 'SIP', got: %s", section1)
	}

	section2 := sections[1]
	if _, err := time.Parse(timeFormat, section2); err != nil {
		return fmt.Errorf("section 2 must be a valid date in format YYYY-MM-DD: %w", err)
	}

	section3 := sections[2]
	if !ianusRX.MatchString(section3) {
		return fmt.Errorf("section 3 must be in format IANUS####, got: %s", section3)
	}

	section4 := sections[3]
	if !lastSectionRX.MatchString(section4) {
		return fmt.Errorf("section 4 must be exactly 3 uppercase alphabetic characters, got: %s", section4)
	}

	return nil
}
