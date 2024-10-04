package handler

import (
	"fmt"
	"github.com/szonov/go-upnp-lib/scpd"
	"reflect"
)

type scpdMaker struct {
	h    *Handler
	s    *scpd.SCPD
	vars map[string]string
}

func MakeSCPD(h *Handler, majorMinor ...uint) (*scpd.SCPD, error) {
	return new(scpdMaker).Make(h, majorMinor...)
}

func (m *scpdMaker) Make(h *Handler, majorMinor ...uint) (*scpd.SCPD, error) {

	v := scpd.SpecVersion{Major: 1, Minor: 0}
	if len(majorMinor) > 0 {
		v.Major = majorMinor[0]
	}
	if len(majorMinor) > 1 {
		v.Minor = majorMinor[1]
	}

	m.h = h
	m.s = &scpd.SCPD{SpecVersion: v}
	m.vars = map[string]string{}

	for actionName, actionDef := range m.h.Actions {
		info := actionDef()
		if err := m.parseAction(&scpd.Action{Name: actionName}, info.ArgIn, info.ArgOut); err != nil {
			return nil, err
		}
	}

	return m.s, nil
}
func (m *scpdMaker) parseAction(action *scpd.Action, in, out any) error {

	//fmt.Printf("---- [%s]\n", action.Name)

	if err := m.parseActionArgs(action, "in", in); err != nil {
		return err
	}
	if err := m.parseActionArgs(action, "out", out); err != nil {
		return err
	}
	m.s.Actions = append(m.s.Actions, *action)

	return nil
}

func (m *scpdMaker) parseActionArgs(action *scpd.Action, direction string, args any) error {

	if reflect.ValueOf(args).Kind() != reflect.Pointer {
		return fmt.Errorf("[SCPD] %s[%s]: must be pointer of struct", action.Name, direction)
	}

	rv := reflect.ValueOf(&args).Elem().Elem().Elem()
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return fmt.Errorf("[SCPD] %s[%s]: invalid or non struct", action.Name, direction)
	}

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Type().Field(i)
		scpdTag := field.Tag.Get("scpd")
		// skip non-scpd arguments
		if scpdTag == "" {
			continue
		}
		//fmt.Printf("  >> [%s] [%s]\n", direction, field.Name)
		stateVariable := &scpd.StateVariable{}
		if err := stateVariable.LoadString(scpdTag); err != nil {
			return fmt.Errorf("[SCPD] %s[%s]: can't load %s", action.Name, direction, scpdTag)
		}

		if knownTag, ok := m.vars[stateVariable.Name]; ok {
			// already defined variable, check that is completely the same
			if knownTag != scpdTag {
				return fmt.Errorf(
					"[SCPD] %s[%s] %s: different tags for the same variable '%s', '%s'",
					action.Name, direction, field.Name, scpdTag, knownTag)
			}
		} else {
			m.vars[stateVariable.Name] = scpdTag
			m.s.Variables = append(m.s.Variables, stateVariable)
		}
		action.Arguments = append(action.Arguments, &scpd.Argument{
			Name:      field.Name,
			Direction: direction,
			Variable:  stateVariable.Name,
		})
	}

	return nil
}
