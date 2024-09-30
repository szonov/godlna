package scpd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type dumpArgument struct {
	argName string
	argType string
	argVar  string
}

func (a dumpArgument) toString(nameLen int, typeLen int) string {
	v := "\t" +
		a.argName + strings.Repeat(" ", nameLen-len(a.argName)) +
		a.argType + strings.Repeat(" ", typeLen-len(a.argType)) +
		"`scpd:\"" + a.argVar + "\"`\n"
	return v
}

type dumpAction struct {
	name    string
	inArgs  []dumpArgument
	outArgs []dumpArgument
}

func (a dumpAction) toString(prefix string) string {
	var arguments []dumpArgument
	if prefix == "ArgIn" {
		arguments = a.inArgs
	} else {
		arguments = a.outArgs
	}
	ret := "type " + prefix + a.name + " struct {\n"
	var nameLen, typeLen int
	for _, aa := range arguments {
		nameLen = max(nameLen, len(aa.argName))
		typeLen = max(typeLen, len(aa.argType))
	}
	for _, aa := range arguments {
		ret += aa.toString(nameLen+1, typeLen+1)
	}
	ret += "}\n"
	return ret
}

// DumpArgs returns string with code, which can be used in package of developed service as golang source
func (s *SCPD) DumpArgs() (string, error) {
	if s == nil {
		return "", errors.New("SCPD is nil")
	}
	ret := ""
	for _, action := range s.Actions {

		dAction := dumpAction{
			name:    action.Name,
			inArgs:  make([]dumpArgument, 0),
			outArgs: make([]dumpArgument, 0),
		}

		for _, arg := range action.Arguments {
			if arg.Direction != "in" && arg.Direction != "out" {
				return "", fmt.Errorf("%s: invalid argument direction '%s'", arg.Name, arg.Direction)
			}
			if arg.Variable == "" {
				return "", fmt.Errorf("%s: absent related state variable", arg.Name)
			}
			variable := s.GetVariable(arg.Variable)
			if variable == nil {
				return "", fmt.Errorf("%s: invalid related state variable '%s'", arg.Name, arg.Variable)
			}
			a := dumpArgument{
				argName: arg.Name,
				argVar:  arg.Variable,
				argType: "string",
			}
			switch variable.DataType {
			// `ui1` Unsigned 1 Byte int. Same format as int without leading sign.
			case "ui1":
				a.argType = "uint8"
			// `ui2` Unsigned 2 Byte int. Same format as int without leading sign.
			case "ui2":
				a.argType = "uint16"
			// `ui4` Unsigned 4 Byte int. Same format as int without leading sign.
			case "ui4":
				a.argType = "uint32"
			// `i1` 1 Byte int. Same format as int.
			case "i1":
				a.argType = "int8"
			// `i2` 2 Byte int. Same format as int.
			case "i2":
				a.argType = "int16"
			// `i4` 4 Byte int. Same format as int. Must be between -2147483648 and 2147483647.
			case "i4":
				a.argType = "int32"
			// `int` Fixed point, integer number. May have leading sign. May have leading zeros.
			// (No currency symbol.) (No grouping of digits to the left of the decimal, e.g., no commas.)
			case "int":
				a.argType = "int64"
			// `r4` 4 Byte float. Same format as float. Must be between 3.40282347E+38 to 1.17549435E-38.
			case "r4":
				a.argType = "float32"
			// `r8` 8 Byte float. Same format as float. Must be between -1.79769313486232E308 and -4.94065645841247E-324 for negative values,
			//      and between 4.94065645841247E-324 and 1.79769313486232E308 for positive values, i.e., IEEE 64-bit (8-Byte) double.
			// `number` Same as r8.
			// `fixed.14.4` Same as r8 but no more than 14 digits to the left of the decimal point and no more than 4 to the right.
			// `float` Floating point number. Mantissa (left of the decimal) and/or exponent may
			//         have a leading sign. Mantissa and/or exponent may have leading zeros. Decimal
			//         character in mantissa is a period, i.e., whole digits in mantissa separated from
			//         fractional digits by period. Mantissa separated from exponent by E. (No currency symbol.)
			//         (No grouping of digits in the mantissa, e.g., no commas.)
			case "r8", "number", "fixed.14.4", "float":
				a.argType = "float64"
			// `char` Unicode string. One character long.
			case "char":
				a.argType = "rune"
			// `string` Unicode string. No limit on length.
			case "string":
				a.argType = "string"
			// `boolean` “0” for false or “1” for true. The values “true”, “yes”, “false”, or “no”
			//           may also be used but are not recommended.
			case "boolean":
				a.argType = "bool"
			// `uri` Universal Resource Identifier.
			case "uri":
				a.argType = "string"
			// TODO: types for golang >>>>
			// `date` Date in a subset of ISO 8601 format without time data.
			// `dateTime` Date in ISO 8601 format with optional time but no time zone.
			// `dateTime.tz` Date in ISO 8601 format with optional time and optional time zone.
			// `time` Time in a subset of ISO 8601 format with no date and no time zone.
			// `time.tz` Time in a subset of ISO 8601 format with optional time zone but no date.
			// `bin.base64` MIME-style Base64 encoded binary BLOB. Takes 3 Bytes, splits them into 4 parts,
			//              and maps each 6 bit piece to an octet. (3 octets are encoded as 4.) No limit on size.
			// `bin.hex` Hexadecimal digits representing octets. Treats each nibble as a hex digit and
			//           encodes as a separate Byte. (1 octet is encoded as 2.) No limit on size.
			// `uuid` Universally Unique ID. Hexadecimal digits representing octets.
			//        Optional embedded hyphens are ignored.
			// TODO: <<<<<<<<<<<<<<<<<<<<<
			default:
				fmt.Printf("WARN: %s %s:%s unhandled variable type '%s'\n", arg.Name, arg.Direction, arg.Variable, variable.DataType)
			}

			if arg.Direction == "in" {
				dAction.inArgs = append(dAction.inArgs, a)
			} else {
				dAction.outArgs = append(dAction.outArgs, a)
			}
		}
		ret += dAction.toString("ArgIn") + dAction.toString("ArgOut")
	}

	return ret, nil
}

// DumpArgsToFile writes golang code with arguments to defined file
func (s *SCPD) DumpArgsToFile(filename string) error {
	body, err := s.DumpArgs()
	if err != nil {
		return err
	}
	return addCodeToFile(filename, body)
}

func addCodeToFile(file string, code string) error {
	var err error
	if file, err = filepath.Abs(file); err != nil {
		return err
	}
	dir := filepath.Dir(file)
	if _, err = os.Stat(dir); err != nil {
		return fmt.Errorf("%s: directory does not exist", dir)
	}
	packageName := strings.Replace(filepath.Base(dir), "-", "_", -1)
	code = "package " + packageName + "\n\n" + code
	if err = os.WriteFile(file, []byte(code), 0644); err != nil {
		return err
	}
	return nil
}
