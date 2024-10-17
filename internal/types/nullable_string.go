package types

type NullableString string

func (s *NullableString) String() string {
	if s != nil {
		return string(*s)
	}
	return ""
}
