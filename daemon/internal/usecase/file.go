package usecase

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path"
)

const (
	MAGIC_SIZE = 2
	CRC32_SIZE = 4
	ADDR_SIZE  = 20
)

type StorageUC struct {
	basePath string
}

func NewStorageUC(basePath string) *StorageUC {
	return &StorageUC{basePath: basePath}
}

func (f *StorageUC) VerifyFile(contents []byte, addr []byte) error {
	if len(contents) < MAGIC_SIZE+CRC32_SIZE+ADDR_SIZE {
		return errors.New("file too short")
	}
	if !bytes.Equal(contents[:MAGIC_SIZE], []byte{0xd1, 0x57}) {
		return errors.New("file is invalid (wrong magic string)")
	}
	if !bytes.Equal(f.GetAddress(contents), addr) {
		return errors.New("address mismatch")
	}
	buf := new(bytes.Buffer)
	err := binary.Write(
		buf,
		binary.LittleEndian,
		crc32.ChecksumIEEE(contents[:len(contents)-CRC32_SIZE]),
	)
	if err != nil {
		return err
	}
	if !bytes.Equal(buf.Bytes(), contents[len(contents)-CRC32_SIZE:]) {
		return errors.New("crc32 checksum check fail")
	}
	return nil
}

func (f *StorageUC) ReadFile(fileName string, addr []byte) ([]byte, error) {
	contents, err := os.ReadFile(path.Join(f.basePath, fileName))
	if err != nil {
		return nil, err
	}
	if err := f.VerifyFile(contents, addr); err != nil {
		return nil, err
	}

	return contents, nil
}

func (f *StorageUC) StoreFile(fileName string, addr []byte, contents []byte) error {
	file, err := os.Create(path.Join(f.basePath, fileName))
	defer file.Close()
	if err != nil {
		return err
	}
	contents = append(append([]byte{0xd1, 0x57}, addr...), contents...)
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, crc32.ChecksumIEEE(contents))
	if err != nil {
		return err
	}
	contents = append(contents, buf.Bytes()...)
	_, err = file.Write(contents)
	return err
}

func (f *StorageUC) DeleteFile(fileName string, addr []byte) error {
	contents, err := f.ReadFile(fileName, addr)
	if err != nil {
		return err
	}
	if err := f.VerifyFile(contents, addr); err != nil {
		return err
	}
	return os.Remove(path.Join(f.basePath, fileName))
}

func (f *StorageUC) GetAddress(contents []byte) []byte {
	return contents[MAGIC_SIZE : MAGIC_SIZE+ADDR_SIZE]
}

func (f *StorageUC) GetFileContents(contents []byte) []byte {
	return contents[MAGIC_SIZE+ADDR_SIZE : len(contents)-CRC32_SIZE]
}
