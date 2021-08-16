package soffa

import "github.com/mitchellh/mapstructure"

func Convert(input interface{}, dest interface{}) error {
	return mapstructure.Decode(input, &dest)
}
