package scpd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type dumpArgument struct {
	argName    string
	argType    string
	argVar     string
	argEvents  bool
	argRange   string
	argAllowed string
	argDefault string
}

func (a dumpArgument) toString(nameLen int, typeLen int) string {
	nam := a.argName + strings.Repeat(" ", nameLen-len(a.argName))
	typ := a.argType + strings.Repeat(" ", typeLen-len(a.argType))

	t := make([]string, 0)
	if a.argEvents {
		t = append(t, ` events:"yes"`)
	}
	if a.argDefault != "" {
		t = append(t, fmt.Sprintf(` default:"%s"`, a.argDefault))
	}
	if a.argRange != "" {
		t = append(t, fmt.Sprintf(` range:"%s"`, a.argRange))
	}
	if a.argAllowed != "" {
		t = append(t, fmt.Sprintf(` allowed:"%s"`, a.argAllowed))
	}
	tag := fmt.Sprintf(`scpd:"%s"%s`, a.argVar, strings.Join(t, ""))

	return fmt.Sprintf("\t%s %s `%s`\n", nam, typ, tag)
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
		ret += aa.toString(nameLen, typeLen)
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
				return "", fmt.Errorf("%s: related state variable not found", arg.Name)
			}
			variable := s.GetVariable(arg.Variable)
			if variable == nil {
				return "", fmt.Errorf("%s: invalid related state variable '%s'", arg.Name, arg.Variable)
			}
			a := dumpArgument{
				argName:    arg.Name,
				argVar:     arg.Variable,
				argType:    "string",
				argDefault: variable.Default,
			}

			if variable.Events == "yes" {
				a.argEvents = true
			}

			var ok bool
			if a.argType, ok = DataTypeMap[variable.DataType]; !ok {
				fmt.Printf("WARN: %s %s:%s unhandled variable type '%s'\n", arg.Name, arg.Direction, arg.Variable, variable.DataType)
				a.argType = "string"
			}

			if variable.AllowedRange != nil {
				ran := variable.AllowedRange
				a.argRange = fmt.Sprintf("%d,%d,%d", ran.Min, ran.Max, ran.Step)
			}
			if variable.AllowedValues != nil {
				a.argAllowed = strings.Join(*variable.AllowedValues, ",")
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

	code = "package " + packageName + "\n\n" +
		"import (\n" +
		"\t\"github.com/szonov/go-upnp-lib/scpd\"\n" +
		")\n\n" +
		code

	if err = os.WriteFile(file, []byte(code), 0644); err != nil {
		return err
	}
	return nil
}
