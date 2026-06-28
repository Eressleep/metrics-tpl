package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAndLoadKeys(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "public.pem")
	privPath := filepath.Join(tmpDir, "private.pem")

	privKey, pubKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	err = SavePublicKey(pubPath, pubKey)
	if err != nil {
		t.Fatalf("Failed to save public key: %v", err)
	}

	err = SavePrivateKey(privPath, privKey)
	if err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	loadedPubKey, err := LoadPublicKey(pubPath)
	if err != nil {
		t.Fatalf("Failed to load public key: %v", err)
	}

	loadedPrivKey, err := LoadPrivateKey(privPath)
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	plaintext := []byte("secret data")
	ciphertext, err := EncryptRSA(plaintext, loadedPubKey)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := DecryptRSA(ciphertext, loadedPrivKey)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted data doesn't match original. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestEncryptDecryptWithGeneratedKeys(t *testing.T) {
	privKey, pubKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	testData := []byte("test message for encryption")

	encrypted, err := EncryptRSA(testData, pubKey)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := DecryptRSA(encrypted, privKey)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Decrypted data doesn't match original. Expected: %s, Got: %s", testData, decrypted)
	}
}

func TestLoadInvalidKey(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid.pem")
	err := os.WriteFile(tmpFile, []byte("invalid key"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = LoadPublicKey(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid key, got nil")
	}
}
