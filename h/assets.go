package h

import "github.com/soffa-io/soffa-core-go/log"

func AssertNotNil(value interface{}, message string) {
	if IsNil(value) {
		log.Default.Fatal(message)
	}
}
