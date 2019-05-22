package account

import (
	"net"
)

type Account struct {
	Principal         string
	DisableDirectAuth bool
	LimitIP           net.IP
	Privileges        map[string]Privilege
}

type Group struct {
	AllMembers []*Account
}

func (g *Group) HasMember(user string) bool {
	for _, member := range g.AllMembers {
		if member.Principal == user {
			return true
		}
	}
	return false
}
