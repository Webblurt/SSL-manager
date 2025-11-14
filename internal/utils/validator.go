package utils

import (
	"regexp"
)

var domainRegexp = regexp.MustCompile(`^[a-zA-Z0-9-]{1,63}(?:\.[a-zA-Z0-9-]{1,63})*$`)

func IsValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	return domainRegexp.MatchString(domain)
}
