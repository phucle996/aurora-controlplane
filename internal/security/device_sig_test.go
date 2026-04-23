package security

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
)

func TestNormalizeDeviceKeyAlgorithmCanonicalizesES256(t *testing.T) {
	t.Parallel()

	got, err := NormalizeDeviceKeyAlgorithm("ES256")
	if err != nil {
		t.Fatalf("normalize algorithm: %v", err)
	}
	if got != AlgECDSAP256 {
		t.Fatalf("expected %q, got %q", AlgECDSAP256, got)
	}
}

func TestValidateDevicePublicKeyAcceptsECDSAP256AndEd25519(t *testing.T) {
	t.Parallel()

	ecdsaPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	gotAlg, err := ValidateDevicePublicKey(encodeECDSAPublicKeyPEM(&ecdsaPriv.PublicKey), "ES256")
	if err != nil {
		t.Fatalf("validate ecdsa public key: %v", err)
	}
	if gotAlg != AlgECDSAP256 {
		t.Fatalf("expected canonical ecdsa algorithm, got %q", gotAlg)
	}

	edPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	gotAlg, err = ValidateDevicePublicKey(encodeEd25519PublicKeyPEM(edPub), AlgEd25519)
	if err != nil {
		t.Fatalf("validate ed25519 public key: %v", err)
	}
	if gotAlg != AlgEd25519 {
		t.Fatalf("expected canonical ed25519 algorithm, got %q", gotAlg)
	}
}

func TestValidateDevicePublicKeyRejectsMalformedPEM(t *testing.T) {
	t.Parallel()

	if _, err := ValidateDevicePublicKey("not-a-pem", "ES256"); err == nil {
		t.Fatal("expected malformed PEM to be rejected")
	}
}

func TestVerifyDeviceSignatureAcceptsES256Alias(t *testing.T) {
	t.Parallel()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}

	payload := CanonicalRefreshPayload(
		"jti-123",
		123,
		"POST",
		"https://controlplane.example.com/api/v1/auth/refresh",
		"refresh-token-hash",
		"device-123",
	)
	digest := sha256.Sum256([]byte(payload))
	r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("sign payload: %v", err)
	}
	sig := rawECDSASignature(r, s)

	if err := VerifyDeviceSignature(encodeECDSAPublicKeyPEM(&priv.PublicKey), "ES256", payload, base64.RawURLEncoding.EncodeToString(sig)); err != nil {
		t.Fatalf("verify device signature with ES256 alias: %v", err)
	}
}

func TestCanonicalRefreshPayloadIncludesDPoPLiteFields(t *testing.T) {
	t.Parallel()

	got := CanonicalRefreshPayload(
		"jti-123",
		1710000000,
		"post",
		"https://controlplane.example.com/api/v1/auth/refresh",
		"refresh-token-hash",
		"device-123",
	)

	want := "jti-123\n1710000000\nPOST\nhttps://controlplane.example.com/api/v1/auth/refresh\nrefresh-token-hash\ndevice-123"
	if got != want {
		t.Fatalf("unexpected canonical payload\nwant: %q\ngot:  %q", want, got)
	}
}

func encodeECDSAPublicKeyPEM(pub *ecdsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func encodeEd25519PublicKeyPEM(pub ed25519.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func rawECDSASignature(r, s *big.Int) []byte {
	sig := make([]byte, 64)
	rb := r.Bytes()
	sb := s.Bytes()
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):], sb)
	return sig
}
