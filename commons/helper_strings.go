package commons

import (
	"regexp"
	"strings"
)

var (
	 regexCleanPathInt = regexp.MustCompile(`/+`)
	 regexCleanPath = regexp.MustCompile(`^/+|/+$`)
)

func IsStrEmpty(value string) bool {
	return len(value) == 0
}

func JoinPath(values ...string) string {
	return regexCleanPathInt.ReplaceAllString(strings.Join(values, "/"), "/")
}

func IsSamePath(value1 string, value2 string) bool {
	return strings.EqualFold(regexCleanPath.ReplaceAllString(value1, ""), regexCleanPath.ReplaceAllString(value2, ""))
}