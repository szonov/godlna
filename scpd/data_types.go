package scpd

import (
	"net/url"
	"strconv"
)

var DataTypeMap = map[string]string{
	"ui1":    "scpd.UI1",
	"ui2":    "scpd.UI2",
	"ui4":    "scpd.UI4",
	"string": "string",
	"uri":    "scpd.URI",
}

// DONE [ui1] :: UI1
// Unsigned 1 Byte int. Same format as int without leading sign.

// DONE [ui2] :: UI2
// Unsigned 2 Byte int. Same format as int without leading sign.

// DONE [ui4] :: UI4
// Unsigned 4 Byte int. Same format as int without leading sign.

// TODO: [i1]
// 1 Byte int. Same format as int.

// TODO: [i2]
// 2 Byte int. Same format as int.

// TODO: [i4]
// 4 Byte int. Same format as int. Must be between -2147483648 and 2147483647.

// TODO: [int] Fixed point, integer number. May have leading sign. May have leading zeros.
// (No currency symbol.) (No grouping of digits to the left of the decimal, e.g., no commas.)

// TODO: [r4]
// 4 Byte float. Same format as float. Must be between 3.40282347E+38 to 1.17549435E-38.

// TODO: [r8]
// 8 Byte float. Same format as float. Must be between -1.79769313486232E308 and -4.94065645841247E-324
// for negative values, and between 4.94065645841247E-324 and 1.79769313486232E308 for positive values,
// i.e., IEEE 64-bit (8-Byte) double.

// TODO: [number]
// Same as r8.

// TODO: [fixed.14.4]
// Same as r8 but no more than 14 digits to the left of the decimal point and no more than 4 to the right.

// TODO: [float]
// Floating point number. Mantissa (left of the decimal) and/or exponent may
// have a leading sign. Mantissa and/or exponent may have leading zeros. Decimal
// character in mantissa is a period, i.e., whole digits in mantissa separated from
// fractional digits by period. Mantissa separated from exponent by E. (No currency symbol.)
// (No grouping of digits in the mantissa, e.g., no commas.)

// TODO: [char]
// Unicode string. One character long.

// DONE [string] :: use native 'string' type
// Unicode string. No limit on length.

// TODO: [date]
// Date in a subset of ISO 8601 format without time data.

// TODO: [dateTime]
// Date in ISO 8601 format with optional time but no time zone.

// TODO: [dateTime.tz]
// Date in ISO 8601 format with optional time and optional time zone.

// TODO: [time]
// Time in a subset of ISO 8601 format with no date and no time zone.

// TODO: [time.tz]
// Time in a subset of ISO 8601 format with optional time zone but no date.

// TODO: [boolean]
// “0” for false or “1” for true. The values “true”, “yes”, “false”, or “no” may
// also be used but are not recommended.

// TODO: [bin.base64]
// MIME-style Base64 encoded binary BLOB. Takes 3 Bytes, splits them into 4 parts,
// and maps each 6 bit piece to an octet. (3 octets are encoded as 4.) No limit on size.

// TODO: [bin.hex]
// Hexadecimal digits representing octets. Treats each nibble as a hex digit and
// encodes as a separate Byte. (1 octet is encoded as 2.) No limit on size.

// DONE [uri]
// Universal Resource Identifier.

// TODO: UID [uuid]
// Universally Unique ID. Hexadecimal digits representing octets.
// Optional embedded hyphens are ignored.

// For now implementing only few types, appears in ContentDirectory scpd
// and several additional for testing...
//
//➜ cat scpd.xml| grep -oE "<dataType>.+</dataType>" | sort | uniq
//<dataType>string</dataType>
//<dataType>ui4</dataType>
//<dataType>uri</dataType>

func uintString[T UI1 | UI2 | UI4](v *T) string {
	return strconv.FormatUint(uint64(*v), 10)
}

func uintMarshalText[T UI1 | UI2 | UI4](v *T) ([]byte, error) {
	return strconv.AppendUint(nil, uint64(*v), 10), nil
}

// UI1 'ui1' presentation of SCPD dataType
type UI1 uint8

func (v *UI1) String() string {
	return uintString(v)
}
func (v *UI1) MarshalText() ([]byte, error) {
	return uintMarshalText(v)
}
func (v *UI1) UnmarshalText(b []byte) error {
	t, err := strconv.ParseUint(string(b), 10, 8)
	*v = UI1(t)
	return err
}

// UI2 'ui2' presentation of SCPD dataType
type UI2 uint16

func (v *UI2) String() string {
	return uintString(v)
}
func (v *UI2) MarshalText() ([]byte, error) {
	return uintMarshalText(v)
}
func (v *UI2) UnmarshalText(b []byte) error {
	t, err := strconv.ParseUint(string(b), 10, 16)
	*v = UI2(t)
	return err
}

// UI4 'ui4' presentation of SCPD dataType
type UI4 uint32

func (v *UI4) String() string {
	return uintString(v)
}
func (v *UI4) MarshalText() ([]byte, error) {
	return uintMarshalText(v)
}
func (v *UI4) UnmarshalText(b []byte) error {
	t, err := strconv.ParseUint(string(b), 10, 32)
	*v = UI4(t)
	return err
}

// URI 'uri' presentation of SCPD dataType
type URI url.URL

func (v *URI) Type() string {
	return "uri"
}

func (v *URI) String() string {
	return (*url.URL)(v).String()
}
func (v *URI) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}
func (v *URI) UnmarshalText(b []byte) error {
	u, err := url.Parse(string(b))
	if err != nil {
		return err
	}
	*v = URI(*u)
	return nil
}
