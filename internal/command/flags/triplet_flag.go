package flags

import (
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
	"github.com/urfave/cli/v2"
)

func TripletValidator(cliCtx *cli.Context, parsedValue string) error {
	_, err := utils.ParseTriplet(parsedValue)
	if err != nil {
		return trace.Wrap(err, "failed to parse triplet flag with value %q", parsedValue)
	}

	return nil
}
