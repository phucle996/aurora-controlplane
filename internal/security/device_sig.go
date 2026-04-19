package security

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	// ErrUnsupportedKeyAlgorithm is returned for an unrecognised key_algorithm value.
	ErrUnsupportedKeyAlgorithm = errors.New("security: unsupported key algorithm")
	// ErrBadPublicKey is returned when a PEM public key cannot be parsed.
	ErrBadPublicKey = errors.New("security: malformed public key")
	// ErrSignatureInvalid is returned when the signature does not match.
	ErrSignatureInvalid = errors.New("security: signature invalid")
)

// Supported key_algorithm values stored in iam.devices.
const (
	AlgEd25519   = "ed25519"
	AlgECDSAP256 = "ecdsa-p256"
)

// VerifyDeviceSignature verifies that `signature` (base64-raw-url encoded) over
// `payload` was produced by the private key corresponding to `pemPublicKey`.
//
// keyAlgorithm must be one of AlgEd25519 or AlgECDSAP256.
// payload is the canonical signing string built by the caller.
func VerifyDeviceSignature(pemPublicKey, keyAlgorithm, payload, signatureB64 string) error {
	sigBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(signatureB64))
	if err != nil {
		return fmt.Errorf("%w: decode signature: %v", ErrSignatureInvalid, err)
	}

	payloadDigest := sha256.Sum256([]byte(payload))

	switch strings.ToLower(strings.TrimSpace(keyAlgorithm)) {
	case AlgEd25519:
		return verifyEd25519(pemPublicKey, payloadDigest[:], sigBytes)
	case AlgECDSAP256:
		return verifyECDSAP256(pemPublicKey, payloadDigest[:], sigBytes)
	default:
		return ErrUnsupportedKeyAlgorithm
	}
}

// CanonicalRefreshPayload builds the deterministic string that the client must
// sign when requesting a token rotation.
//
//	payload = refresh_token + "." + device_id + "." + nonce + "." + timestamp_unix
func CanonicalRefreshPayload(rawToken, deviceID, nonce string, timestampUnix int64) string {
	return fmt.Sprintf("%s.%s.%s.%d", rawToken, deviceID, nonce, timestampUnix)
}

// ── internal verifiers ─────────────────────────────────────────────────────────

func verifyEd25519(pemKey string, digest, sig []byte) error {
	pub, err := parseEd25519PEM(pemKey)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, digest, sig) {
		return ErrSignatureInvalid
	}
	return nil
}

func verifyECDSAP256(pemKey string, digest, sig []byte) error {
	pub, err := parseECDSAP256PEM(pemKey)
	if err != nil {
		return err
	}

	// sig is DER-encoded ECDSA signature (r || s big-endian, 32 bytes each)
	if len(sig) != 64 {
		return fmt.Errorf("%w: expected 64-byte raw P-256 sig", ErrSignatureInvalid)
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])

	if !ecdsa.Verify(pub, digest, r, s) {
		return ErrSignatureInvalid
	}
	return nil
}

func parseEd25519PEM(pemKey string) (ed25519.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemKey)))
	if block == nil {
		return nil, ErrBadPublicKey
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadPublicKey, err)
	}

	ed, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: not an Ed25519 key", ErrBadPublicKey)
	}
	return ed, nil
}

func parseECDSAP256PEM(pemKey string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemKey)))
	if block == nil {
		return nil, ErrBadPublicKey
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadPublicKey, err)
	}

	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: not an ECDSA key", ErrBadPublicKey)
	}
	if ec.Curve != elliptic.P256() {
		return nil, fmt.Errorf("%w: curve is not P-256", ErrBadPublicKey)
	}
	return ec, nil
}
