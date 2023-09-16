package build

type ISourceBuilder interface {
	SetSourceDirectoryPath(string)
	GetSourceDirectoryPath() string
}

type SourceBuilder struct {
	SourceDirectoryPath string
}

func (sb *SourceBuilder) SetSourceDirectoryPath(sourceDirectoryPath string) {
	sb.SourceDirectoryPath = sourceDirectoryPath
}

func (sb *SourceBuilder) GetSourceDirectoryPath() string {
	return sb.SourceDirectoryPath
}
