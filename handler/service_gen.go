package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/szonov/go-upnp-lib/scpd"
)

var DataTypeMap = map[string]string{
	"ui1":    "uint8",
	"ui2":    "uint16",
	"ui4":    "uint32",
	"string": "string",
	"uri":    "scpd.URI",
}

// ServiceGen all params required
type ServiceGen struct {
	// serviceSCPD can be taken from xml by scpd.Load([]byte)
	ServiceSCPD *scpd.SCPD

	// ServiceType for example "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceType string

	// ServiceId for example "urn:upnp-org:serviceId:ContentDirectory"
	ServiceId string

	// Directory to which store generated files
	Directory string

	// ControllerName for example "ServiceController"
	ControllerName string

	// ControllerFile for example "controller.go"
	ControllerFile string

	// ArgumentsFile for example "arguments.go"
	ArgumentsFile string

	// CreateHandlerFile for example "actions.go"
	CreateHandlerFile string

	dir         string
	pkg         string
	serviceName string
	actions     []genAction
	argsScpd    bool
}

// GenerateService generate *.go files
// ATTENTION: files will be rewritten
func (gen *ServiceGen) GenerateService() (err error) {
	if err = gen.validateInParams(); err != nil {
		return
	}
	if gen.dir, err = gen.makeDir(); err != nil {
		return
	}
	gen.pkg = strings.Replace(filepath.Base(gen.dir), "-", "_", -1)

	if gen.actions, err = gen.readActions(); err != nil {
		return
	}
	if len(gen.actions) == 0 {
		err = errors.New("no actions found")
		return
	}

	if err = gen.generateArguments(); err != nil {
		return
	}
	if err = gen.generateHandlerConfig(); err != nil {
		return
	}
	if err = gen.generateController(); err != nil {
		return
	}

	return nil
}

func (gen *ServiceGen) validateInParams() error {
	if gen.ServiceSCPD == nil {
		return errors.New("missing ServiceSCPD")
	}

	if gen.ServiceType == "" {
		return errors.New("missing Service")
	}

	parts := strings.Split(gen.ServiceType, ":")
	if len(parts) != 5 || parts[0] != "urn" || parts[2] != "service" {
		return errors.New("invalid Service")
	}
	gen.serviceName = parts[3]

	if gen.ServiceId == "" {
		return errors.New("missing ServiceId")
	}

	if gen.Directory == "" {
		return errors.New("missing Directory")
	}

	if gen.ControllerName == "" {
		return errors.New("missing ControllerName")
	}

	if !gen.isValidFileName(gen.ControllerFile) {
		return errors.New("missing or invalid ControllerFile, should have suffix '.go'")
	}

	if !gen.isValidFileName(gen.ArgumentsFile) {
		return errors.New("missing or invalid ArgumentsFile, should have suffix '.go'")
	}

	if !gen.isValidFileName(gen.CreateHandlerFile) {
		return errors.New("missing or invalid CreateHandlerFile, should have suffix '.go'")
	}

	return nil
}

func (gen *ServiceGen) isValidFileName(name string) bool {
	return name != "" &&
		!strings.Contains(name, "/") &&
		!strings.HasPrefix(name, ".") &&
		strings.HasSuffix(name, ".go")
}

func (gen *ServiceGen) makeDir() (fullPath string, err error) {
	fullPath, err = filepath.Abs(gen.Directory)
	if err != nil {
		return
	}
	if _, err = os.Stat(fullPath); os.IsNotExist(err) {
		err = os.MkdirAll(fullPath, os.ModePerm)
		return
	}
	return
}

type genArg struct {
	argName string
	argType string
	argTag  string
}

func (a genArg) toString(nameLen int, typeLen int) string {
	nam := a.argName + strings.Repeat(" ", nameLen-len(a.argName))
	typ := a.argType + strings.Repeat(" ", typeLen-len(a.argType))

	return fmt.Sprintf("\t%s %s `scpd:\"%s\"`\n", nam, typ, a.argTag)
}

type genAction struct {
	name    string
	inArgs  []genArg
	outArgs []genArg
}

func (a genAction) toString(prefix string) string {
	var arguments []genArg
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

func (gen *ServiceGen) readActions() ([]genAction, error) {
	ret := make([]genAction, 0)

	for _, action := range gen.ServiceSCPD.Actions {

		dAction := genAction{
			name:    action.Name,
			inArgs:  make([]genArg, 0),
			outArgs: make([]genArg, 0),
		}

		for _, arg := range action.Arguments {
			if arg.Direction != "in" && arg.Direction != "out" {
				return nil, fmt.Errorf("%s:%s: invalid argument direction '%s'", action.Name, arg.Name, arg.Direction)
			}
			if arg.Variable == "" {
				return nil, fmt.Errorf("%s:%s: related state variable not found", action.Name, arg.Name)
			}
			variable := gen.ServiceSCPD.GetVariable(arg.Variable)
			if variable == nil {
				return nil, fmt.Errorf("%s:%s: invalid related state variable '%s'", action.Name, arg.Name, arg.Variable)
			}
			a := genArg{
				argName: arg.Name,
				argTag:  variable.String(),
			}

			var ok bool
			if a.argType, ok = DataTypeMap[variable.DataType]; !ok {
				slog.Warn("scpd::unhandled variable type",
					slog.String("var", arg.Variable),
					slog.String("arg", arg.Name),
					slog.String("dir", arg.Direction),
					slog.String("type", variable.DataType),
				)
				//fmt.Printf("WARN: %s %s:%s unhandled variable type '%s'\n", arg.Name, arg.Direction, arg.Variable, variable.DataType)
				a.argType = "string"
			}
			if strings.HasPrefix(a.argType, "scpd.") {
				gen.argsScpd = true
			}
			if arg.Direction == "in" {
				dAction.inArgs = append(dAction.inArgs, a)
			} else {
				dAction.outArgs = append(dAction.outArgs, a)
			}
		}

		ret = append(ret, dAction)
	}

	return ret, nil
}

func (gen *ServiceGen) addCodeToFile(f string, imports []string, code string) error {

	file := filepath.Join(gen.dir, f)
	body := "package " + gen.pkg + "\n\n"
	if len(imports) > 0 {
		body += "import (\n"
		for _, str := range imports {
			body += "\t\"" + str + "\"\n"
		}
		body += ")\n\n"
	}
	body += code

	return os.WriteFile(file, []byte(body), 0644)
}

func (gen *ServiceGen) generateArguments() error {

	code := ""
	for _, action := range gen.actions {
		code += action.toString("ArgIn") + action.toString("ArgOut")
	}
	imports := make([]string, 0)
	if gen.argsScpd {
		imports = append(imports, "github.com/szonov/go-upnp-lib/scpd")
	}
	return gen.addCodeToFile(
		gen.ArgumentsFile,
		imports,
		code,
	)
}

func (gen *ServiceGen) generateHandlerConfig() error {
	actionsBody := ""
	for _, action := range gen.actions {
		tm := `
		{
			Name: "%[1]s",
			Func: ctl.%[1]s,
			Args: func() (interface{}, interface{}) {
				return &ArgIn%[1]s{}, &ArgOut%[1]s{}
			},
		},`
		actionsBody += fmt.Sprintf(tm, action.name)
	}
	tmpl := `func (ctl *%[1]s) createActions() []handler.Action {
	return []handler.Action{%[2]s
	}
}
`
	code := fmt.Sprintf(tmpl, gen.ControllerName, actionsBody)

	return gen.addCodeToFile(
		gen.CreateHandlerFile,
		[]string{
			"github.com/szonov/go-upnp-lib/handler",
		},
		code,
	)
}

func (gen *ServiceGen) generateController() error {
	actionsBody := ""
	for _, action := range gen.actions {
		tm := `func (ctl *%[1]s) %[2]s(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgIn%[2]s)
	//out := ctx.ArgOut.(*ArgOut%[2]s)
	return nil
}
`
		actionsBody += fmt.Sprintf(tm, gen.ControllerName, action.name)
	}
	tmpl := `const (
	ServiceType = "%[2]s"
	ServiceId   = "%[3]s"
)

type %[1]s struct {
	Handler *handler.Handler
	Service *device.Service
}

func New%[1]s() *%[1]s {
	ctl := new(%[1]s)
	ctl.Service = &device.Service{
		ServiceType: ServiceType,
		ServiceId:   ServiceId,
		SCPDURL:     "/%[4]s.xml",
		ControlURL:  "/ctl/%[4]s",
		EventSubURL: "/evt/%[4]s",
	}
	ctl.Handler = &handler.Handler{
		Service: ctl.Service.ServiceType,
		Actions: ctl.createActions(),
	}
	return ctl
}

// OnServerStart implements upnp.Controller interface
func (ctl *%[1]s) OnServerStart(server *upnp.Server) error {
	if err := ctl.Handler.Init(); err != nil {
		return err
	}
	server.Device.ServiceList = append(server.Device.ServiceList, ctl.Service)
	return nil
}

// Handle implements upnp.Controller interface
func (ctl *%[1]s) Handle(w http.ResponseWriter, r *http.Request) bool {

	if r.URL.Path == ctl.Service.SCPDURL {
		ctl.Handler.HandleSCPDURL(w, r)
		return true
	}

	if r.URL.Path == ctl.Service.ControlURL {
		ctl.Handler.HandleControlURL(w, r)
		return true
	}

	if r.URL.Path == ctl.Service.EventSubURL {
		ctl.Handler.HandleEventSubURL(w, r)
		return true
	}

	return false
}

`
	code := fmt.Sprintf(
		tmpl,
		gen.ControllerName, /*1*/
		gen.ServiceType,    /*2*/
		gen.ServiceId,      /*3*/
		gen.serviceName,    /*4*/
	) + actionsBody

	return gen.addCodeToFile(
		gen.ControllerFile,
		[]string{
			"github.com/szonov/go-upnp-lib",
			"github.com/szonov/go-upnp-lib/device",
			"github.com/szonov/go-upnp-lib/handler",
			"net/http",
		},
		code,
	)
}
