package usecase

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/x509"
	"fmt"
)

type CryptoUC struct {
}

func NewCryptoUC() *CryptoUC {
	return &CryptoUC{}
}

func (c *CryptoUC) GenerateECDHKey() (*ecdh.PrivateKey, error) {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func (c *CryptoUC) ExecuteECDH(own *ecdh.PrivateKey, remoteBytes []byte) ([]byte, error) {
	remote, err := x509.ParsePKIXPublicKey(remoteBytes)
	if err != nil {
		return nil, err
	}
	switch v := remote.(type) {
	case *ecdh.PublicKey:
		return own.ECDH(v)
	default:
		return nil, fmt.Errorf("recived wrong key type: %T", v)
	}
}
