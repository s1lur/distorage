package usecase

import (
	"crypto/aes"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/wealdtech/go-merkletree/keccak256"
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

func (c *CryptoUC) VerifyAddress(aesKey []byte, nonce []byte, sig []byte, pubKeyBytes []byte) error {
	if len(nonce) != aes.BlockSize {
		return errors.New(fmt.Sprintf(
			"nonce has incorrect length %d, which is not equal to the block size %d",
			len(nonce),
			aes.BlockSize,
		),
		)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return err
	}
	encNonce := make([]byte, aes.BlockSize)
	block.Encrypt(nonce, encNonce)
	keccak := keccak256.New()
	encNonceHash := keccak.Hash(encNonce)
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return err
	}
	switch v := pubKey.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(v, encNonceHash[:], sig) {
			return errors.New("signature check failed")
		}
	default:
		return fmt.Errorf("recived wrong key type: %T", v)
	}
	return nil
}

func (c *CryptoUC) GetAddress(pubKeyBytes []byte) []byte {
	keccak := keccak256.New()
	return keccak.Hash(pubKeyBytes)[12:]
}
