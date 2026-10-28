package domain

import (
	"net/mail"
	"regexp"
	"strings"
)

// MaxEmailAddressLength is the maximum length of an email address, in characters.
const MaxEmailAddressLength = 254

// hostLabel is a part of the domain name: letters, digits and dashes, without a dash at the start or at the end.
var hostLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// EmailAddress identifies an Admin. It has only ASCII characters, with no spaces around it, and all letters are lowercase.
type EmailAddress string

// NewEmailAddress normalizes and checks an email address.
//
// Only ASCII characters are valid. Then the lowercase rule is the same in Go and in Postgres,
// and the length in bytes is the length in characters. A person with a domain name that has
// other characters uses its ASCII form (punycode).
func NewEmailAddress(raw string) (EmailAddress, error) {
	trimmed := strings.TrimSpace(raw)
	// Check the characters before the change to lowercase: some other characters, for example the Kelvin sign, change to ASCII letters.
	if len(trimmed) > MaxEmailAddressLength || !isASCIIWithoutSpaces(trimmed) {
		return "", ErrInvalidEmailAddress
	}
	address := strings.ToLower(trimmed)
	// ParseAddress also accepts a name before the address, as in "Ann <ann@example.com>". Accept the plain address only.
	parsed, err := mail.ParseAddress(address)
	if err != nil || parsed.Address != address {
		return "", ErrInvalidEmailAddress
	}
	_, host, _ := strings.Cut(address, "@")
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return "", ErrInvalidEmailAddress
	}
	for _, label := range labels {
		if !hostLabel.MatchString(label) {
			return "", ErrInvalidEmailAddress
		}
	}
	return EmailAddress(address), nil
}

func isASCIIWithoutSpaces(s string) bool {
	for i := range len(s) {
		if s[i] <= ' ' || s[i] >= 0x7f {
			return false
		}
	}
	return true
}
