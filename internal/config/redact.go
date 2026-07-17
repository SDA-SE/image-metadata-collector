package config

import (
	"net/url"
	"strings"
)

const redactedPlaceholder = "<redacted>"

var sensitiveHeaderNames = map[string]struct{}{
	"api-key":             {},
	"authorization":       {},
	"proxy-authorization": {},
	"x-api-key":           {},
}

// Redacted returns a copy of the configuration with sensitive values removed.
func (c Config) Redacted() Config {
	redacted := c
	redacted.GitPassword = redactedPlaceholder
	redacted.ApiKey = redactedPlaceholder
	redacted.ApiSignature = redactedPlaceholder
	redacted.GitUrl = redactURLUserInfo(redacted.GitUrl)
	redacted.HTTPHeaders = redactHTTPHeaders(redacted.HTTPHeaders)

	return redacted
}

func redactURLUserInfo(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.User == nil {
		return rawURL
	}

	username := parsedURL.User.Username()
	if username == "" {
		parsedURL.User = url.UserPassword(redactedPlaceholder, redactedPlaceholder)
		return parsedURL.String()
	}

	parsedURL.User = url.UserPassword(username, redactedPlaceholder)
	return parsedURL.String()
}

func redactHTTPHeaders(headers []string) []string {
	if len(headers) == 0 {
		return nil
	}

	redacted := make([]string, len(headers))
	for i, header := range headers {
		name, value, found := strings.Cut(header, ":")
		if !found {
			redacted[i] = redactHeaderValueIfSensitive("", header)
			continue
		}

		redacted[i] = name + ":" + redactHeaderValueIfSensitive(name, strings.TrimSpace(value))
	}

	return redacted
}

func redactHeaderValueIfSensitive(name, value string) string {
	normalizedName := strings.TrimSpace(strings.ToLower(name))
	if _, ok := sensitiveHeaderNames[normalizedName]; ok {
		return redactedPlaceholder
	}

	if containsSensitiveKeyword(normalizedName) {
		return redactedPlaceholder
	}

	if name == "" && containsSensitiveKeyword(strings.ToLower(strings.TrimSpace(value))) {
		return redactedPlaceholder
	}

	return value
}

func containsSensitiveKeyword(value string) bool {
	return strings.Contains(value, "token") ||
		strings.Contains(value, "secret") ||
		strings.Contains(value, "signature")
}
