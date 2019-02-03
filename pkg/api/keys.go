package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	// "crypto/sha512"
	"encoding/hex"

	"github.com/rs/zerolog/log"
	"go.etcd.io/etcd/client"
)

type AgentCredentials struct {
	Identity string
	Secret   *string
}

func GenerateAgentCredentials(domain string) (*AgentCredentials, error) {
	log.Debug().Str("domain", domain).Msg("Generating a new client Identity.")
	bitLength := 2048
	r := rand.Reader

	privateKey, err := rsa.GenerateKey(r, bitLength)
	if err != nil {
		return nil, err
	}
	publicKey := privateKey.PublicKey

	privateKeySerialized := x509.MarshalPKCS1PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	privateKeyHex := hex.EncodeToString(privateKeySerialized)

	publicKeySerialized := x509.MarshalPKCS1PublicKey(&publicKey)
	if err != nil {
		return nil, err
	}
	publicKeyHex := hex.EncodeToString(publicKeySerialized)

	// msg := sha512.Sum512(publicKeySerialized)
	// label := []byte("")
	// hash := sha512.New()

	// Challenge, err := rsa.EncryptOAEP(hash, r, &publicKey, msg[:], label)
	// if err != nil {
	// 	return nil, err
	// }
	// ChallengeHex := hex.EncodeToString(Challenge)

	credentials := &AgentCredentials{
		Identity: publicKeyHex,
		Secret:   &privateKeyHex,
	}
	log.Debug().
		Str("domain", domain).
		Str("Identity", publicKeyHex).
		Msg("Generated account credentials.")
	return credentials, nil
}

func PersistAgentCredentials(etcd client.KeysAPI, creds AgentCredentials) error {
	_, err := etcd.Set(context.Background(), fmt.Sprintf("/agents/%s", creds.Identity), "{}", nil)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("Could not persist agent in etcd.")
		return err
	}
	return nil
}
