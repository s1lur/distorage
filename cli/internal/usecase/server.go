package usecase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ServerUC struct {
	serverURL string
}

func NewServerUC(serverURL string) *ServerUC {
	u := url.URL{Scheme: "http", Host: serverURL}
	serverURL, _ = url.PathUnescape(u.String())
	return &ServerUC{
		serverURL: serverURL,
	}
}

func (s *ServerUC) GetAvailableNodes() (map[string]string, error) {
	availableNodes := make(map[string]string)
	resp, err := http.Get(s.serverURL)
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
