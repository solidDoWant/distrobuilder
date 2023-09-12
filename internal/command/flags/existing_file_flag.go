package flags

import (
	"os"

	"github.com/gravitational/trace"
	"github.com/urfave/cli/v2"
)

func ExistingFileValidator(cliCtx *cli.Context, path string) error {
	pathInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return trace.Wrap(err, "file %q does not exist", path)
		}

		return trace.Wrap(err, "failed to query file info for %q", path)
	}

	if !pathInfo.Mode().IsRegular() {
		return trace.Wrap(err, "the %q must be a regular file", path)
	}

	return nil
}
