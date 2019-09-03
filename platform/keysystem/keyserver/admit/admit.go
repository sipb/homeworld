package admit

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sipb/homeworld/platform/util/pgpword"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

type AdmitApproval struct {
	Fingerprint string
	Principal   string
}

func (approval *AdmitApproval) Normalize() error {
	fpr, err := NormalizedFingerprint(approval.Fingerprint)
	if err != nil {
		return err
	}
	approval.Fingerprint = fpr
	return nil
}

type AdmitState struct {
	Seen              bool
	SeenPrincipal     string
	SeenAt            time.Time
	Approved          bool
	ApprovedPrincipal string
	Expiration        time.Time
}

// we require RSA with 2048+ bits to avoid some brute-forcing problems
// TODO: is this actually a reasonable concern?
const MinimumRSABits = 2048

func Fingerprint(key crypto.PublicKey) (string, error) {
	rsakey, ok := key.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("fingerprint expects RSA public key")
	}
	if rsakey.N.BitLen() < MinimumRSABits {
		return "", fmt.Errorf("fingerprint expects 2048+ bits in RSA key, not %d", rsakey.N.BitLen())
	}
	// this isn't an SSH key, but it's convenient to reuse the hashing tools
	pubkey, err := ssh.NewPublicKey(rsakey)
	if err != nil {
		// should never happen
		return "", err
	}
	// based on the implementation of ssh.FingerprintSHA256
	sha256sum := sha256.Sum256(pubkey.Marshal())
	return pgpword.BinToWords(sha256sum[:]), nil
}

// use in addition to Fingerprint; depends on a multiple of 8 words
func WrappedFingerprint(fpr string) string {
	fields := strings.Fields(fpr)
	var lines []string
	for i := 0; i < len(fields); i += 8 {
		line := strings.Join(fields[i:i+8], " ")
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// use when comparing or using a dictionary
func NormalizedFingerprint(fpr string) (string, error) {
	data, err := pgpword.WordsToBin(fpr)
	if err != nil {
		return "", err
	}
	return pgpword.BinToWords(data), nil
}

func LoadFingerprint(path string) (string, error) {
	privkey, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "while reading keyfile")
	}
	key, err := wraputil.LoadRSAKeyFromPEM(privkey)
	if err != nil {
		return "", err
	}
	return Fingerprint(key.Public())
}

const ApprovalExpiration = time.Minute * 5

// TODO: automatically clear out old state so that we don't just grow in size forever
type AdmitChecker struct {
	State        sync.Mutex
	Fingerprints map[string]*AdmitState
	Granting     *authorities.TLSAuthority
	Admits       []Admittable
}

type Admittable struct {
	Address   net.IP
	Principal string
}

func NewAdmitChecker(granting *authorities.TLSAuthority) *AdmitChecker {
	return &AdmitChecker{
		Fingerprints: map[string]*AdmitState{},
		Granting:     granting,
		// Admits should be populated later by the creator
	}
}

func (c *AdmitChecker) PrincipalByAddress(ip net.IP) string {
	for _, admit := range c.Admits {
		if admit.Address.Equal(ip) {
			return admit.Principal
		}
	}
	return ""
}

func (c *AdmitChecker) CheckApproved(fingerprint string, ip net.IP) (string, error) {
	c.State.Lock()
	defer c.State.Unlock()
	principal := c.PrincipalByAddress(ip)
	if principal == "" {
		return "", errors.New("no principal associated with IP")
	}
	admit := c.Fingerprints[fingerprint]
	if admit == nil {
		admit = &AdmitState{}
		c.Fingerprints[fingerprint] = admit
	}
	now := time.Now()
	admit.Seen = true
	admit.SeenAt = now
	admit.SeenPrincipal = principal
	if !admit.Approved {
		return "", errors.New("admit not yet approved")
	}
	if admit.Expiration.Before(now) {
		return "", errors.New("admittance expired")
	}
	if admit.ApprovedPrincipal != principal {
		return "", errors.New("admit permission was for a different principal")
	}
	return principal, nil
}

func (c *AdmitChecker) ListRequests() (result map[string]AdmitState) {
	c.State.Lock()
	defer c.State.Unlock()
	result = map[string]AdmitState{}
	for fingerprint, admit := range c.Fingerprints {
		// deref to take snapshot
		result[fingerprint] = *admit
	}
	return result
}

func (c *AdmitChecker) Approve(approval AdmitApproval) {
	c.State.Lock()
	defer c.State.Unlock()
	admit := c.Fingerprints[approval.Fingerprint]
	if admit == nil {
		admit = &AdmitState{}
		c.Fingerprints[approval.Fingerprint] = admit
	}
	now := time.Now()
	admit.Approved = true
	admit.ApprovedPrincipal = approval.Principal
	admit.Expiration = now.Add(ApprovalExpiration)
}

const OneDay = time.Hour * 24

func (c *AdmitChecker) HandleRequest(body []byte, ip net.IP) ([]byte, error) {
	csr, err := wraputil.LoadX509CSRFromPEM(body)
	if err != nil {
		return nil, err
	}
	err = csr.CheckSignature()
	if err != nil {
		return nil, err
	}
	fpr, err := Fingerprint(csr.PublicKey)
	if err != nil {
		return nil, err
	}
	principal, err := c.CheckApproved(fpr, ip)
	if err != nil {
		return nil, err
	}
	// these settings replicated in the TLS grant privilege for renewal
	response, err := c.Granting.Sign(string(body), false, OneDay*40, principal, nil)
	if err != nil {
		return nil, err
	}
	return []byte(response), nil
}

func PrintRequests(data map[string]AdmitState, writer io.Writer) error {
	if _, err := fmt.Fprintf(writer, "found %d requests:\n", len(data)); err != nil {
		return err
	}
	for fingerprint, admitState := range data {
		if _, err := fmt.Fprintf(writer, "  "); err != nil {
			return err
		}
		if admitState.Seen {
			if _, err := fmt.Fprintf(writer, "[seen] principal %s ", admitState.SeenPrincipal); err != nil {
				return err
			}
		}
		if admitState.Approved {
			if _, err := fmt.Fprintf(writer, "[approved] principal %s ", admitState.ApprovedPrincipal); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(writer, "with fingerprint %s\n", fingerprint); err != nil {
			return err
		}
	}
	return nil
}
