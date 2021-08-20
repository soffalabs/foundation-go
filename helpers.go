package sf

func AnyStr(candidates ...string) string {
	for _,s := range candidates {
		if !IsStrEmpty(s) {
			return s
		}
	}
	return ""
}
