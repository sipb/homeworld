package config

import (
	"testing"
)

const MINIMAL_YAML = `
authoritydir: /etc/hyades/keyserver/authorities/
staticdir: /etc/hyades/keyserver/static/
authentication-authority: keygranting
servertls: servertls

staticfiles:
  - cluster.conf
  - README.txt

authorities:
  keygranting:
    type: TLS
    key: keygrant.key
    cert: keygrant.pem

  ssh-host:
    type: SSH
    key: ssh_host_ca
    cert: ssh_host_ca.pub

accounts:
  - principal: ruby-01.mit.edu
    group: example-nodes
    limit-ip: true
    metadata:
      ip: 18.181.0.97
      hostname: ruby-01

  - principal: cela@ATHENA.MIT.EDU
    disable-direct-auth: true
    group: root-admins

groups:
  root-admins:
  nodes:
  example-nodes:
    inherit: nodes

grants:
  # GRANTS!

  test-1:
    group: root-admins
    privilege: sign-ssh
    scope: creep
    authority: ssh-user
    lifespan: 4 hours
    ishost: false
    common-name: temporary-ssh-grant-(principal)
    allowed-names:
    - (hostname).mit.edu
    - (hostname)
    - (ip)
    contents: |
      # generated automatically by keyserver
      HOST_NODE=(hostname)
      HOST_DNS=(hostname).mit.edu
      HOST_IP=(ip)
`

const BROKEN_YAML = "nope-this-is-wrong: nah"

func TestParseConfig(t *testing.T) {
	config, err := parseConfigFromBytes([]byte(MINIMAL_YAML))
	if err != nil {
		t.Error(err)
	} else {
		if config.AuthorityDir != "/etc/hyades/keyserver/authorities/" {
			t.Error("Wrong authoritydir.")
		}
		if config.StaticDir != "/etc/hyades/keyserver/static/" {
			t.Error("Wrong staticdir.")
		}
		if config.AuthenticationAuthority != "keygranting" {
			t.Errorf("Wrong authenticationauthority '%s'.", config.AuthenticationAuthority)
		}
		if config.ServerTLS != "servertls" {
			t.Error("Wrong servertls.")
		}
		if len(config.StaticFiles) != 2 || config.StaticFiles[0] != "cluster.conf" || config.StaticFiles[1] != "README.txt" {
			t.Error("Wrong staticfiles.")
		}
		if len(config.Authorities) != 2 {
			t.Error("Wrong # of authorities.")
		} else if _, found := config.Authorities["keygranting"]; !found {
			t.Error("Expected keygranting authority.")
		} else if _, found := config.Authorities["ssh-host"]; !found {
			t.Error("Expected ssh-host authority.")
		} else {
			keygranting := config.Authorities["keygranting"]
			sshhost := config.Authorities["ssh-host"]
			if keygranting.Type != "TLS" {
				t.Error("Wrong type of authority.")
			}
			if sshhost.Type != "SSH" {
				t.Error("Wrong type of authority.")
			}
			if keygranting.Key != "keygrant.key" {
				t.Error("Wrong authority key.")
			}
			if sshhost.Key != "ssh_host_ca" {
				t.Error("Wrong authority key.")
			}
			if keygranting.Cert != "keygrant.pem" {
				t.Error("Wrong authority cert.")
			}
			if sshhost.Cert != "ssh_host_ca.pub" {
				t.Error("Wrong authority cert.")
			}
		}
		if len(config.Accounts) != 2 {
			t.Error("Wrong number of accounts")
		} else {
			ruby := config.Accounts[0]
			cela := config.Accounts[1]
			if ruby.Principal != "ruby-01.mit.edu" {
				t.Error("Wrong principal.")
			}
			if cela.Principal != "cela@ATHENA.MIT.EDU" {
				t.Error("Wrong principal.")
			}
			if ruby.Group != "example-nodes" {
				t.Error("Wrong group.")
			}
			if cela.Group != "root-admins" {
				t.Error("Wrong group.")
			}
			if ruby.DisableDirectAuth {
				t.Error("Wrong disable direct auth.")
			}
			if !cela.DisableDirectAuth {
				t.Error("Wrong disable direct auth.")
			}
			if !ruby.LimitIP {
				t.Error("Wrong limit ip.")
			}
			if cela.LimitIP {
				t.Error("Wrong limit ip.")
			}
			if len(ruby.Metadata) != 2 {
				t.Error("Wrong metadata count.")
			} else if _, found := ruby.Metadata["ip"]; !found {
				t.Error("Could not find 'ip' field.")
			} else if _, found := ruby.Metadata["hostname"]; !found {
				t.Errorf("Could not find 'hostname' field in %v.", ruby.Metadata)
			} else {
				if ruby.Metadata["ip"] != "18.181.0.97" {
					t.Error("Wrong metadata IP")
				}
				if ruby.Metadata["hostname"] != "ruby-01" {
					t.Error("Wrong metadata hostname")
				}
			}
			if len(cela.Metadata) != 0 {
				t.Error("Wrong metadata count.")
			}
		}
		if len(config.Groups) != 3 {
			t.Error("Expected three groups.")
		} else if _, found := config.Groups["root-admins"]; !found {
			t.Error("Missing root-admins.")
		} else if _, found := config.Groups["nodes"]; !found {
			t.Error("Missing nodes.")
		} else if _, found := config.Groups["example-nodes"]; !found {
			t.Error("Missing example-nodes.")
		} else {
			if config.Groups["root-admins"].Inherit != "" {
				t.Error("Expected empty inherit for root-admins.")
			}
			if config.Groups["nodes"].Inherit != "" {
				t.Error("Expected empty inherit for nodes.")
			}
			if config.Groups["example-nodes"].Inherit != "nodes" {
				t.Error("Expected 'nodes' inherit for example-nodes.")
			}
		}
		if len(config.Grants) != 1 {
			t.Error("Expected one grant.")
		} else {
			grant, found := config.Grants["test-1"]
			if !found {
				t.Error("Expected to find grant test-1.")
			} else {
				if grant.Group != "root-admins" {
					t.Error("Wrong group.")
				}
				if grant.Privilege != "sign-ssh" {
					t.Error("Wrong privilege.")
				}
				if grant.Scope != "creep" {
					t.Errorf("Wrong scope '%s'.", grant.Scope)
				}
				if grant.Authority != "ssh-user" {
					t.Error("Wrong authority.")
				}
				if grant.Lifespan != "4 hours" {
					t.Error("Wrong lifespan.")
				}
				if grant.IsHost != "false" {
					t.Error("Wrong ishost.")
				}
				if grant.CommonName != "temporary-ssh-grant-(principal)" {
					t.Error("Wrong common-name.")
				}
				if len(grant.AllowedNames) != 3 {
					t.Error("Wrong allowed-names length.")
				} else {
					if grant.AllowedNames[0] != "(hostname).mit.edu" {
						t.Error("Wrong allowed-name.")
					}
					if grant.AllowedNames[1] != "(hostname)" {
						t.Error("Wrong allowed-name.")
					}
					if grant.AllowedNames[2] != "(ip)" {
						t.Error("Wrong allowed-name.")
					}
				}
				if grant.Contents != "# generated automatically by keyserver\nHOST_NODE=(hostname)\nHOST_DNS=(hostname).mit.edu\nHOST_IP=(ip)\n" {
					t.Error("Wrong contents.")
				}
			}
		}
	}
}

func TestParseConfig_Fail(t *testing.T) {
	_, err := parseConfigFromBytes([]byte(BROKEN_YAML))
	if err == nil {
		t.Error("Expected yaml failure.")
	}
}
