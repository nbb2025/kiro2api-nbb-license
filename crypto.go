package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var (
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
)

func initKeys(dataDir string) error {
	privPath := filepath.Join(dataDir, "private.key")
	pubPath := filepath.Join(dataDir, "public.key")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	privData, errPriv := os.ReadFile(privPath)
	pubData, errPub := os.ReadFile(pubPath)

	if errPriv == nil && errPub == nil {
		priv, err := base64.StdEncoding.DecodeString(string(privData))
		if err != nil {
			return fmt.Errorf("decode private key: %w", err)
		}
		pub, err := base64.StdEncoding.DecodeString(string(pubData))
		if err != nil {
			return fmt.Errorf("decode public key: %w", err)
		}
		privateKey = ed25519.PrivateKey(priv)
		publicKey = ed25519.PublicKey(pub)
		fmt.Printf("Loaded existing keys\n")
	} else {
		var err error
		publicKey, privateKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return fmt.Errorf("generate keys: %w", err)
		}
		if err := os.WriteFile(privPath, []byte(base64.StdEncoding.EncodeToString(privateKey)), 0600); err != nil {
			return fmt.Errorf("save private key: %w", err)
		}
		if err := os.WriteFile(pubPath, []byte(base64.StdEncoding.EncodeToString(publicKey)), 0644); err != nil {
			return fmt.Errorf("save public key: %w", err)
		}
		fmt.Printf("Generated new key pair\n")
	}

	fmt.Printf("Public Key (base64): %s\n", base64.StdEncoding.EncodeToString(publicKey))
	return nil
}

type License struct {
	LicenseID  string   `json:"license_id"`
	AllowedIPs []string `json:"allowed_ips"`
	IssuedAt   string   `json:"issued_at"`
	ExpiresAt  *string  `json:"expires_at"`
	Signature  string   `json:"signature"`
}

func signLicense(l *License) error {
	tmp := License{
		LicenseID:  l.LicenseID,
		AllowedIPs: l.AllowedIPs,
		IssuedAt:   l.IssuedAt,
		ExpiresAt:  l.ExpiresAt,
	}
	payload, err := json.Marshal(tmp)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	sig := ed25519.Sign(privateKey, payload)
	l.Signature = base64.StdEncoding.EncodeToString(sig)
	return nil
}
