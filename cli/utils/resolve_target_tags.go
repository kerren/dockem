package utils

import "fmt"

// TagReason records which branch of the shared tag-resolution rule produced a
// target image name, so callers (the copy path and the push path) can emit the
// correct per-branch log message without re-implementing the rule.
type TagReason int

const (
	// TagReasonTag is a name derived from an explicit --tag X (becomes X-vVERSION).
	TagReasonTag TagReason = iota
	// TagReasonFallback is the vVERSION fallback used when none of --tag,
	// --latest or --main-version was given.
	TagReasonFallback
	// TagReasonLatest is the "latest" tag added by --latest.
	TagReasonLatest
	// TagReasonMainVersion is the vVERSION tag added by --main-version.
	TagReasonMainVersion
)

// ResolvedTag pairs a fully-qualified target image name with the reason it was
// produced, so the caller can log the appropriate message for that branch.
type ResolvedTag struct {
	ImageName string
	Reason    TagReason
}

// ResolveTargetTags applies the single tag-resolution rule shared by the copy
// path (CopyExistingImageTag) and the push path (TagAndPushNewImages). It
// returns the fully-qualified target image names, via GenerateDockerImageName,
// in the exact order they must be produced — callers append these to
// buildLog.outputTags in this order, and the e2e assertions depend on it.
//
// The rule: each --tag X becomes X-vVERSION; if none of --tag, --latest or
// --main-version was given it falls back to vVERSION; --latest adds latest;
// --main-version adds vVERSION.
func ResolveTargetTags(params BuildDockerImageParams, version string) []ResolvedTag {
	resolved := []ResolvedTag{}
	for _, tag := range params.Tag {
		tagVersion := fmt.Sprintf("%s-%s", tag, version)
		resolved = append(resolved, ResolvedTag{
			ImageName: GenerateDockerImageName(params.Registry, params.ImageName, tagVersion),
			Reason:    TagReasonTag,
		})
	}
	if len(params.Tag) == 0 && !params.Latest && !params.MainVersion {
		resolved = append(resolved, ResolvedTag{
			ImageName: GenerateDockerImageName(params.Registry, params.ImageName, version),
			Reason:    TagReasonFallback,
		})
	}
	if params.Latest {
		resolved = append(resolved, ResolvedTag{
			ImageName: GenerateDockerImageName(params.Registry, params.ImageName, "latest"),
			Reason:    TagReasonLatest,
		})
	}
	if params.MainVersion {
		resolved = append(resolved, ResolvedTag{
			ImageName: GenerateDockerImageName(params.Registry, params.ImageName, version),
			Reason:    TagReasonMainVersion,
		})
	}
	return resolved
}
