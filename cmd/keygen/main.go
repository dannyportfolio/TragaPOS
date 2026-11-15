package main

import (
	"fmt"
	"os"

	"pos-engine/internal/pkg/crypto"
)

func main() {
	pub, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate keypair: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ED25519_PUBLIC_KEY=%s\n", pub)
	fmt.Printf("ED25519_PRIVATE_KEY=%s\n", priv)
}
