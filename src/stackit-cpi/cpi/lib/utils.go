package lib

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

func JitterWait(maxMilliseconds int64) int64 {
	if maxMilliseconds == 0 {
		return 0
	}
	jitter, err := rand.Int(rand.Reader, big.NewInt(maxMilliseconds/2))
	jitter = big.NewInt(0).Add(big.NewInt(maxMilliseconds/2), jitter)

	if err != nil {
		jitter = big.NewInt(maxMilliseconds)
	}
	time.Sleep(time.Duration(jitter.Int64()) * time.Millisecond)
	return jitter.Int64()
}

func ConvertErrToRPCResponse(err error) *RPCResponse {
	return &RPCResponse{
		Result: "",
		Error: &RPCError{
			Type:      CpiErrorType,
			Message:   fmt.Sprintf("%s", err),
			OkToRetry: false,
		},
		Log: "",
	}
}

// validateSecurityGroups validates that all security groups are non-empty strings
func validateSecurityGroups(groups []string) error {
	for i, group := range groups {
		if strings.TrimSpace(group) == "" {
			return fmt.Errorf("security group at index %d is empty", i)
		}
	}
	return nil
}

// IsValidUUID checks if a string is a valid UUID format
func IsValidUUID(uuidString string) bool {
	// google considers the no hyphen and others formats valid, our API doesn't. so we're checking for length of 36 chars as well
	return uuid.Validate(uuidString) == nil && len(uuidString) == 36
}

// UniqueArray removes duplicate dns strings from a slice and returns a new dns slice with unique values.
func UniqueArray(input []string) []string {
	slices.Sort(input)
	return slices.Compact(input)
}

func runRegexReplaceAndShortenIfNecessary(validatorRe *regexp.Regexp, input string, maxLength int) string {
	out := strings.Join(validatorRe.FindAllString(input, -1), "")
	if len(out) > maxLength {
		out = out[:maxLength]
	}
	return out
}

func MakeIaasCompatibleTagKeyString(input string) string {
	re := regexp.MustCompile(`([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]*`)
	return runRegexReplaceAndShortenIfNecessary(re, input, TagKeyMaxLength)
}

func MakeIaasCompatibleTagValueString(input string) string {
	re := regexp.MustCompile(`([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]*`)
	return runRegexReplaceAndShortenIfNecessary(re, input, TagValueMaxLength)
}

func MakeIaasCompatibleServerNameString(input string) string {
	// replace all the chars that we think should be a `-`
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", `\`, "-")

	if len(input) > ServerNameMaxLength {
		// from the go docs: `a[low : high]` ... `The result has indices starting at low and length equal to high - low`.
		input = input[:ServerNameMaxLength]
	}
	input = replacer.Replace(strings.ToLower(string(input))) // Lowercase and replace special characters
	dnsValidCharRegex := regexp.MustCompile(`[a-zA-Z0-9\-\.]`)
	input = strings.Join(dnsValidCharRegex.FindAllString(input, -1), "")

	// replace all consecutive `-` with exatly one
	dashRe := regexp.MustCompile("-+")
	input = dashRe.ReplaceAllString(input, "-")
	// replace all consecutive `.` with exatly one
	dotRe := regexp.MustCompile(`\.+`)
	input = dotRe.ReplaceAllString(input, ".")

	// remove all leading and trailing dots and hypens
	input = strings.Trim(input, ".-")

	// remove all hyphens that follow or prefix a dot
	hyphDotRe := regexp.MustCompile(`-?\.-?`)
	input = hyphDotRe.ReplaceAllString(input, ".")

	re := regexp.MustCompile(`^(([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9])$`)
	return runRegexReplaceAndShortenIfNecessary(re, input, ServerNameMaxLength)
}
