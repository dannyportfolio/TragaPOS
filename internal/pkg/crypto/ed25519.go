package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

var (
	ErrInvalidSignature = errors.New("invalid cryptographic signature")
	ErrInvalidKeySize   = errors.New("invalid ed25519 key size")
	ErrEmptyPayload     = errors.New("payload cannot be empty")
)

// GenerateKeyPair generates a fresh Ed25519 keypair encoded as standard base64 strings.
func GenerateKeyPair() (pubKeyBase64, privKeyBase64 string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ed25519 keypair: %w", err)
	}
	return base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(priv), nil
}

// Sign signs data using an Ed25519 private key (in base64 format).
func Sign(privKeyBase64 string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrEmptyPayload
	}
	privBytes, err := base64.StdEncoding.DecodeString(privKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}
	if len(privBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidKeySize, ed25519.PrivateKeySize, len(privBytes))
	}

	privKey := ed25519.PrivateKey(privBytes)
	sig := ed25519.Sign(privKey, data)
	return base64.RawURLEncoding.EncodeToString(sig), nil
}

// Verify verifies that sigBase64 is a valid Ed25519 signature of data using pubKeyBase64.
func Verify(pubKeyBase64 string, data []byte, sigBase64 string) error {
	if len(data) == 0 {
		return ErrEmptyPayload
	}
	pubBytes, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidKeySize, ed25519.PublicKeySize, len(pubBytes))
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(sigBase64)
	if err != nil {
		// Fallback to StdEncoding in case standard base64 was passed
		sigBytes, err = base64.StdEncoding.DecodeString(sigBase64)
		if err != nil {
			return fmt.Errorf("failed to decode signature: %w", err)
		}
	}

	if !ed25519.Verify(ed25519.PublicKey(pubBytes), data, sigBytes) {
		return ErrInvalidSignature
	}
	return nil
}
