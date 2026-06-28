package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/Eressleep/metrics-tpl/internal/crypto"
)

func main() {
	pubPath := flag.String("pub", "public.pem", "path to save public key")
	privPath := flag.String("priv", "private.pem", "path to save private key")
	bits := flag.Int("bits", 2048, "key size in bits")

	flag.Parse()

	fmt.Printf("Generating RSA key pair with %d bits...\n", *bits)

	privKey, pubKey, err := crypto.GenerateRSAKeyPair(*bits)
	if err != nil {
		log.Fatalf("Failed to generate keys: %v", err)
	}

	if err := crypto.SavePublicKey(*pubPath, pubKey); err != nil {
		log.Fatalf("Failed to save public key: %v", err)
	}

	if err := crypto.SavePrivateKey(*privPath, privKey); err != nil {
		log.Fatalf("Failed to save private key: %v", err)
	}

	absPub, _ := filepath.Abs(*pubPath)
	absPriv, _ := filepath.Abs(*privPath)

	fmt.Printf("✅ Public key saved to: %s\n", absPub)
	fmt.Printf("✅ Private key saved to: %s\n", absPriv)
	fmt.Println("\nИспользование:")
	fmt.Printf("  Агент: -crypto-key %s\n", absPub)
	fmt.Printf("  Сервер: -crypto-key %s\n", absPriv)
}
