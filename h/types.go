package h

type Map = map[string]interface{}

func UnwrapMap(value Map) map[string]interface{} {
	if IsNil(value) {
		return nil
	}
	out := map[string]interface{}{}
	for k, v := range value {
		out[k] = v
	}
	return out
}
