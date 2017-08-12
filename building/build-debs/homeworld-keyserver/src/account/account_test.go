package account

import (
	"testing"
)

func TestGroup_HasMember(t *testing.T) {
	options := []string{"a", "b", "c", "d", "efghiateounth", "kc", "kd", "ke", "kf", "kg"}

	g1 := &Group{AllMembers: []string{"a", "b", "c", "d", "efghiateounth", "kc", "kd", "ke", "kf"}}
	g2 := &Group{AllMembers: []string{"d", "efghiateounth", "kc", "kd", "ke", "kf"}, SubgroupOf: g1}
	g3 := &Group{AllMembers: []string{"efghiateounth", "kc", "kd", "ke", "kf"}, SubgroupOf: g2}
	g4a := &Group{AllMembers: []string{"efghiateounth"}, SubgroupOf: g3}
	g4b := &Group{AllMembers: []string{"kc", "kd", "ke", "kf"}, SubgroupOf: g3}

	groups := []*Group{g1, g2, g3, g4a, g4b}

	for _, group := range groups {
		expect := group.AllMembers
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
