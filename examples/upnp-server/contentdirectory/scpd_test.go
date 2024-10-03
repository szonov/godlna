package contentdirectory

import (
	"testing"
)

func TestValidateArgs(t *testing.T) {

	InitSCPD()

}

func TestValidateDump(t *testing.T) {

	InitSCPD()

	if err := ServiceSCPD.DumpArgsToFile("arguments.go"); err != nil {
		t.Error(err)
		return
	}
}
