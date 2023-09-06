package command

import "fireman/internal/database"

type Performer interface {
	Actuator(command string) ([]byte, error)
	SudoActuator(command string) ([]byte, error)
	Ping() bool
	OSName() string
	GetError() []byte
	UploadFile(localPath, remotePath string) (int64, error)
	DownloadFile(remotePath string, localPath string) (int64, error)
	Close()
}

type BuilderFunc func(r *database.Resource) (Performer, error)

func GetPerformer(r *database.Resource) (Performer, error) {
	var Builder BuilderFunc
	if r.Protocol == "SSH" {
		Builder = CreateSSHClient
	} else {
		//Builder = CreateWindowsClient
	}
	return Builder(r)
}
