package utils

import "testing"

// These are pure unit tests. ResolveBuilder takes buildx availability as an
// argument rather than probing for it, so none of these need docker installed -
// in particular the multi-platform error path, which is the one that must never
// regress.

func TestResolveBuilderTruthTable(t *testing.T) {
	single := []string{"linux/amd64"}
	multi := []string{"linux/amd64", "linux/arm64"}

	cases := []struct {
		name            string
		builder         string
		platforms       []string
		buildxAvailable bool
		wantBuilder     string
		wantErr         bool
	}{
		// --builder=auto
		{"auto/buildx-present/single", "auto", single, true, "buildx", false},
		{"auto/buildx-present/multi", "auto", multi, true, "buildx", false},
		{"auto/buildx-present/none", "auto", nil, true, "buildx", false},
		{"auto/buildx-absent/single", "auto", single, false, "docker", false},
		{"auto/buildx-absent/none", "auto", nil, false, "docker", false},
		{"auto/buildx-absent/multi", "auto", multi, false, "", true},

		// empty preference behaves exactly like auto
		{"empty/buildx-present/multi", "", multi, true, "buildx", false},
		{"empty/buildx-absent/multi", "", multi, false, "", true},

		// --builder=buildx
		{"buildx/present/single", "buildx", single, true, "buildx", false},
		{"buildx/present/multi", "buildx", multi, true, "buildx", false},
		{"buildx/absent/single", "buildx", single, false, "", true},
		{"buildx/absent/multi", "buildx", multi, false, "", true},

		// --builder=docker (buildx availability is irrelevant; it is never probed)
		{"docker/single/buildx-present", "docker", single, true, "docker", false},
		{"docker/single/buildx-absent", "docker", single, false, "docker", false},
		{"docker/none", "docker", nil, false, "docker", false},
		{"docker/multi/buildx-present", "docker", multi, true, "", true},
		{"docker/multi/buildx-absent", "docker", multi, false, "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveBuilder(tc.builder, tc.platforms, tc.buildxAvailable, "v0.36.1")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ResolveBuilder(%q, %v, buildx=%v) = (%q, nil), want an error", tc.builder, tc.platforms, tc.buildxAvailable, got)
				}
				if got != "" {
					t.Fatalf("ResolveBuilder(%q, %v, buildx=%v) returned an error but also a builder %q; want empty", tc.builder, tc.platforms, tc.buildxAvailable, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveBuilder(%q, %v, buildx=%v) returned an unexpected error: %v", tc.builder, tc.platforms, tc.buildxAvailable, err)
			}
			if got != tc.wantBuilder {
				t.Fatalf("ResolveBuilder(%q, %v, buildx=%v) = %q, want %q", tc.builder, tc.platforms, tc.buildxAvailable, got, tc.wantBuilder)
			}
		})
	}
}

// TestResolveBuilderMultiPlatformWithoutBuildxErrors is the load-bearing
// guarantee of this rung, called out on its own: a build that names more than
// one platform must NEVER silently fall back to the classic single-arch builder.
// It must error whether buildx is simply unavailable under --builder=auto, or
// the classic builder was explicitly requested with --builder=docker.
func TestResolveBuilderMultiPlatformWithoutBuildxErrors(t *testing.T) {
	multi := []string{"linux/amd64", "linux/arm64"}

	if _, err := ResolveBuilder("auto", multi, false, ""); err == nil {
		t.Fatalf("multi-platform build with buildx unavailable under --builder=auto must error, got nil")
	}
	if _, err := ResolveBuilder("docker", multi, true, "v0.36.1"); err == nil {
		t.Fatalf("multi-platform build under --builder=docker must error even when buildx is available, got nil")
	}
}
