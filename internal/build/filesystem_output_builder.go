package build

type IFilesystemOutputBuilder interface {
	SetOutputDirectoryPath(string)
	GetOutputDirectoryPath() string
}

type FilesystemOutputBuilder struct {
	OutputDirectoryPath string
}

func (fob *FilesystemOutputBuilder) SetOutputDirectoryPath(outputDirectoryPath string) {
	fob.OutputDirectoryPath = outputDirectoryPath
}

func (fob *FilesystemOutputBuilder) GetOutputDirectoryPath() string {
	return fob.OutputDirectoryPath
}
