package strutil

import (
	"strings"
	"testing"
)

func TestSubstituteVars(t *testing.T) {
	vars := map[string]string{
		"ruby":     "red",
		"sapphire": "blue",
		"quartz":   "rose",
		"pearl":    "white",
		"amethyst": "purple",
		"peridot":  "green",
	}
	for _, test := range []struct {
		input  string
		output string
	}{
		{"with those we (sapphire) (ruby)", "with those we blue red"},
		{"with those we sapphire (ruby)", "with those we sapphire red"},
		{"(amethyst) world network: business (sapphire) online", "purple world network: business blue online"},
		{"(peridot) shrine", "green shrine"},
		{"exe(quartz)", "exerose"},
		{"(peridot).haze", "green.haze"},
		{"(pearl)", "white"},
		{"pearl", "pearl"},
		{"", ""},
	} {
		out, err := SubstituteVars(test.input, vars)
		if err != nil {
			t.Error(err)
		} else if test.output != out {
			t.Errorf("Mismatch between output '%s' and expected output '%s'", out, test.output)
		}
	}
}

func TestSubstituteVars_MismatchParens(t *testing.T) {
	vars := map[string]string{
		"ruby":     "red",
		"sapphire": "blue",
		"quartz":   "rose",
		"pearl":    "white",
		"amethyst": "purple",
		"peridot":  "green",
	}
	for _, test := range []string{
		"with those we (sapphire) (ruby",
		"with those we (sapphire) ruby)",
		"with those we (sapphire) )ruby(",
		"with those we (sapphire) ruby(",
		"with those we (sapphire) )ruby",
		"with those we (sapphire (ruby",
		"with those we )sapphire (ruby",
		"with those we )sapphire) (ruby",
		"with those we )sapphire( ruby",
		"with those we sapphire( ruby",
		"with those we sapphire) ruby",
		"with those we )sapphire ruby",
		"with those we (sapphire ruby",
		"with those we (sapphire ru(by",
		"with those we sapphire ruby(",
		")with those we sapphire ruby(",
		")with those we sapphire ruby",
		")amethyst) world network: business (sapphire) online",
		")amethyst( world network: business )sapphire( online",
		")amethyst( world network: business sapphire online",
		"amethyst( world network: business sapphire online",
		"(",
		")",
	} {
		_, err := SubstituteVars(test, vars)
		if err == nil {
			t.Errorf("Expected an error in %s!", test)
		} else if !strings.Contains(err.Error(), "parenthes") {
			t.Errorf("Expected error in %s to be about parentheses, not %s!", test, err)
		}
	}
}

func TestSubstituteVars_MissingVars(t *testing.T) {
	_, err := SubstituteVars("(principal)", map[string]string{})
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "principal") {
		t.Error("Expected error to talk about missing variable name.")
	}
}

func TestSubstituteAllVars(t *testing.T) {
	vars := map[string]string{
		"ruby":     "red",
		"sapphire": "blue",
		"quartz":   "rose",
		"pearl":    "white",
		"amethyst": "purple",
		"peridot":  "green",
	}
	tests := []string{"(pearl): a history of sapphic (quartz)", "howling (amethyst)", "their (sapphire) understanding", "frolic (ruby)"}
	result, err := SubstituteAllVars(tests, vars)
	if err != nil {
		t.Error(err)
	} else {
		expected := "white: a history of sapphic rose//howling purple//their blue understanding//frolic red"
		if strings.Join(result, "//") != expected {
			t.Errorf("Result mismatch: expected %v, not %v", strings.Split(expected, "//"), result)
		}
	}
}

func TestSubstituteAllVars_Fail(t *testing.T) {
	vars := map[string]string{
		"ruby":     "red",
		"sapphire": "blue",
		"quartz":   "rose",
		"pearl":    "white",
		"amethyst": "purple",
		"peridot":  "green",
	}
	tests := []string{"(pearl): a history of sapphic (quartz)", "howling (missingvar)", "their (sapphire) understanding", "frolic (ruby)"}
	_, err := SubstituteAllVars(tests, vars)
	if err == nil {
		t.Error("Expected failure!")
	} else if !strings.Contains(err.Error(), "missingvar") {
		t.Error("Expected mention of missing variable name")
	}
}
