package account

import (
	"net"
)

type Account struct {
	Principal         string
	Group             *Group
	DisableDirectAuth bool
	Metadata          map[string]string
	LimitIP           net.IP
}

type Group struct {
	Name    string
	Members []string
	Inherit *Group
}

// order not guaranteed
func (g *Group) AllMembers() []string {
	members := make([]string, 0, 10)
	for g != nil {
		for _, member := range g.Members {
			members = append(members, member)
		}
		g = g.Inherit
	}
	return members
}

func (g *Group) HasMember(user string) bool {
	for g != nil {
		for _, member := range g.Members {
			if member == user {
				return true
			}
		}
		g = g.Inherit
	}
	return false
}
