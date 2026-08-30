package utils

import (
	"context"
	"errors"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/errs"
	"github.com/regclient/regclient/types/ref"
)

// CheckManifestHead performs a HEAD request against the registry to check
// whether the image hash tag already exists.
//
// The error, if any, is classified with errors.Is against the sentinel
// values in regclient/types/errs so callers can tell "the tag genuinely does
// not exist yet" (errs.ErrNotFound - the expected case that simply means
// "build it") apart from a registry problem - an expired token, a rate
// limit, a 5xx - that looks identical to a naive nil-check but should not be
// silently treated as a cache miss.
//
// Return values:
//   - (true, nil) the manifest exists on the registry.
//   - (false, nil) the manifest genuinely does not exist (errs.ErrNotFound).
//   - (false, err) the check failed; err wraps errs.ErrHTTPUnauthorized,
//     errs.ErrHTTPRateLimit, or another error. exists is always false
//     alongside a non-nil error, so a caller that ignores the error keeps
//     today's original behaviour of falling through to a build. See
//     BuildDockerImage and --strict-registry for how the error can instead
//     abort the build.
func CheckManifestHead(tag string, ref ref.Ref, client *regclient.RegClient) (bool, error) {
	mOpts := []regclient.ManifestOpts{}
	LogInfo("Checking for the image hash %s on the registry\n", tag)
	_, manifestError := client.ManifestHead(context.Background(), ref, mOpts...)

	if manifestError == nil {
		return true, nil
	}

	switch {
	case errors.Is(manifestError, errs.ErrNotFound):
		LogInfo("The image hash %s does not exist on the registry\n", tag)
		return false, nil
	case errors.Is(manifestError, errs.ErrHTTPUnauthorized):
		LogError("Failed to authenticate with the registry while checking for the image hash %s. Please check your --docker-username and --docker-password flags, or that you are already logged in via `docker login`.\n", tag)
		return false, manifestError
	case errors.Is(manifestError, errs.ErrHTTPRateLimit):
		LogError("The registry's rate limit was hit while checking for the image hash %s. Please check your registry provider's request limits.\n", tag)
		return false, manifestError
	default:
		LogError("An error occurred while checking for the image hash %s on the registry: %s\n", tag, manifestError)
		return false, manifestError
	}
}
