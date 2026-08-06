package whatsapp

import (
	"fmt"
	"regexp"
	"strings"
)

var colombianMobilePattern = regexp.MustCompile(`^\+573\d{9}$`)
var basicE164Pattern = regexp.MustCompile(`^\+?\d{7,15}$`)

func CanonicalizeE164(raw string) (string, error) {
	cleaned := strings.TrimSpace(raw)

	if cleaned == "" {
		return "", fmt.Errorf("phone number is empty")
	}

	if !strings.HasPrefix(cleaned, "+") {
		cleaned = "+" + cleaned
	}

	if !basicE164Pattern.MatchString(cleaned) {
		return cleaned, fmt.Errorf("phone number %s does not match basic E.164 pattern", cleaned)
	}

	if !colombianMobilePattern.MatchString(cleaned) {
		return cleaned, fmt.Errorf("phone number %s is not a Colombian mobile number (expected +573XXXXXXXXX)", cleaned)
	}

	return cleaned, nil
}
