package test

import (
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJwt(t *testing.T) {
	secret := "FT12871782168261NN"
	token, err := h.CreateJwt(secret, "soffa", "SOFFA", "soffa", h.Map{"name": "Factory"})
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	decoded, err := h.DecodeJwt(secret, token)
	assert.Nil(t, err)
	assert.NotNil(t, decoded)
}
