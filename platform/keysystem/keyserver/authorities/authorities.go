package authorities

/*
 * Roughly speaking, the point of this package is to abstract away the details of how different kinds of certificate
 * authorities actually work.
 *
 * An authority, roughly speaking, is an entity that is capable of issuing certain certificates.
 *
 * There are two kinds of authorities in this package:
 * - TLS authorities. These are used for issuing TLS certificates for HTTPS connections (both on the server end and
 *   the client end)
 * - SSH authorities. These are used for issuing SSH certificates to use to access cluster nodes over SSH, or
 *   authenticate as a node itself.
 *
 * A privilege can issue certificates from a SSH authority or a TLS authority.
 *
 * All authorities can have their public keys be downloaded by anyone, with no authentication, and these are produced
 * by the GetPublicKey() method.
 */

type Authority interface {
	GetPublicKey() []byte
}
