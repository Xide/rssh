package api

import (
	"fmt"
	"context"
	"crypto/x509"
	"crypto/rand"
	"crypto/rsa"
	// "crypto/sha512"
	"encoding/hex"
	"github.com/rs/zerolog/log"
	"go.etcd.io/etcd/client"
)

type AgentCredentials struct {
	challenge string
	identity string
	secret *string
}

func GenerateAgentCredentials(domain string) (*AgentCredentials, error) {
	log.Debug().Msg("Generating a new client identity.")
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

	// challenge, err := rsa.EncryptOAEP(hash, r, &publicKey, msg[:], label)
	// if err != nil {
	// 	return nil, err
	// }
	// challengeHex := hex.EncodeToString(challenge)
		
	credentials := &AgentCredentials {
		identity: publicKeyHex,
		secret: &privateKeyHex,
	}
	log.Debug().
		Str("Identity", publicKeyHex).
		Msg("Generated account credentials.")
	return credentials, nil
}


func PersistAgentCredentials(etcd client.KeysAPI, creds AgentCredentials, domain string) error {
	_, err := etcd.Set(context.Background(), fmt.Sprintf("/agents/%s/%s", domain, creds.identity), "{}", nil)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("Could not persist agent in etcd.")
		return err
	}
	return nil
}