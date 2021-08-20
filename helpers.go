package sf

func AnyStr(candidates ...string) string {
	for _,s := range candidates {
		if !IsStrEmpty(s) {
			return s
		}
	}
	return ""
}


func A(err error, fn func() error) error {
	if err != nil {
		return err
	}
	return fn()
}