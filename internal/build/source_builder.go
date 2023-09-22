package build

type ISourceBuilder interface {
	SetSourceDirectoryPath(string)
	GetSourceDirectoryPath() string
}

type SourceBuilder struct {
	SourceDirectoryPath string
}

func (sb *SourceBuilder) SetSourceDirectoryPath(sourceDirectoryPath string) {
	// Default value
	if sourceDirectoryPath == "" {
		sb.SourceDirectoryPath = "/tmp/source" // TODO move this to appropriate place under /var
		return
	}

	sb.SourceDirectoryPath = sourceDirectoryPath
}

func (sb *SourceBuilder) GetSourceDirectoryPath() string {
	return sb.SourceDirectoryPath
}
