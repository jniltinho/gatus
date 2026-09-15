package endpoint

import "unicode/utf8"

const (
	// ResultOriginPush is the origin of the results pushed through /api/push (fork)
	ResultOriginPush = "push"

	// MaximumResultMessageLength is the maximum length, in bytes, of the message of a result
	MaximumResultMessageLength = 1024
)

// TruncateResultMessage returns message cut to at most MaximumResultMessageLength bytes, without splitting a UTF-8
// character
func TruncateResultMessage(message string) string {
	if len(message) <= MaximumResultMessageLength {
		return message
	}
	cut := MaximumResultMessageLength
	for cut > 0 && !utf8.RuneStart(message[cut]) {
		cut--
	}
	return message[:cut]
}
