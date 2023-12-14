package entity

type ChunkInfo struct {
	Number int
	Hash   string
	Nodes  []string
}

type FileInfo struct {
	Name      string
	Hash      string
	Available bool
	Size      int
	Chunks    []ChunkInfo
}
