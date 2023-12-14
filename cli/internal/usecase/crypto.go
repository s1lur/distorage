package usecase

import (
	"cli/internal/entity"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/wealdtech/go-merkletree/keccak256"
	"io"
	"os"
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

func (c *CryptoUC) ReadECDSAPrivKey(path string) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	keys := &entity.Keys{}
	if err := json.NewDecoder(f).Decode(&keys); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	ecdsaKeyBytes, err := hex.DecodeString(keys.EcdsaKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseECPrivateKey(ecdsaKeyBytes)
}

func (c *CryptoUC) ReadAesKey(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	keys := &entity.Keys{}
	if err := json.NewDecoder(f).Decode(&keys); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return hex.DecodeString(keys.AesKey)
}

func (c *CryptoUC) GetAddress(pubKeyBytes []byte) []byte {
	return c.Hash(pubKeyBytes)[12:]
}

func (c *CryptoUC) AESEncrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

func (c *CryptoUC) AESDecrypt(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (c *CryptoUC) PrepareVerification(aesKey []byte, ecdsaKey *ecdsa.PrivateKey) ([]byte, error) {
	nonce := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	encNonce := make([]byte, aes.BlockSize)
	block.Encrypt(encNonce, nonce)
	keccak := keccak256.New()
	encNonceHash := keccak.Hash(encNonce)
	sig, err := ecdsa.SignASN1(rand.Reader, ecdsaKey, encNonceHash)
	if err != nil {
		return nil, err
	}
	res := make([]byte, len(nonce)+len(sig))
	copy(res[:len(nonce)], nonce)
	copy(res[len(nonce):], sig)
	return res, nil
}

func (c *CryptoUC) Hash(contents []byte) []byte {
	keccak := keccak256.New()
	return keccak.Hash(contents)
}
