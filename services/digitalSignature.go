package services

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func GenerateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}
func SignData(data []byte, privateKey *rsa.PrivateKey) (string, error) {

	hash := sha256.Sum256(data)

	signature, err := rsa.SignPKCS1v15(
		rand.Reader,
		privateKey,
		crypto.SHA256,
		hash[:],
	)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}
func VerifySignature(data []byte, base64Signature string, publicKey *rsa.PublicKey) error {
	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(base64Signature)
	if err != nil {
		return err
	}

	// Hash data again
	hash := sha256.Sum256(data)

	// Verify
	return rsa.VerifyPKCS1v15(
		publicKey,
		crypto.SHA256,
		hash[:],
		signature,
	)
}
func DigitalSignature() {
	document := []byte("I, guardian of the patient, give consent for treatment.")

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	signature, err := SignData(document, privateKey)
	if err != nil {
		panic(err)
	}

	fmt.Println("Digital Signature (Base64):")
	fmt.Println(signature)

	// Verify signature
	err = VerifySignature(document, signature, publicKey)
	if err != nil {
		fmt.Println(" Signature verification failed:", err)
	} else {
		fmt.Println(" Signature verified successfully")
	}

	// --- Tampering demo ---
	tamperedDoc := []byte("I, guardian, DO NOT give consent.")

	err = VerifySignature(tamperedDoc, signature, publicKey)
	if err != nil {
		fmt.Println(" Tampered document detected")
	} else {
		fmt.Println(" This should NOT happen")
	}
}
