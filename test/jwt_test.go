package test

import (
	"github.com/soffa-io/soffa-core-go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJwt(t *testing.T) {
	secret := "FT12871782168261NN"
	token, err := sf.CreateJwt(secret, "soffa", "SOFFA", "soffa", sf.H{"name": "Factory"})
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	decoded, err := sf.DecodeJwt(secret, token)
	assert.Nil(t, err)
	assert.NotNil(t, decoded)
}
