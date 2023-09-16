package runners

import (
	"fmt"
	"regexp"

	"github.com/gravitational/trace"
	"golang.org/x/mod/semver"
)

const SemverRegex string = `((?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:-(?:(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?:[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)`

type VersionChecker struct {
	CommandRunner
	VersionRegex    string
	VersionChecker  func(versionCaptureGroups []string) (bool, string, error)
	IgnoreErrorExit bool
	UseStdErr       bool
}

func (vc *VersionChecker) ValidateOrError() error {
	isValidVersion, version, err := vc.IsValidVersion()
	if err != nil {
		return trace.Wrap(err, "failed to check command version via %q", vc.PrettyPrint())
	}

	if !isValidVersion {
		// TODO give a better error message here
		return trace.Errorf("version %q failed version check for %q", version, vc.PrettyPrint())
	}
	return nil
}

func (vc *VersionChecker) IsValidVersion() (bool, string, error) {
	cmdResult, err := Run(vc)
	if err != nil {
		if !vc.IgnoreErrorExit || cmdResult.ExitCode == 0 {
			return false, "", trace.Wrap(err, "failed to run version checker command")
		}
	}

	versionExtractor, err := regexp.Compile(vc.VersionRegex)
	if err != nil {
		return false, "", trace.Wrap(err, "failed to compile version regex %q", vc.VersionRegex)
	}

	outputText := cmdResult.Stdout
	if vc.UseStdErr {
		outputText = cmdResult.Stderr
	}

	extractedVersionMatches := versionExtractor.FindAllStringSubmatch(outputText, -1)
	if len(extractedVersionMatches) == 0 {
		return false, "", trace.Errorf("failed to extract version from command output %q with regex %q", cmdResult.Stdout, vc.VersionRegex)
	}
	if len(extractedVersionMatches) > 1 {
		return false, "", trace.Errorf("found too many versions in command output %q with regex %q", cmdResult.Stdout, vc.VersionRegex)
	}

	extractedVersionGroups := extractedVersionMatches[0]
	didVersionMatch, extractedVersion, err := vc.VersionChecker(extractedVersionGroups)
	if err != nil {
		return didVersionMatch, extractedVersion, trace.Wrap(err, "version checker callback errored")
	}

	return didVersionMatch, extractedVersion, nil
}

func SemverChecker(comparatorCallback func(string) bool) func([]string) (bool, string, error) {
	return func(versionCaptureGroups []string) (bool, string, error) {
		captureGroupCount := len(versionCaptureGroups) - 1 // Passed value will always contain the entire match first
		if captureGroupCount != 1 {
			return false, "", trace.Errorf("expected one capture group, found %d", captureGroupCount)
		}

		version := versionCaptureGroups[1]
		if !semver.IsValid(semverLibConverter(version)) {
			return false, version, nil
		}

		// If no (non-nil) callback was provided, assume any semver value is valid
		passed := true
		if comparatorCallback != nil {
			passed = comparatorCallback(version)
		}
		return passed, version, nil
	}
}

func ValidSemverChecker() func([]string) (bool, string, error) {
	return SemverChecker(nil)
}

func MinSemverChecker(minVersion string) func([]string) (bool, string, error) {
	return SemverChecker(func(parsedVersion string) bool {
		return semver.Compare(semverLibConverter(parsedVersion), semverLibConverter(minVersion)) > 0
	})
}

func ExactSemverChecker(requiredVersion string) func([]string) (bool, string, error) {
	return SemverChecker(func(parsedVersion string) bool {
		return semver.Compare(semverLibConverter(parsedVersion), semverLibConverter(requiredVersion)) == 0
	})
}

func semverLibConverter(version string) string {
	if version[0] == 'v' {
		return version
	}

	return fmt.Sprintf("v%s", version)
}
