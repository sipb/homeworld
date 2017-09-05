package strutil

import (
	"fmt"
	"strings"
)

// substitutes variable references of the form "(VAR)". As in: "(hostname).mit.edu"
func SubstituteVars(within string, vars map[string]string) (string, error) {
	parts := strings.Split(within, "(")
	if strings.Contains(parts[0], ")") {
		return "", fmt.Errorf("Extraneous close parenthesis in substitution string '%s'", within)
	}
	snippets := []string{parts[0]}
	for _, part := range parts[1:] {
		subparts := strings.Split(part, ")")
		if len(subparts) < 2 {
			return "", fmt.Errorf("Missing close parenthesis in substitution string '%s'", within)
		}
		if len(subparts) > 2 {
			return "", fmt.Errorf("Extraneous close parenthesis in substitution string '%s'", within)
		}
		varname, text := subparts[0], subparts[1]
		value := vars[varname]
		if value == "" {
			return "", fmt.Errorf("Undefined variable %s in substitution string '%s'", varname, within)
		}
		snippets = append(snippets, value)
		snippets = append(snippets, text)
	}
	return strings.Join(snippets, ""), nil
}

func SubstituteAllVars(within []string, vars map[string]string) ([]string, error) {
	out := make([]string, len(within))
	for i, str := range within {
		value, err := SubstituteVars(str, vars)
		if err != nil {
			return nil, err
		}
		out[i] = value
	}
	return out, nil
}
