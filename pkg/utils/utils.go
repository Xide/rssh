package utils

import (
	"regexp"
	"strings"
)

// Min ...
func Min(x, y uint16) uint16 {
	if x < y {
		return x
	}
	return y
}

// Max ...
func Max(x, y uint16) uint16 {
	if x > y {
		return x
	}
	return y
}

// IsValidDomain : Taken from https://www.socketloop.com/tutorials/golang-use-regular-expression-to-validate-domain-name
// Does not validate the tld, so that it can be used with .localhost domains.
func IsValidDomain(d string) bool {
	re := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30})$`)
	return re.MatchString(d)
}

// SplitParts Splits {"a,b", "c"} into {"a", "b", "c"}
// Temporary fix (hopefully) because Cobra doesn't
// handle separators if they are not followed by a whitespace.
func SplitParts(maybeParted []string) []string {
	r := []string{}
	for _, x := range maybeParted {
		if strings.Contains(x, ",") {
			for _, newKey := range strings.Split(x, ",") {
				r = append(r, newKey)
			}
		} else {
			r = append(r, x)
		}
	}
	return r
}

// SplitDomainRequest split the fqdn at the first subdomain
func SplitDomainRequest(fqdn string) (subDomain string, rootDomain string) {
	domainSlice := strings.Split(fqdn, ".")

	subDomain = domainSlice[0]
	rootDomain = strings.Join(domainSlice[1:], ".")
	return
}
