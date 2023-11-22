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

var MAGIC = [2]byte{0xd1, 0x57}

// StorageUC это юзкейс для работы с файловой системой
type StorageUC struct {
	basePath string
}

// NewStorageUC создает экземпляр StorageUC для дальнейшей работы
func NewStorageUC(basePath string) *StorageUC {
	return &StorageUC{basePath: basePath}
}

// VerifyFile проверяет целостность переданного содержимого файла
//
// А именно,
//
// 1. Проверяет его длину
// 2. Проверяет magic в начале файла
// 3. Сверяет адрес, записанный в файле с переданным
// 4. Проверяет CRC32-чексумму в конце файла
func (f *StorageUC) VerifyFile(contents []byte, addr []byte) error {
	// проверка длины
	if len(contents) < MAGIC_SIZE+CRC32_SIZE+ADDR_SIZE {
		return errors.New("file too short")
	}
	// проверка magic
	if !bytes.Equal(contents[:MAGIC_SIZE], MAGIC[:]) {
		return errors.New("file is invalid (wrong magic string)")
	}
	// проверка адреса
	if !bytes.Equal(f.GetAddress(contents), addr) {
		return errors.New("address mismatch")
	}

	// проверка CRC32
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

// ReadFile считывает файл из файловой системы и проверяет его целостность
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

// StoreFile сохраняет файл в файловую систему устройства и дописывает в него служебную информацию
func (f *StorageUC) StoreFile(fileName string, addr []byte, contents []byte) error {
	// создание файла для записи
	file, err := os.Create(path.Join(f.basePath, fileName))
	defer file.Close()
	if err != nil {
		return err
	}

	// запись служебной информации
	fileContents := make([]byte, MAGIC_SIZE+ADDR_SIZE+len(contents)+CRC32_SIZE)
	copy(fileContents[:MAGIC_SIZE], MAGIC[:])
	copy(fileContents[MAGIC_SIZE:MAGIC_SIZE+ADDR_SIZE], addr)
	copy(fileContents[MAGIC_SIZE+ADDR_SIZE:MAGIC_SIZE+ADDR_SIZE+len(contents)], contents)
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, crc32.ChecksumIEEE(fileContents))
	if err != nil {
		return err
	}
	copy(fileContents[MAGIC_SIZE+ADDR_SIZE+len(contents):], buf.Bytes())

	// запись файла в файловую систему устройства
	_, err = file.Write(fileContents)
	return err
}

// DeleteFile удаляет файл из файловой системы устройства, предварительно проверяя его целостность
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

// GetAddress получает адрес клиента, записанный в файле
func (f *StorageUC) GetAddress(contents []byte) []byte {
	return contents[MAGIC_SIZE : MAGIC_SIZE+ADDR_SIZE]
}

// GetFileContents получает тело файла
func (f *StorageUC) GetFileContents(contents []byte) []byte {
	return contents[MAGIC_SIZE+ADDR_SIZE : len(contents)-CRC32_SIZE]
}

// CheckExistence проверяет наличие файла с заданным именем в файловой системе
func (f *StorageUC) CheckExistence(fileName string) bool {
	_, err := os.Stat(path.Join(f.basePath, fileName))
	return err == nil
}

// CanBeStored проверяет, может ли файл быть сохранен: либо такого файла не существует,
// либо он принадлежит тому же пользователю, что сохранял его до этого
func (f *StorageUC) CanBeStored(fileName string, addr []byte) bool {
	if !f.CheckExistence(fileName) {
		return true
	}
	_, err := f.ReadFile(fileName, addr)
	return err == nil
}
