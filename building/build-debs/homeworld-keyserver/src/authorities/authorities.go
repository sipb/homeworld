package authorities

import "net/http"

/*
 * Roughly speaking, the point of this package is to abstract away the details of how different kinds of certificate
 * authorities actually work.
 *
 * An authority, roughly speaking, is an entity that is capable of issuing certain certificates, which can later be
 * verified to have been issued by that entity.
 *
 * In this context, it is both "anything that can be a source of authenticating a client" and "anything that can issue
 * certificates on request". Most of these only do one or the other.
 *
 * There are three kinds of authorities in this package:
 * - TLS authorities. These are used both for issuing TLS certificates for HTTPS connections (both on the server end and
 *   the client end) and for authenticating that a particular client connecting to the keyserver holds a certificate for
 *   this authority (such as a keygranting certificate).
 * - Delegated authorities. These are used for authentication, but don't directly check auth themselves: this is done
 *   indirectly, since these always involve first authenticating to another authority, and then using a privilege to
 *   use an account authenticated by this authority.
 * - SSH authorities. These are used for issuing SSH certificates to use to access cluster nodes over SSH, or
 *   authenticate as a node itself.
 *
 * An account can be authenticated by a delegated authority or a TLS authority.
 * A privilege can issue certificates from a SSH authority or a TLS authority.
 *
 * All authorities can have their public keys be downloaded by anyone, with no authentication, and these are produced
 * by the GetPublicKey() method.
 */

type Authority interface {
	GetPublicKey() []byte
	AsVerifier() Verifier // might return nil if this isn't a verifier
}

type Verifier interface {
	HasAttempt(request *http.Request) bool
	Verify(request *http.Request) (principal string, err error)
}
