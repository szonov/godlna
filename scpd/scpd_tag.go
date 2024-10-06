package scpd

import (
	"fmt"
	"strconv"
	"strings"
)

// This file contains serialize/deserialize StateVariable using tag in scpd:"..." in arguments

// String generate scpd tag for StateVariable in format
// "{state_variable},{data_type},[events,][min={min},][max={int},][step={int},][default={default}][ {allowed}]"
// - {state_variable} - required, name of state variable, example: SystemUpdateID
// - {data_type} - required, data type of state variable, example: ui4
// - events - optional, appears only if sendEvents="yes"
// - min={min} - optional, Minimum Range value, skipped if empty or 0, example min=4
// - max={max} - optional, Maximum Range value, skipped if empty or 0, example max=10
// - step={step} - optional, Step Range value, skipped if empty or 0, example step=1
// - default={default} - optional, Default value, skipped if empty example default=NORMAL
// - {allowed} - optional, comma separated list of allowed values, IMPORTANT: have space before
// Example:
//
//	OnLine string `scpd:"A_ARG_TYPE_OnLine,string,events,default=line busy,line,broken"`
func (sv *StateVariable) String() string {
	ret := sv.Name + "," + sv.DataType
	if sv.Events == "yes" {
		ret += ",events"
	}
	if sv.AllowedRange != nil {
		if sv.AllowedRange.Min != 0 {
			ret += ",min=" + strconv.Itoa(sv.AllowedRange.Min)
		}
		if sv.AllowedRange.Max != 0 {
			ret += ",max=" + strconv.Itoa(sv.AllowedRange.Max)
		}
		if sv.AllowedRange.Step != 0 {
			ret += ",step=" + strconv.Itoa(sv.AllowedRange.Step)
		}
	}
	if sv.Default != "" {
		ret += ",default=" + sv.Default
	}
	if sv.AllowedValues != nil && len(*sv.AllowedValues) > 0 {
		av := make([]string, 0)
		for _, avv := range *sv.AllowedValues {
			if avv != "" {
				av = append(av, avv)
			}
		}
		if len(av) > 0 {
			ret += " " + strings.Join(av, ",")
		}
	}
	return ret
}

// LoadString restores *StateVariable from scpd tag
func (sv *StateVariable) LoadString(s string) error {
	var err error
	if s == "" {
		return fmt.Errorf("scpd: empty tag")
	}
	parts := strings.SplitN(s, " ", 2)
	main := strings.Split(parts[0], ",")
	if len(main) < 2 {
		return fmt.Errorf("scpd: not enough parts ('%s')", s)
	}
	rng := &AllowedRange{}
	for i, val := range main {
		if val == "" {
			return fmt.Errorf("scpd: empty part ('%s')", s)
		}
		switch i {
		case 0:
			sv.Name = val
		case 1:
			sv.DataType = val
		default:
			if val == "events" {
				sv.Events = "yes"
			} else {
				p := strings.SplitN(val, "=", 2)
				if len(p) != 2 {
					return fmt.Errorf("scpd: invalid part: %s ('%s')", val, s)
				}
				switch p[0] {
				case "min":
					if rng.Min, err = strconv.Atoi(p[1]); err != nil {
						return fmt.Errorf("scpd: invalid min: %s ('%s')", val, s)
					}
				case "max":
					if rng.Max, err = strconv.Atoi(p[1]); err != nil {
						return fmt.Errorf("scpd: invalid max: %s ('%s')", val, s)
					}
				case "step":
					if rng.Step, err = strconv.Atoi(p[1]); err != nil {
						return fmt.Errorf("scpd: invalid step: %s ('%s')", val, s)
					}
				case "default":
					sv.Default = p[1]
				}
			}
		}
	}
	if sv.Events == "" {
		sv.Events = "no"
	}
	if rng.Min != 0 || rng.Max != 0 || rng.Step != 0 {
		sv.AllowedRange = rng
	}
	if len(parts) == 2 {
		val := strings.Trim(parts[1], " ")
		if val != "" {
			values := strings.Split(val, ",")
			sv.AllowedValues = &values
		}
	}
	return nil
}
