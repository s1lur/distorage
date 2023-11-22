package usecase

import (
	"crypto/ecdh"
)

type Crypto interface {
	GenerateECDHKey() (*ecdh.PrivateKey, error)
	ExecuteECDH(own *ecdh.PrivateKey, remoteBytes []byte) ([]byte, error)
	VerifyAddress(aesKey []byte, nonce []byte, sig []byte, pubKeyBytes []byte) error
	GetAddress(pubKeyBytes []byte) []byte
}

type Storage interface {
	VerifyFile(contents []byte, addr []byte) error
	ReadFile(fileName string, addr []byte) ([]byte, error)
	StoreFile(fileName string, addr []byte, contents []byte) error
	DeleteFile(fileName string, addr []byte) error
	GetAddress(contents []byte) []byte
	GetFileContents(contents []byte) []byte
	CheckExistence(fileName string) bool
	CanBeStored(fileName string, addr []byte) bool
}
