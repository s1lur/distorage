package usecase

import (
	"cli/internal/entity"
	"crypto/ecdh"
	"crypto/ecdsa"
	"github.com/google/uuid"
)

type Crypto interface {
	GenerateECDHKey() (*ecdh.PrivateKey, error)
	ExecuteECDH(own *ecdh.PrivateKey, remoteBytes []byte) ([]byte, error)
	GetAddress(pubKeyBytes []byte) []byte
	AESEncrypt(key []byte, plaintext []byte) ([]byte, error)
	AESDecrypt(key []byte, ciphertext []byte) ([]byte, error)
	Hash(contents []byte) []byte
	ReadECDSAPrivKey() (*ecdsa.PrivateKey, error)
	ReadAesKey() ([]byte, error)
	PrepareVerification(aesKey []byte, ecdsaKey *ecdsa.PrivateKey) ([]byte, error)
}

type Storage interface {
	GetFileInfos() (map[uuid.UUID]entity.FileInfo, error)
	WriteFileInfos(fileInfos map[uuid.UUID]entity.FileInfo) error
	GetFileInfo(uuid uuid.UUID) (*entity.FileInfo, error)
	AppendFileInfo(fileInfo entity.FileInfo) (*uuid.UUID, error)
	DeleteFileInfo(uuid uuid.UUID) error
	UpdateFileInfo(uuid uuid.UUID, fileInfo entity.FileInfo) error
}

type Server interface {
	GetAvailableNodes() (map[string]string, error)
}
