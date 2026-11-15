package crypto_test

import (
	"testing"
	"time"

	"pos-engine/internal/pkg/crypto"
)

func TestEd25519SignAndVerify(t *testing.T) {
	pubKey, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}

	payload := []byte("TICKET_ID_123|TENANT_PARK|VIP|1|1731300000|1731200000")

	sig, err := crypto.Sign(privKey, payload)
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}

	err = crypto.Verify(pubKey, payload, sig)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}

	tamperedPayload := []byte("TICKET_ID_123|TENANT_PARK|VIP|99|1731300000|1731200000")
	err = crypto.Verify(pubKey, tamperedPayload, sig)
	if err == nil {
		t.Fatal("expected verification failure on tampered payload, but got nil")
	}

	pubKey2, _, _ := crypto.GenerateKeyPair()
	err = crypto.Verify(pubKey2, payload, sig)
	if err == nil {
		t.Fatal("expected verification failure with different public key, but got nil")
	}
}

func TestDeterministicOfflineVerificationSpeed(t *testing.T) {
	pubKey, privKey, _ := crypto.GenerateKeyPair()
	payload := []byte("BENCHMARK_TICKET_ID|TENANT_ZOO|REGULAR|2|1735000000|1731000000")
	sig, _ := crypto.Sign(privKey, payload)

	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		if err := crypto.Verify(pubKey, payload, sig); err != nil {
			t.Fatalf("iteration %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	t.Logf("Verified %d Ed25519 ticket signatures in %v (avg %v per verify)",
		iterations, elapsed, elapsed/time.Duration(iterations))
}
