package scpd

import (
	"fmt"
	"reflect"
	"strconv"
)

func (s *SCPD) ValidateArgs(args interface{}) error {

	v := reflect.ValueOf(args)
	t := reflect.TypeOf(args)

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct")
	}

	var stateVariable *StateVariable

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("scpd")

		if tag == "" {
			continue
		}

		if stateVariable = s.GetVariable(tag); stateVariable == nil {
			return fmt.Errorf("invalid configuration, unknown state variable %s", tag)
		}

		// AllowedValues
		if stateVariable.AllowedValues != nil && len(*stateVariable.AllowedValues) > 0 {
			var ok bool
			var checkValue string

			switch fieldValue.Kind() {
			case reflect.String:
				checkValue = fieldValue.String()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				checkValue = strconv.FormatInt(fieldValue.Int(), 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				checkValue = strconv.FormatUint(fieldValue.Uint(), 10)
			default:
				// for now check only strings and integers
				ok = true
			}

			if !ok {
				for _, val := range *stateVariable.AllowedValues {
					if val == checkValue {
						ok = true
						break
					}
				}
			}
			if !ok {
				return fmt.Errorf("%s: not allowed value '%s'", fieldType.Name, checkValue)
			}
		}

		// AllowedRange - only for integers
		if stateVariable.AllowedRange != nil {
			switch fieldValue.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val := fieldValue.Int()
				minV := int64(stateVariable.AllowedRange.Min)
				maxV := int64(stateVariable.AllowedRange.Max)
				if val < minV || val > maxV {
					return fmt.Errorf("%s: value '%d' not in range [%d, %d]", fieldType.Name, val, minV, maxV)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val := fieldValue.Uint()
				minV := uint64(stateVariable.AllowedRange.Min)
				maxV := uint64(stateVariable.AllowedRange.Max)
				if val < minV || val > maxV {
					return fmt.Errorf("%s: value '%d' not in range [%d, %d]", fieldType.Name, val, minV, maxV)
				}
			default:
			}
		}
	}
	return nil
}
