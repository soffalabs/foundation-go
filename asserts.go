package sf

import "github.com/soffa-io/soffa-core-go/log"

func AssertNotEmpty(value string, message string) {
	if IsStrEmpty(value) {
		log.Fatal(message)
	}
}
