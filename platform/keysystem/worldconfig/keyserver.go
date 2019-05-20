package worldconfig

import (
	"fmt"
	"github.com/pkg/errors"
	"net"
	"os"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/verifier"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
)

type Groups struct {
	KerberosAccounts *account.Group
	RootAdmins       *account.Group
	Nodes            *account.Group
	SupervisorNodes  *account.Group
	WorkerNodes      *account.Group
	MasterNodes      *account.Group
}

func GenerateGroups(context *config.Context) Groups {
	kerberosAccountsGroup := &account.Group{
		Name: "kerberos-accounts",
	}
	nodesGroup := &account.Group{
		Name: "nodes",
	}
	groups := Groups{
		KerberosAccounts: kerberosAccountsGroup,
		Nodes:            nodesGroup,
		RootAdmins: &account.Group{
			Name:       "root-admins",
			SubgroupOf: kerberosAccountsGroup,
		},
		SupervisorNodes: &account.Group{
			Name:       "supervisor-nodes",
			SubgroupOf: nodesGroup,
		},
		WorkerNodes: &account.Group{
			Name:       "worker-nodes",
			SubgroupOf: nodesGroup,
		},
		MasterNodes: &account.Group{
			Name:       "master-nodes",
			SubgroupOf: nodesGroup,
		},
	}

	return groups
}

func GenerateAccounts(context *config.Context, conf *SpireSetup, groups Groups) error {
	var accounts []*account.Account

	// TODO: ensure that node hostnames are not duplicated

	for _, node := range conf.Nodes {
		var schedule string
		var group *account.Group
		if node.Kind == "worker" {
			schedule = "true"
			group = groups.WorkerNodes
		} else {
			schedule = "false"
			if node.Kind == "supervisor" {
				group = groups.SupervisorNodes
			} else if node.Kind == "master" {
				group = groups.MasterNodes
			} else {
				return fmt.Errorf("unrecognized kind of node: %s", node.Kind)
			}
		}

		limitIP := net.ParseIP(node.IP)
		if limitIP == nil {
			return fmt.Errorf("invalid IP address: %s", node.IP)
		}

		principal := node.Hostname + "." + conf.Cluster.ExternalDomain

		acc := &account.Account{
			Principal: principal,
			Group:     group,
			LimitIP:   limitIP,
			Metadata: map[string]string{
				"ip":       node.IP,
				"hostname": node.Hostname,
				"schedule": schedule,
				"kind":     node.Kind,
			},
		}
		accounts = append(accounts, acc)

		groups.Nodes.AllMembers = append(groups.Nodes.AllMembers, acc)
		group.AllMembers = append(group.AllMembers, acc)
	}

	// metrics principal used by homeworld-ssh-checker
	allAdmins := append([]string{"metrics@NONEXISTENT.REALM.INVALID"}, conf.RootAdmins...)

	for _, rootAdmin := range allAdmins {
		if rootAdmin == "" {
			return errors.New("cannot have an admin with an unnamed account")
		}
		// TODO: ensure that root admins are unique, including against the metrics admin
		acc := &account.Account{
			Principal:         rootAdmin,
			DisableDirectAuth: true,
			Group:             groups.RootAdmins,
			Metadata:          map[string]string{},
		}
		accounts = append(accounts, acc)
		groups.RootAdmins.AllMembers = append(groups.RootAdmins.AllMembers, acc)
		groups.KerberosAccounts.AllMembers = append(groups.KerberosAccounts.AllMembers, acc)
	}

	if len(conf.RootAdmins) > 0 {
		for _, node := range conf.Nodes {
			if node.Kind == "supervisor" {
				principal := "host/" + node.Hostname + "." + conf.Cluster.ExternalDomain + "@" + conf.Cluster.KerberosRealm
				acc := &account.Account{
					Principal:         principal,
					DisableDirectAuth: true,
					Group:             groups.KerberosAccounts,
					Metadata:          map[string]string{},
				}
				accounts = append(accounts, acc)
				groups.KerberosAccounts.AllMembers = append(groups.KerberosAccounts.AllMembers, acc)
			}
		}
	}

	for _, ac := range accounts {
		ac.Metadata["principal"] = ac.Principal
		context.Accounts[ac.Principal] = ac
	}
	return nil
}

type Authorities struct {
	Keygranting    *authorities.TLSAuthority
	ServerTLS      *authorities.TLSAuthority
	ClusterTLS     *authorities.TLSAuthority
	SshUser        *authorities.SSHAuthority
	SshHost        *authorities.SSHAuthority
	EtcdServer     *authorities.TLSAuthority
	EtcdClient     *authorities.TLSAuthority
	Kubernetes     *authorities.TLSAuthority
	ServiceAccount *authorities.StaticAuthority
}

func GenerateAuthorities(conf *SpireSetup) map[string]config.ConfigAuthority {
	var presentAs []string
	for _, node := range conf.Nodes {
		if node.Kind == "supervisor" {
			presentAs = append(presentAs, node.Hostname+"."+conf.Cluster.ExternalDomain)
		}
	}

	return map[string]config.ConfigAuthority{
		AuthenticationAuthority: {
			Type: "TLS",
			Key:  "keygrant.key",
			Cert: "keygrant.pem",
		},
		ServerTLS: {
			Type:      "TLS",
			Key:       "server.key",
			Cert:      "server.pem",
			PresentAs: presentAs,
		},
		"clustertls": {
			Type: "TLS",
			Key:  "cluster.key",
			Cert: "cluster.cert",
		},
		"ssh-user": {
			Type: "SSH",
			Key:  "ssh_user_ca",
			Cert: "ssh_user_ca.pub",
		},
		"ssh-host": {
			Type: "SSH",
			Key:  "ssh_host_ca",
			Cert: "ssh_host_ca.pub",
		},
		"etcd-server": {
			Type: "TLS",
			Key:  "etcd-server.key",
			Cert: "etcd-server.pem",
		},
		"etcd-client": {
			Type: "TLS",
			Key:  "etcd-client.key",
			Cert: "etcd-client.pem",
		},
		"kubernetes": {
			Type: "TLS",
			Key:  "kubernetes.key",
			Cert: "kubernetes.pem",
		},
		"serviceaccount": {
			Type: "static",
			Key:  "serviceaccount.key",
			Cert: "serviceaccount.pem",
		},
	}
}

func GenerateGrants(context *config.Context, conf *SpireSetup, groups Groups, auth Authorities) error {
	grants := map[string]config.ConfigGrant{
		// ADMIN ACCESS TO THE RUNNING CLUSTER

		"access-ssh": {
			Group: groups.RootAdmins,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				return account.NewSSHGrantPrivilege(
					auth.SshUser, false, 4*time.Hour,
					"temporary-ssh-grant-"+ac.Principal, []string{"root"},
				)
			},
		},

		"access-etcd": {
			Group: groups.RootAdmins,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				return account.NewTLSGrantPrivilege(
					auth.EtcdClient, false, 4*time.Hour,
					"temporary-etcd-grant-"+ac.Principal, nil,
				)
			},
		},

		"access-kubernetes": {
			Group: groups.RootAdmins,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				return account.NewTLSGrantPrivilege(
					auth.Kubernetes, false, 4*time.Hour,
					"temporary-kube-grant-"+ac.Principal, nil,
				)
			},
		},

		// MEMBERSHIP IN THE CLUSTER

		"bootstrap": {
			Group: groups.RootAdmins,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				return account.NewBootstrapPrivilege(groups.Nodes, time.Hour, context.TokenVerifier.Registry)
			},
		},

		"bootstrap-keyinit": {
			Group: groups.SupervisorNodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				if context.TokenVerifier.Registry == nil {
					panic("expected registry to exist")
				}
				return account.NewBootstrapPrivilege(groups.Nodes, time.Hour, context.TokenVerifier.Registry)
			},
		},

		"renew-keygrant": {
			Group: groups.Nodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				return account.NewTLSGrantPrivilege(auth.Keygranting, false, OneDay*40, ac.Principal, nil)
			},
		},

		"auth-to-kerberos": { // integration with kerberos gateway
			Group: groups.SupervisorNodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				return account.NewImpersonatePrivilege(context.GetAccount, groups.KerberosAccounts)
			},
		},

		// CONFIGURATION ENDPOINT

		"get-local-config": {
			Group: groups.Nodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				hostname := ac.Metadata["hostname"]
				ip := ac.Metadata["ip"]
				schedule := ac.Metadata["schedule"]
				kind := ac.Metadata["kind"]
				return account.NewConfigurationPrivilege(
					`# generated automatically by keyserver
HOST_NODE=` + hostname + `
HOST_DNS=` + hostname + `.` + conf.Cluster.ExternalDomain + `
HOST_IP=` + ip + `
SCHEDULE_WORK=` + schedule + `
KIND=` + kind,
				)
			},
		},

		// SERVER CERTIFICATES

		"grant-ssh-host": {
			Group: groups.Nodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				hostname := ac.Metadata["hostname"]
				ip := ac.Metadata["ip"]
				return account.NewSSHGrantPrivilege(
					auth.SshHost, true, OneDay*60, "admitted-"+ac.Principal,
					[]string{
						hostname + "." + conf.Cluster.ExternalDomain,
						hostname,
						ip,
					},
				)
			},
		},

		"grant-kubernetes-master": {
			Group: groups.MasterNodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				hostname := ac.Metadata["hostname"]
				ip := ac.Metadata["ip"]
				return account.NewTLSGrantPrivilege(
					auth.Kubernetes, true, 30*OneDay, "kube-master-"+hostname,
					[]string{
						hostname + "." + conf.Cluster.ExternalDomain,
						hostname,
						"kubernetes",
						"kubernetes.default",
						"kubernetes.default.svc",
						"kubernetes.default.svc." + conf.Cluster.InternalDomain,
						ip,
						conf.Addresses.ServiceAPI,
					},
				)
			},
		},

		"grant-etcd-server": {
			Group: groups.MasterNodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				hostname := ac.Metadata["hostname"]
				ip := ac.Metadata["ip"]
				return account.NewTLSGrantPrivilege(
					auth.EtcdServer, true, 30*OneDay, "etcd-server-"+hostname,
					[]string{
						hostname + "." + conf.Cluster.ExternalDomain,
						hostname,
						ip,
					},
				)
			},
		},

		"grant-registry-host": {
			Group: groups.SupervisorNodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				hostname := ac.Metadata["hostname"]
				return account.NewTLSGrantPrivilege(
					auth.ClusterTLS, true, 30*OneDay, "homeworld-supervisor-"+hostname,
					[]string{"homeworld.private"},
				)
			},
		},

		// CLIENT CERTIFICATES

		"grant-kubernetes-worker": {
			Group: groups.Nodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				hostname := ac.Metadata["hostname"]
				ip := ac.Metadata["ip"]
				return account.NewTLSGrantPrivilege(
					auth.Kubernetes, true, 30*OneDay, "kube-worker-"+hostname,
					[]string{
						hostname + "." + conf.Cluster.ExternalDomain,
						hostname,
						ip,
					},
				)
			},
		},

		"grant-etcd-client": {
			Group: groups.MasterNodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				hostname := ac.Metadata["hostname"]
				ip := ac.Metadata["ip"]
				return account.NewTLSGrantPrivilege(auth.EtcdClient, false, 30*OneDay, "etcd-client-"+hostname,
					[]string{
						hostname + "." + conf.Cluster.ExternalDomain,
						hostname,
						ip,
					},
				)
			},
		},

		"fetch-serviceaccount-key": {
			Group: groups.MasterNodes,
			Specialize: func(ac *account.Account, context *config.Context) account.Privilege {
				return account.NewFetchKeyPrivilege(auth.ServiceAccount)
			},
		},
	}

	for api, grant := range grants {
		privileges := map[string]account.Privilege{}
		for _, ac := range grant.Group.AllMembers {
			privileges[ac.Principal] = grant.Specialize(ac, context)
		}
		context.Grants[api] = privileges
	}
	return nil
}

func ValidateStaticFiles(context *config.Context) error {
	for _, static := range context.StaticFiles {
		// check for existence
		openfile, err := os.Open(static.Filepath)
		if err != nil {
			return err
		}
		err = openfile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

const AuthorityKeyDirectory = "/etc/homeworld/keyserver/authorities/"
const ClusterConfigPath = "/etc/homeworld/keyserver/static/cluster.conf"
const MachineListPath = "/etc/homeworld/keyserver/static/machine.list"
const AuthenticationAuthority = "keygranting"
const ServerTLS = "servertls"

func GenerateConfig() (*config.Context, error) {
	conf, err := LoadSpireSetup(paths.SpireSetupPath)
	if err != nil {
		return nil, err
	}

	context := &config.Context{
		TokenVerifier: verifier.NewTokenVerifier(),
		StaticFiles: map[string]config.StaticFile{
			"cluster.conf": {
				Filename: "cluster.conf",
				Filepath: ClusterConfigPath,
			},
			"machine.list": {
				Filename: "machine.list",
				Filepath: MachineListPath,
			},
		},
		Authorities: map[string]authorities.Authority{},
		Accounts:    map[string]*account.Account{},
		Grants:      map[string]map[string]account.Privilege{},
	}
	err = ValidateStaticFiles(context)
	if err != nil {
		return nil, err
	}
	for name, authority := range GenerateAuthorities(conf) {
		loaded, err := authority.Load(AuthorityKeyDirectory)
		if err != nil {
			return nil, err
		}
		context.Authorities[name] = loaded
	}
	auth := Authorities{
		Keygranting:    context.Authorities[AuthenticationAuthority].(*authorities.TLSAuthority),
		ServerTLS:      context.Authorities[ServerTLS].(*authorities.TLSAuthority),
		ClusterTLS:     context.Authorities["clustertls"].(*authorities.TLSAuthority),
		EtcdClient:     context.Authorities["etcd-client"].(*authorities.TLSAuthority),
		EtcdServer:     context.Authorities["etcd-server"].(*authorities.TLSAuthority),
		Kubernetes:     context.Authorities["kubernetes"].(*authorities.TLSAuthority),
		ServiceAccount: context.Authorities["serviceaccount"].(*authorities.StaticAuthority),
		SshHost:        context.Authorities["ssh-host"].(*authorities.SSHAuthority),
		SshUser:        context.Authorities["ssh-user"].(*authorities.SSHAuthority),
	}
	context.AuthenticationAuthority = auth.Keygranting
	context.ServerTLS = auth.ServerTLS
	groups := GenerateGroups(context)
	err = GenerateAccounts(context, conf, groups)
	if err != nil {
		return nil, err
	}
	err = GenerateGrants(context, conf, groups, auth)
	if err != nil {
		return nil, err
	}
	return context, nil
}
