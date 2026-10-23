package tests

import (
	"crypto/sha256"

	"webtyp.com/base64"
	"webtyp.com/crypto/asym"
)

// buildCreateResponse simulates a browser returning a registration response.
func buildCreateResponse(rpID string, challenge, origin, spki []byte, signCount uint32, credID []byte, flags byte) string {
	cdJSON := `{"type":"webauthn.create","challenge":"` + base64.URLEncode(challenge) + `","origin":"` + string(origin) + `"}`

	rpIDHash := sha256.Sum256([]byte(rpID))

	authData := make([]byte, 0, 37+16+2+len(credID))
	authData = append(authData, rpIDHash[:]...)
	authData = append(authData, flags)
	authData = append(authData, byte(signCount>>24), byte(signCount>>16), byte(signCount>>8), byte(signCount))

	// AAGUID (16 bytes)
	authData = append(authData, make([]byte, 16)...)

	// Credential ID length
	authData = append(authData, byte(len(credID)>>8), byte(len(credID)))

	// Credential ID
	authData = append(authData, credID...)

	respJSON := `{"clientDataJSON":"` + base64.URLEncode([]byte(cdJSON)) + `",` +
		`"authenticatorData":"` + base64.URLEncode(authData) + `",` +
		`"publicKey":"` + base64.URLEncode(spki) + `",` +
		`"publicKeyAlgorithm":-7}`

	credJSON := `{"id":"` + base64.URLEncode(credID) + `",` +
		`"rawId":"` + base64.URLEncode(credID) + `",` +
		`"type":"public-key",` +
		`"response":` + respJSON + `}`

	return base64.URLEncode([]byte(credJSON))
}

// buildGetResponse simulates a browser returning an assertion response.
func buildGetResponse(rpID string, challenge, origin, privKey []byte, signCount uint32, credID []byte, flags byte) string {
	cdJSON := `{"type":"webauthn.get","challenge":"` + base64.URLEncode(challenge) + `","origin":"` + string(origin) + `"}`

	rpIDHash := sha256.Sum256([]byte(rpID))

	authData := make([]byte, 0, 37)
	authData = append(authData, rpIDHash[:]...)
	authData = append(authData, flags)
	authData = append(authData, byte(signCount>>24), byte(signCount>>16), byte(signCount>>8), byte(signCount))

	clientDataHash := sha256.Sum256([]byte(cdJSON))
	msg := make([]byte, 0, len(authData)+len(clientDataHash))
	msg = append(msg, authData...)
	msg = append(msg, clientDataHash[:]...)

	sig, err := asym.Sign(msg, privKey)
	if err != nil {
		panic(err)
	}

	respJSON := `{"clientDataJSON":"` + base64.URLEncode([]byte(cdJSON)) + `",` +
		`"authenticatorData":"` + base64.URLEncode(authData) + `",` +
		`"signature":"` + base64.URLEncode(sig) + `"}`

	credJSON := `{"id":"` + base64.URLEncode(credID) + `",` +
		`"rawId":"` + base64.URLEncode(credID) + `",` +
		`"type":"public-key",` +
		`"response":` + respJSON + `}`

	return base64.URLEncode([]byte(credJSON))
}
