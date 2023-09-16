package build

import (
	"fmt"
	"path"
	"strings"
)

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

func (trb *ToolchainRequiredBuilder) GetLinkerFlags() map[string]string {
	return map[string]string{
		"LDFLAGS": strings.Join(
			[]string{
				fmt.Sprintf("-fuse-ld=%s", path.Join(trb.ToolchainPath, "ld.lld")), // Ensures that the LLVM linker for the toolchain is used
			},
			" ",
		),
	}
}
