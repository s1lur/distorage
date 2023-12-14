package usecase

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ServerUC struct {
	serverIpAddr string
}

func NewServerUC(serverIpAddr string) *ServerUC {
	return &ServerUC{
		serverIpAddr: serverIpAddr,
	}
}

func (s *ServerUC) GetAvailableNodes() (map[string]string, error) {
	availableNodes := make(map[string]string)
	resp, err := http.Get(s.serverIpAddr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}
	err = json.NewDecoder(resp.Body).Decode(&availableNodes)
	if err != nil {
		return nil, err
	}
	return availableNodes, nil
}
