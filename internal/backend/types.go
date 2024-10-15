package backend

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	Folder = 1
	Video  = 2
)

var (
	MediaDir string
	CacheDir string
	DB       *sql.DB
)

type (
	Object struct {
		ID          int64
		ObjectID    string
		ParentID    string
		Type        int
		Title       string
		Path        string
		Timestamp   *NullableNumber
		MetaData    string
		UpdateID    uint64
		Size        *NullableNumber
		Resolution  *NullableString
		Channels    *NullableNumber
		SampleRate  *NullableNumber
		BitRate     *NullableNumber
		Bookmark    *NullableNumber
		DurationSec *Duration
		MimeType    *NullableString
	}

	ObjectFilter struct {
		ObjectID string
		ParentID string
		Limit    int64
		Offset   int64
	}

	Duration       float64
	NullableNumber uint64
	NullableString string
)

func (d *Duration) Duration() time.Duration {
	if d != nil {
		return time.Duration(float64(*d) * float64(time.Second))
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

	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

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

func (s *NullableString) String() string {
	if s != nil {
		return string(*s)
	}
	return ""
}
