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
	Name       string
	AllMembers []string
	SubgroupOf *Group
}

func (g *Group) HasMember(user string) bool {
	for _, member := range g.AllMembers {
		if member == user {
			return true
		}
	}
	return false
}
