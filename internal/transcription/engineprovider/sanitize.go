package engineprovider

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var absolutePathPattern = regexp.MustCompile(`(?:[A-Za-z]:\\|/)[^\s:;,'")]+`)

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	msg = absolutePathPattern.ReplaceAllString(msg, "[redacted-path]")
	msg = redactTokenLikeValues(msg)
	return errors.New(msg)
}

func sanitizeErrorf(format string, args ...any) error {
	return sanitizeError(fmt.Errorf(format, args...))
}

func redactTokenLikeValues(msg string) string {
	parts := strings.Fields(msg)
	for i, part := range parts {
		lower := strings.ToLower(part)
		if strings.Contains(lower, "token") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") {
			if strings.Contains(part, "=") {
				key := strings.SplitN(part, "=", 2)[0]
				parts[i] = key + "=[redacted]"
			}
		}
	}
	return strings.Join(parts, " ")
}
