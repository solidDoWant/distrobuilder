package build

type IToolchainRequiredBuilder interface {
	SetToolchainDirectory(string)
	GetToolchainDirectory() string
}

type ToolchainRequiredBuilder struct {
	ToolchainPath string
}

func (trb *ToolchainRequiredBuilder) SetToolchainDirectory(toolchainDirectory string) {
	trb.ToolchainPath = toolchainDirectory
}

func (trb *ToolchainRequiredBuilder) GetToolchainDirectory() string {
	return trb.ToolchainPath
}
