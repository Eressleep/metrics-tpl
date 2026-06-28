package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	if path == "" {
		return nil, errors.New("crypto key path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var pubKey *rsa.PublicKey

	if parsed, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		var ok bool
		pubKey, ok = parsed.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA type")
		}
		return pubKey, nil
	}

	if parsed, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return parsed, nil
	}

	return nil, errors.New("unsupported public key format")
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, errors.New("crypto key path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var privateKey *rsa.PrivateKey

	if parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return parsed, nil
	}

	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		var ok bool
		privateKey, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA type")
		}
		return privateKey, nil
	}

	return nil, errors.New("unsupported private key format")
}

func EncryptRSA(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return nil, errors.New("public key is nil")
	}

	if len(data) == 0 {
		return nil, errors.New("data is empty")
	}

	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, publicKey, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return ciphertext, nil
}

func DecryptRSA(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key is nil")
	}

	if len(ciphertext) == 0 {
		return nil, errors.New("ciphertext is empty")
	}

	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

func SavePublicKey(path string, publicKey *rsa.PublicKey) error {
	if publicKey == nil {
		return errors.New("public key is nil")
	}

	data, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: data,
	}

	return os.WriteFile(path, pem.EncodeToMemory(block), 0644)
}

func SavePrivateKey(path string, privateKey *rsa.PrivateKey) error {
	if privateKey == nil {
		return errors.New("private key is nil")
	}

	data, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: data,
	}

	return os.WriteFile(path, pem.EncodeToMemory(block), 0600)
}
