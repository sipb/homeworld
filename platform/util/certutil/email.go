package certutil

import "encoding/asn1"

func OIDEmailAddress() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}
}

func IsOIDEqual(a asn1.ObjectIdentifier, b asn1.ObjectIdentifier) bool {
	if len(a) != len(b) {
		return false
	}
	for i, av := range a {
		if av != b[i] {
			return false
		}
	}
	return true
}
