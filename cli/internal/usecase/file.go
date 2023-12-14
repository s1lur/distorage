package usecase

import (
	"cli/internal/entity"
	"encoding/json"
	"fmt"
	u "github.com/google/uuid"
	"os"
)

type StorageUC struct {
	fileInfoPath string
}

func NewStorageUC(fileInfoPath string) *StorageUC {
	return &StorageUC{fileInfoPath: fileInfoPath}
}

func (s *StorageUC) GetFileInfos() (map[u.UUID]entity.FileInfo, error) {
	file, err := os.Open(s.fileInfoPath)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	fileInfos := make(map[u.UUID]entity.FileInfo)
	if err := json.NewDecoder(file).Decode(&fileInfos); err != nil {
		return nil, err
	}
	return fileInfos, nil
}

func (s *StorageUC) WriteFileInfos(fileInfos map[u.UUID]entity.FileInfo) error {
	file, err := os.Create(s.fileInfoPath)
	defer file.Close()
	if err != nil {
		return err
	}
	return json.NewEncoder(file).Encode(&fileInfos)
}

func (s *StorageUC) GetFileInfo(uuid u.UUID) (*entity.FileInfo, error) {
	fileInfos, err := s.GetFileInfos()
	if err != nil {
		return nil, err
	}
	fileInfo, ok := fileInfos[uuid]
	if !ok {
		return nil, fmt.Errorf("file %s not found", uuid)
	}
	return &fileInfo, nil
}

func (s *StorageUC) AppendFileInfo(fileInfo entity.FileInfo) error {
	fileInfos, err := s.GetFileInfos()
	if err != nil {
		return err
	}
	fileInfos[u.New()] = fileInfo
	return s.WriteFileInfos(fileInfos)
}

func (s *StorageUC) UpdateFileInfo(uuid u.UUID, fileInfo entity.FileInfo) error {
	fileInfos, err := s.GetFileInfos()
	if err != nil {
		return err
	}
	_, exists := fileInfos[uuid]
	if !exists {
		return fmt.Errorf("file %s not found", uuid)
	}
	fileInfos[uuid] = fileInfo
	return s.WriteFileInfos(fileInfos)

}

func (s *StorageUC) DeleteFileInfo(uuid u.UUID) error {
	fileInfos, err := s.GetFileInfos()
	if err != nil {
		return err
	}
	_, exists := fileInfos[uuid]
	if !exists {
		return fmt.Errorf("file %s not found", uuid)
	}
	delete(fileInfos, uuid)
	return nil
}
