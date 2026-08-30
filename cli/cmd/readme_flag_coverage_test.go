package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// TestEveryBuildFlagIsDocumentedInReadme is the mechanical enforcement of the
// CLAUDE.md convention "The README documents every flag and concept; update
// it alongside any flag change in cli/cmd/build.go". Without this test, a
// flag added to (or renamed in) build.go silently drops out of sync with
// README.md and nothing notices until a user goes looking for it.
//
// Flags are enumerated from the live *cobra.Command via Flags().VisitAll,
// not by re-parsing build.go's source, so this test tracks whatever cobra
// actually registers - including flags cobra adds itself (see the
// knownUndocumented exception below) - rather than only what a human wrote
// as an explicit buildCmd.Flags().___ call.
//
// README.md documents flags by their long form (eg. "--tag"), so each flag's
// long name, prefixed with "--", must appear somewhere in the README text.
// This is deliberately loose (it does not check the description text matches,
// or that short flags like "-t" appear) - the goal is to catch a flag that
// was added or renamed and never mentioned at all, not to validate prose.
func TestEveryBuildFlagIsDocumentedInReadme(t *testing.T) {
	// cobra only registers the auto-generated --help and --version flags
	// inside Command.execute() (called from Execute()), which this test
	// never calls. Force them into existence here so VisitAll sees exactly
	// the flag set a real `dockem build --help` would show - otherwise this
	// test would never have had a chance to catch the --version gap below.
	buildCmd.InitDefaultHelpFlag()
	buildCmd.InitDefaultVersionFlag()

	// Known, deliberate exception: cobra auto-generates -v/--version from
	// buildCmd.Version (see build.go's `Version: Version` field). It is not
	// declared alongside the rest of the flags in the init() block below,
	// and - per the Phase T0 testing-plan note that first flagged this gap -
	// it is genuinely absent from README.md today. Documenting it there is
	// a separate, human decision about README content; this test's job is
	// to guard against *new*, unnoticed gaps, not to silently paper over
	// this pre-existing, already-known one by failing on it forever.
	knownUndocumented := map[string]bool{
		"version": true,
	}

	readmePath := "../../README.md"
	readmeBytes, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("could not read %s to verify flag documentation - if cli/cmd/build.go or the repo layout moved, update this test's relative path: %s", readmePath, err)
	}
	readme := string(readmeBytes)

	var missing []string
	buildCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if knownUndocumented[f.Name] {
			return
		}
		if !strings.Contains(readme, "--"+f.Name) {
			missing = append(missing, f.Name)
		}
	})

	if len(missing) > 0 {
		t.Errorf("the following build flag(s) are registered in cli/cmd/build.go but not mentioned in README.md - update the README alongside any flag change (see CLAUDE.md conventions): %s", strings.Join(missing, ", "))
	}
}
