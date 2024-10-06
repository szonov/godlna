package scpd

import (
	"fmt"
	"reflect"
)

type Builder struct {
	s       *SCPD
	vars    map[string]string
	actions map[string]bool
}

func NewBuilder(majorMinor ...uint) *Builder {
	if len(majorMinor) == 0 {
		majorMinor = append(majorMinor, 1)
	}
	if len(majorMinor) == 1 {
		majorMinor = append(majorMinor, 0)
	}
	return &Builder{
		s: &SCPD{
			SpecVersion: SpecVersion{
				Major: majorMinor[0],
				Minor: majorMinor[1],
			},
		},
		vars:    make(map[string]string),
		actions: make(map[string]bool),
	}
}

func (b *Builder) SCPD() *SCPD {
	return b.s
}

func (b *Builder) Add(actionName string, in, out any) error {
	if _, duplicate := b.actions[actionName]; duplicate {
		return fmt.Errorf("duplicate scpd action: %s", actionName)
	}
	b.actions[actionName] = true
	action := &Action{
		Name: actionName,
	}
	if err := b.parseAction(action, in, out); err != nil {
		return err
	}

	return nil
}
func (b *Builder) parseAction(action *Action, in, out any) error {

	//fmt.Printf("---- [%s]\n", action.Name)
	if err := b.parseArgs(action, "in", in); err != nil {
		return err
	}
	if err := b.parseArgs(action, "out", out); err != nil {
		return err
	}
	b.s.Actions = append(b.s.Actions, *action)

	return nil
}
func (b *Builder) parseArgs(action *Action, direction string, args any) error {

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
		stateVariable := &StateVariable{}
		if err := stateVariable.LoadString(scpdTag); err != nil {
			return fmt.Errorf("[SCPD] %s[%s]: can't load %s", action.Name, direction, scpdTag)
		}

		if knownTag, ok := b.vars[stateVariable.Name]; ok {
			// already defined variable, check that is completely the same
			if knownTag != scpdTag {
				return fmt.Errorf(
					"[SCPD] %s[%s] %s: different tags for the same variable '%s', '%s'",
					action.Name, direction, field.Name, scpdTag, knownTag)
			}
		} else {
			b.vars[stateVariable.Name] = scpdTag
			b.s.Variables = append(b.s.Variables, stateVariable)
		}

		action.Arguments = append(action.Arguments, &Argument{
			Name:      field.Name,
			Direction: direction,
			Variable:  stateVariable.Name,
		})
	}

	return nil
}
