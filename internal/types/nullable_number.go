package types

import (
	"fmt"
	"time"
)

type NullableNumber int64

func (n *NullableNumber) String() string {
	return fmt.Sprintf("%d", n.Int64())
}

func (n *NullableNumber) Uint64() uint64 {
	if n != nil {
		return uint64(*n)
	}
	return 0
}
func (n *NullableNumber) Int() int {
	if n != nil {
		return int(*n)
	}
	return 0
}
func (n *NullableNumber) Uint() uint {
	if n != nil {
		return uint(*n)
	}
	return 0
}

func (n *NullableNumber) Int64() int64 {
	if n != nil {
		return int64(*n)
	}
	return 0
}

func (n *NullableNumber) Time() time.Time {
	return time.Unix(n.Int64(), 0)
}
