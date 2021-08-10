package soffa

import (
	"fmt"
	"github.com/rs/xid"
)

func NewUniqueId(prefix *string) string {
	p := ""
	if prefix != nil {
		p = *prefix
	}
	return fmt.Sprintf("%s%s", p, xid.New().String())
}