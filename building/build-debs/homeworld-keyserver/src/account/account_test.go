package account

import (
	"testing"
	"sort"
)

func TestGroup_AllMembers(t *testing.T) {
	g1 := &Group{Members: []string {"a", "b", "c"}}
	g2 := &Group{Members: []string {"d"}, Inherit: g1}
	g3 := &Group{Members: []string {}, Inherit: g2}
	g4a := &Group{Members: []string {"efghiateounth"}, Inherit: g3}
	g4b := &Group{Members: []string {"kc", "kd", "ke", "kf"}, Inherit: g3}

	groups := []*Group { g1, g2, g3, g4a, g4b }
	expects := [][]string {
		{"a", "b", "c"},
		{"a", "b", "c", "d"},
		{"a", "b", "c", "d"},
		{"a", "b", "c", "d", "efghiateounth"},
		{"a", "b", "c", "d", "kc", "kd", "ke", "kf"},
	}

	for i, group := range groups {
		expect := expects[i]
		mems := group.AllMembers()
		if len(expect) != len(mems) {
			t.Errorf("Wrong length for group %v", group)
		} else {
			sort.Strings(expect)
			sort.Strings(mems)
			for i, expect := range expect {
				found := mems[i]
				if expect != found {
					t.Errorf("Group member mismatch: %v versus %v", expect, found)
				}
			}
		}
	}
}

func TestGroup_HasMember(t *testing.T) {
	options := []string {"a", "b", "c", "d", "efghiateounth", "kc", "kd", "ke", "kf", "kg"}

	g1 := &Group{Members: []string {"a", "b", "c"}}
	g2 := &Group{Members: []string {"d"}, Inherit: g1}
	g3 := &Group{Members: []string {}, Inherit: g2}
	g4a := &Group{Members: []string {"efghiateounth"}, Inherit: g3}
	g4b := &Group{Members: []string {"kc", "kd", "ke", "kf"}, Inherit: g3}

	groups := []*Group { g1, g2, g3, g4a, g4b }
	expects := [][]string {
		{"a", "b", "c"},
		{"a", "b", "c", "d"},
		{"a", "b", "c", "d"},
		{"a", "b", "c", "d", "efghiateounth"},
		{"a", "b", "c", "d", "kc", "kd", "ke", "kf"},
	}

	for i, group := range groups {
		expect := expects[i]
		tried := make(map[string]bool)
		for _, expectelem := range expect {
			if !group.HasMember(expectelem) {
				t.Errorf("Expected group %v to contain %s", group, expectelem)
			}
			tried[expectelem] = true
		}
		for _, option := range options {
			if !tried[option] {
				if group.HasMember(option) {
					t.Errorf("Expected group %v to not contain %s", group, option)
				}
			}
		}
	}
}
