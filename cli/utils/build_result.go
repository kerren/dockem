package utils

// BuildResult is the machine-readable outcome of a BuildDockerImage run. It
// is the small, stable, exported counterpart to BuildLog: BuildLog carries
// whatever internal detail the test suite needs to assert on the branch a
// build took, whereas BuildResult is the subset that's safe and useful to
// hand to a caller as JSON (see WriteBuildOutput) or as $GITHUB_OUTPUT keys
// (see WriteGitHubOutput).
type BuildResult struct {
	Hash       string   `json:"hash"`
	CacheHit   bool     `json:"cacheHit"`
	Image      string   `json:"image"`      // hashed image name
	Version    string   `json:"version"`    // e.g. v1.0.0
	Tags       []string `json:"tags"`       // fully-qualified, pushed or copied
	PrimaryTag string   `json:"primaryTag"` // first of Tags, for convenience
	Platforms  []string `json:"platforms"`  // populated in Phase 4
	Registry   string   `json:"registry"`
	DurationMs int64    `json:"durationMs"`
}

// Result maps the internal BuildLog onto the exported BuildResult. It is
// safe to call whether or not BuildDockerImage returned an error: BuildLog
// is populated incrementally as the pipeline progresses, so a BuildLog from
// a failed run still carries whatever was computed before the failure (eg.
// the hash, even if the push itself never happened).
func (b BuildLog) Result() BuildResult {
	tags := b.outputTags
	if tags == nil {
		// Prefer an empty slice over nil so this marshals to `[]`, not
		// `null` - downstream tools like `jq '.tags[]'` choke on null.
		tags = []string{}
	}

	primaryTag := ""
	if len(tags) > 0 {
		primaryTag = tags[0]
	}

	return BuildResult{
		Hash:       b.imageHash,
		CacheHit:   b.hashExists,
		Image:      b.hashedImageName,
		Version:    b.version,
		Tags:       tags,
		PrimaryTag: primaryTag,
		Platforms:  []string{},
		Registry:   b.dockerRegistry,
		DurationMs: b.durationMs,
	}
}
