package commands

import (
	"cli/internal/entity"
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/urfave/cli/v2"
	"os"
	"path"
)

func (c *Commands) GetInitCommand() *cli.Command {
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialize the cli (generate necessary keys)",
		Action:  c.init,
	}
}

func (c *Commands) init(cCtx *cli.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error finding home directory")
	}
	folderPath := path.Join(homeDir, ".distorage")
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return err
	}

	storePath := path.Join(folderPath, "store")
	err = os.MkdirAll(storePath, os.ModePerm)
	if err != nil {
		return err
	}

	aesKey := make([]byte, aes.BlockSize)
	_, err = rand.Read(aesKey)
	if err != nil {
		return err
	}
	aesKeyStr := hex.EncodeToString(aesKey)

	curve := elliptic.P256()
	ecdsaKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return err
	}
	ecdsaKeyBytes, err := x509.MarshalECPrivateKey(ecdsaKey)
	if err != nil {
		return err
	}
	ecdsaKeyStr := hex.EncodeToString(ecdsaKeyBytes)

	ecdsaPubKey := ecdsaKey.PublicKey
	ecdsaPubKeyBytes, err := x509.MarshalPKIXPublicKey(&ecdsaPubKey)
	if err != nil {
		return err
	}
	addr := hex.EncodeToString(c.crypto.GetAddress(ecdsaPubKeyBytes))

	daemonConfig := map[string]any{
		"port":       "53591",
		"server_url": "127.0.0.1:8000/connect",
		"base_path":  folderPath,
		"addr":       addr, //адрес вершины в графе системы
	}
	f, err := os.Create(path.Join(folderPath, "daemon.toml"))
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(daemonConfig); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	cliConfig := map[string]any{
		"server_url":        "127.0.0.1:8000/nodes",
		"replication_count": 5,
	}
	f, err = os.Create(path.Join(folderPath, "cli.toml"))
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(cliConfig); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	keys := entity.Keys{
		AesKey:   aesKeyStr,
		EcdsaKey: ecdsaKeyStr,
	}
	f, err = os.Create(path.Join(folderPath, "keys.json"))
	if err := json.NewEncoder(f).Encode(keys); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	f, err = os.Create(path.Join(folderPath, "files.json"))
	if err := json.NewEncoder(f).Encode(map[string]string{}); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	fmt.Printf("Successfuly initialized the app! Your public addr is: %s\n", addr)
	return nil
}
