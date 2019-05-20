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
	AllMembers []*Account
	SubgroupOf *Group
}

func (g *Group) HasMember(user string) bool {
	for _, member := range g.AllMembers {
		if member.Principal == user {
			return true
		}
	}
	return false
}
