package h

import (
	"fmt"
	"github.com/rs/xid"
)

// Create a new UniqueId with a prefix
func NewUniqueIdP(prefix string) string {
	return fmt.Sprintf("%s%s", prefix, xid.New().String())
}

// Create a new UniqueId
func NewUniqueId() string {
	return NewUniqueIdP("")
}

