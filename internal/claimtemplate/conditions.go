package claimtemplate

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

// Parameter conditions (#135).
//
// showWhen and enumWhen let a template say when a parameter applies and
// which values it allows given another parameter. Values are compared as
// strings (fmt.Sprint), so a boolean parameter matches "true" / "false".
//
//	showWhen: {use_case: [dev]}
//	    The parameter applies only when use_case is dev. Several keys must
//	    all match (AND); the listed values are alternatives (OR). A
//	    multiselect controlling parameter matches when one of its selected
//	    values is listed.
//
//	enumWhen: {use_case: {dev: [large, xlarge], rke2: [medium, large, xlarge]}}
//	    While use_case is dev, the parameter allows only large and xlarge.
//	    A value of use_case that is not listed leaves the parameter's enum
//	    as it is. Several controlling parameters narrow it further (the
//	    allowed values are the intersection).
//
// A parameter whose showWhen does not match is inactive: clients hide it,
// the API ignores a value sent for it (its default, if any, still reaches
// KCL, exactly as when a client never sent it) and it has no value for the
// conditions of other parameters. The value a condition looks at is the
// one sent, else the parameter's default.

// FieldErrors maps a parameter name to what is wrong with its value.
type FieldErrors map[string]string

// ValidationError is returned by CheckParams when the parameters break the
// template's rules; the API answers it with 400 and the fields.
type ValidationError struct {
	Fields FieldErrors
}

func (e *ValidationError) Error() string {
	names := make([]string, 0, len(e.Fields))
	for n := range e.Fields {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = n + ": " + e.Fields[n]
	}
	return "invalid parameters: " + strings.Join(parts, "; ")
}

// HasConditions reports whether the parameter carries showWhen or enumWhen.
func (p Parameter) HasConditions() bool {
	return len(p.ShowWhen) > 0 || len(p.EnumWhen) > 0
}

// valueStrings returns v as the strings conditions compare: one per item of
// a list, else one; nil and missing values have none.
func valueStrings(v interface{}) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []string:
		return t
	case []interface{}:
		out := make([]string, len(t))
		for i, it := range t {
			out[i] = fmt.Sprint(it)
		}
		return out
	default:
		return []string{fmt.Sprint(t)}
	}
}

// isEmpty reports whether v counts as not set: nil, "" or an empty list.
func isEmpty(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []string:
		return len(t) == 0
	case []interface{}:
		return len(t) == 0
	}
	return false
}

// Shown reports whether the parameter's showWhen matches values (the
// effective values of the active parameters, see ActiveParams).
func (p Parameter) Shown(values map[string]interface{}) bool {
	for ctrl, want := range p.ShowWhen {
		have := valueStrings(values[ctrl])
		if !slices.ContainsFunc(have, func(s string) bool { return slices.Contains(want, s) }) {
			return false
		}
	}
	return true
}

// AllowedValues returns the values the parameter allows given values:
// its enum narrowed by enumWhen. nil means no restriction (no enum and no
// matching enumWhen).
func (p Parameter) AllowedValues(values map[string]interface{}) []string {
	allowed := p.Enum
	restricted := len(p.Enum) > 0
	ctrls := make([]string, 0, len(p.EnumWhen))
	for ctrl := range p.EnumWhen {
		ctrls = append(ctrls, ctrl)
	}
	sort.Strings(ctrls)
	for _, ctrl := range ctrls {
		byValue := p.EnumWhen[ctrl]
		for _, v := range valueStrings(values[ctrl]) {
			list, ok := byValue[v]
			if !ok {
				continue
			}
			if !restricted {
				allowed, restricted = list, true
				continue
			}
			var both []string
			for _, a := range allowed {
				if slices.Contains(list, a) {
					both = append(both, a)
				}
			}
			allowed = both
		}
	}
	if !restricted {
		return nil
	}
	if allowed == nil {
		allowed = []string{}
	}
	return allowed
}

// effective returns the value a condition sees for p: the one sent, else
// its default.
func effective(p Parameter, input map[string]interface{}) (interface{}, bool) {
	if v, ok := input[p.Name]; ok && !isEmpty(v) {
		return v, true
	}
	if p.Default != nil {
		return p.Default, true
	}
	return nil, false
}

// ActiveParams evaluates the showWhen conditions of the template against
// input (the parameters sent; defaults fill the rest). It returns which
// parameters are active and the effective values of the active ones. An
// inactive parameter has no value, so a condition on it does not match;
// this is repeated until stable, so chains (A hides B, B shows C) resolve.
func ActiveParams(t *ClaimTemplate, input map[string]interface{}) (map[string]bool, map[string]interface{}) {
	active := make(map[string]bool, len(t.Spec.Parameters))
	for _, p := range t.Spec.Parameters {
		active[p.Name] = true
	}
	values := map[string]interface{}{}
	for range len(t.Spec.Parameters) + 1 {
		values = make(map[string]interface{}, len(t.Spec.Parameters))
		for _, p := range t.Spec.Parameters {
			if !active[p.Name] {
				continue
			}
			if v, ok := effective(p, input); ok {
				values[p.Name] = v
			}
		}
		changed := false
		for _, p := range t.Spec.Parameters {
			if len(p.ShowWhen) == 0 {
				continue
			}
			if shown := p.Shown(values); shown != active[p.Name] {
				active[p.Name] = shown
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return active, values
}

// CheckParams validates the parameters sent for an order against the
// template, before anything is rendered:
//   - a value for a parameter that is inactive (showWhen does not match) is
//     dropped from the result;
//   - a required active parameter needs a value, unless it has a default or
//     a valueFrom function;
//   - a value sent must be one of the allowed values (enum narrowed by
//     enumWhen), match pattern and minLength/maxLength; a boolean parameter
//     takes true/false;
//   - the default of a parameter with enumWhen must be allowed too when no
//     value is sent (e.g. size medium with use_case dev).
//
// Empty strings are left alone (as before: the default or KCL decides).
// input is not modified; the result is a copy without the dropped values.
// The error is a *ValidationError.
func CheckParams(t *ClaimTemplate, input map[string]interface{}) (map[string]interface{}, error) {
	active, values := ActiveParams(t, input)
	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		out[k] = v
	}
	fields := FieldErrors{}
	for _, p := range t.Spec.Parameters {
		if !active[p.Name] {
			delete(out, p.Name)
			continue
		}
		v, sent := input[p.Name]
		if !sent || isEmpty(v) {
			if p.Required && p.Default == nil && p.ValueFrom == nil {
				fields[p.Name] = "is required"
				continue
			}
			if len(p.EnumWhen) > 0 && p.Default != nil {
				if msg := checkAllowed(p, p.Default, values); msg != "" {
					fields[p.Name] = msg + " (default)"
				}
			}
			continue
		}
		if msg := CheckValue(p, v, values); msg != "" {
			fields[p.Name] = msg
		}
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	return out, nil
}

// CheckValue checks one value sent for p against its allowed values (given
// the effective values of the others), type boolean, pattern and length.
// It returns what is wrong, or "".
func CheckValue(p Parameter, v interface{}, values map[string]interface{}) string {
	if msg := checkAllowed(p, v, values); msg != "" {
		return msg
	}
	s, isString := v.(string)
	if p.Type == "boolean" {
		if _, isBool := v.(bool); !isBool && (!isString || (s != "true" && s != "false")) {
			return "must be true or false"
		}
		return ""
	}
	if !isString {
		return ""
	}
	n := utf8.RuneCountInString(s)
	if p.MinLength != nil && n < *p.MinLength {
		return fmt.Sprintf("needs at least %d characters", *p.MinLength)
	}
	if p.MaxLength != nil && n > *p.MaxLength {
		return fmt.Sprintf("allows at most %d characters", *p.MaxLength)
	}
	if p.Pattern != "" {
		if re, err := regexp.Compile(p.Pattern); err == nil && !re.MatchString(s) {
			return fmt.Sprintf("does not match %s", p.Pattern)
		}
	}
	return ""
}

// checkAllowed checks every value of v (one, or each of a list) against
// the parameter's allowed values.
func checkAllowed(p Parameter, v interface{}, values map[string]interface{}) string {
	allowed := p.AllowedValues(values)
	if allowed == nil {
		return ""
	}
	for _, s := range valueStrings(v) {
		if slices.Contains(allowed, s) {
			continue
		}
		if why := p.enumWhenReason(values); why != "" {
			return fmt.Sprintf("%s allows %s, got %s", why, joinOr(allowed), s)
		}
		return fmt.Sprintf("must be one of %s, got %s", strings.Join(allowed, ", "), s)
	}
	return ""
}

// enumWhenReason names the controlling values that narrow the enum, e.g.
// "use_case dev", or "".
func (p Parameter) enumWhenReason(values map[string]interface{}) string {
	var why []string
	ctrls := make([]string, 0, len(p.EnumWhen))
	for ctrl := range p.EnumWhen {
		ctrls = append(ctrls, ctrl)
	}
	sort.Strings(ctrls)
	for _, ctrl := range ctrls {
		for _, v := range valueStrings(values[ctrl]) {
			if _, ok := p.EnumWhen[ctrl][v]; ok {
				why = append(why, ctrl+" "+v)
			}
		}
	}
	return strings.Join(why, " and ")
}

func joinOr(list []string) string {
	switch len(list) {
	case 0:
		return "no value"
	case 1:
		return "only " + list[0]
	}
	return strings.Join(list[:len(list)-1], ", ") + " or " + list[len(list)-1]
}

// CheckConditions returns problems with the template's conditions: a
// showWhen or enumWhen naming an unknown parameter, or an enumWhen value
// outside the parameter's enum. The loaders log them; nothing is refused.
func (t *ClaimTemplate) CheckConditions() []string {
	names := make(map[string]bool, len(t.Spec.Parameters))
	for _, p := range t.Spec.Parameters {
		names[p.Name] = true
	}
	var problems []string
	for _, p := range t.Spec.Parameters {
		for ctrl := range p.ShowWhen {
			if !names[ctrl] {
				problems = append(problems, fmt.Sprintf("parameter %s: showWhen names unknown parameter %s", p.Name, ctrl))
			}
		}
		for ctrl, byValue := range p.EnumWhen {
			if !names[ctrl] {
				problems = append(problems, fmt.Sprintf("parameter %s: enumWhen names unknown parameter %s", p.Name, ctrl))
			}
			if len(p.Enum) == 0 {
				continue
			}
			for v, list := range byValue {
				for _, a := range list {
					if !slices.Contains(p.Enum, a) {
						problems = append(problems, fmt.Sprintf("parameter %s: enumWhen %s=%s allows %s, which is not in its enum", p.Name, ctrl, v, a))
					}
				}
			}
		}
	}
	sort.Strings(problems)
	return problems
}
