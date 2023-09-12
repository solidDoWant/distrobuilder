package flags

import (
	"os"

	"github.com/gravitational/trace"
	"github.com/urfave/cli/v2"
)

func ExistingDirValidator(cliCtx *cli.Context, path string) error {
	pathInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return trace.Wrap(err, "directory %q does not exist", path)
		}

		return trace.Wrap(err, "failed to query directory info for %q", path)
	}

	if !pathInfo.IsDir() {
		return trace.Wrap(err, "the %q must be a directory", path)
	}

	return nil
}
