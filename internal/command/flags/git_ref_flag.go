package flags

import (
	"strings"

	"github.com/gravitational/trace"
	"github.com/urfave/cli/v2"
)

func GitRefValidator(cliCtx *cli.Context, parsedValue string) error {
	// HEAD, special case
	if parsedValue == "HEAD" {
		return nil
	}

	lowerParsedValue := strings.ToLower(parsedValue)

	// Check if SHA-1, or short SHA-1
	// Git requires at least 4 characters for the commit short to be unambiguous
	if len(lowerParsedValue) >= 4 {
		isValidSha := true
		for _, parsedValueChar := range lowerParsedValue {
			if !strings.Contains("abcdef0123456789", string(parsedValueChar)) {
				isValidSha = false
				break
			}
		}
		if isValidSha {
			return nil
		}
	}

	// Git ref "alias"
	// Technically a ref is valid even if it's not a head or tag, provided it exists under .git/refs/
	if strings.HasPrefix(lowerParsedValue, "refs/") {
		return nil
	}

	return trace.Errorf("provided git ref %q is not syntactically valid", parsedValue)
}
