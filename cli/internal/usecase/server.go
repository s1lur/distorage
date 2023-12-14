package usecase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ServerUC struct {
	serverAddr string
}

func NewServerUC(serverIpAddr string) *ServerUC {
	u := url.URL{Scheme: "http", Host: serverIpAddr, Path: "/nodes"}
	return &ServerUC{
		serverAddr: u.String(),
	}
}

func (s *ServerUC) GetAvailableNodes() (map[string]string, error) {
	availableNodes := make(map[string]string)
	resp, err := http.Get(s.serverAddr)
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
