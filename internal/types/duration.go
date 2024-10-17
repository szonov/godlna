package types

import (
	"fmt"
	"time"
)

type Duration int64

func (d *Duration) Duration() time.Duration {
	if d != nil {
		return time.Duration(int64(*d) * int64(time.Second))
	}
	return 0
}

func (d *Duration) Uint64() uint64 {
	if d != nil {
		return uint64(*d)
	}
	return 0
}

func (d *Duration) String() string {
	dur := d.Duration()
	ms := dur.Milliseconds() % 1000
	s := int(dur.Seconds()) % 60
	m := int(dur.Minutes()) % 60
	h := int(dur.Hours())

	return fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
}

func (d *Duration) PercentOf(full *Duration) uint8 {
	dLen := d.Uint64()
	fullLen := full.Uint64()
	if dLen > fullLen {
		return 100
	}
	if dLen > 0 && fullLen > 0 {
		return uint8(100 * dLen / fullLen)
	}
	return 0
}
