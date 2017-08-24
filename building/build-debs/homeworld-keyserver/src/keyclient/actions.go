package keyclient

import (
	"errors"
	"fmt"
	"keycommon"
	"strconv"
	"time"
)

func PrepareKeygenAction(m *Mainloop, k ConfigKey) (Action, error) {
	switch k.Type {
	case "tls":
		return TLSKeygenAction{keypath: k.Key, logger: m.logger}, nil
	case "ssh":
		// should probably include creating a .pub file as well
		return nil, errors.New("Unimplemented operation: SSH key generation")
	case "tls-pubkey":
		return nil, nil // key is pregenerated
	case "ssh-pubkey":
		return nil, nil // key is pregenerated
	default:
		return nil, fmt.Errorf("Unrecognized key type: %s", k.Type)
	}
}

func PrepareDownloadAction(m *Mainloop, d ConfigDownload) (Action, error) {
	refresh_period, err := time.ParseDuration(d.Refresh)
	if err != nil {
		return nil, err
	}
	mode, err := strconv.ParseUint(d.Mode, 8, 9)
	if err != nil {
		return nil, err
	}
	if mode&0002 != 0 {
		return nil, fmt.Errorf("Disallowed mode: %o (will not grant world-writable access)", mode)
	}
	switch d.Type {
	case "authority":
		return &DownloadAction{Fetcher: &AuthorityFetcher{Keyserver: m.ks, AuthorityName: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	case "static":
		return &DownloadAction{Fetcher: &StaticFetcher{Keyserver: m.ks, StaticName: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	case "api":
		return &DownloadAction{Fetcher: &APIFetcher{Mainloop: m, API: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	default:
		return nil, fmt.Errorf("Unrecognized download type: %s", d.Type)
	}
}

func PrepareBootstrapAction(m *Mainloop, tokenfilepath string, api string) (Action, error) {
	if api == "" {
		return nil, errors.New("No bootstrap API provided.")
	}
	return &BootstrapAction{Mainloop: m, TokenFilePath: tokenfilepath, TokenAPI: api}, nil
}

func PrepareRequestOrRenewKeys(m *Mainloop, key ConfigKey, inadvance time.Duration) (Action, error) {
	if inadvance <= 0 {
		return nil, errors.New("Invalid in-advance for key renewal.")
	}
	if key.API == "" {
		return nil, errors.New("No renew API provided.")
	}
	switch key.Type {
	case "tls":
		fallthrough
	case "tls-pubkey":
		return &RequestOrRenewAction{Mainloop: m, InAdvance: inadvance, API: key.API, Name: key.Name, CheckExpiration: CheckTLSCertExpiration, GenCSR: keycommon.BuildTLSCSR, KeyFile: key.Key, CertFile: key.Cert}, nil
	case "ssh":
		fallthrough
	case "ssh-pubkey":
		return &RequestOrRenewAction{Mainloop: m, InAdvance: inadvance, API: key.API, Name: key.Name, CheckExpiration: CheckSSHCertExpiration, GenCSR: keycommon.BuildSSHCSR, KeyFile: key.Key, CertFile: key.Cert}, nil
	default:
		return nil, fmt.Errorf("Unrecognized key type: %s", key.Type)
	}
}
