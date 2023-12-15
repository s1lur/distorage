package commands

import (
	"cli/config"
	"cli/internal/usecase"
	"github.com/urfave/cli/v2"
)

// uploaded file info file format
/*
{
	"uuid" : {
		"name": "cat.jpg",
		"keccak": "file_hash",
		"size": file_size,
		"chunks": [
			{
				"number": chunk_number
				"hash": "chunk_hash",
				"nodes": [
					"node_1_addr",
					"node_2_addr"
				]
			},
			{
				"number": chunk_number
				"hash": "chunk_hash",
				"nodes": [
					"node_3_addr",
					"node_4_addr"
				]
			}
		]
	},
	"uuid" : {
		"name": "dog.jpg",
		"id": "uuid",
		"keccak": "file_hash",
		"size": file_size,
		"chunks": [
			{
				"number": chunk_number
				"hash": "chunk_hash",
				"nodes": [
					"node_1_addr",
					"node_3_addr"
				]
			},
			{
				"number": chunk_number
				"hash": "chunk_hash",
				"nodes": [
					"node_2_addr",
					"node_4_addr"
				]
			}
		]
	}
}
*/

const CHUNK_SIZE = 1 << (10 * 2) // 1 MB

type Commands struct {
	cfg     *config.Config
	crypto  usecase.Crypto
	server  usecase.Server
	storage usecase.Storage
}

func NewCommands(cfg *config.Config, c usecase.Crypto, s usecase.Server, st usecase.Storage) *Commands {
	return &Commands{cfg: cfg, crypto: c, server: s, storage: st}
}

func InitCommandOnly(c usecase.Crypto, s usecase.Storage) []*cli.Command {
	commands := NewCommands(nil, c, nil, s)
	return []*cli.Command{commands.GetInitCommand()}
}

func (c *Commands) GetCommands() []*cli.Command {
	return []*cli.Command{
		c.GetUploadCommand(),
		c.GetDownloadCommand(),
		c.GetListCommand(),
		c.GetDeleteCommand(),
		c.GetInitCommand(),
	}
}
