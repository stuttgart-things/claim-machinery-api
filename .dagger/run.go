package main

import (
	"context"
	"dagger/dagger/internal/dagger"
)

// RunApi builds and runs the API as a service with configurable mounts and environment
func (m *Dagger) RunApi(
	ctx context.Context,
	// Source directory
	source *dagger.Directory,
	// Profile file to mount into the container
	profileFile *dagger.File,
	// Filename to use for the mounted profile
	// +optional
	// +default="profile.yaml"
	profileFileName string,
	// Repository tag (only used if push is true)
	// +optional
	// +default="ttl.sh/claim-machinery-api:latest"
	repo string,
	// Push to registry
	// +optional
	// +default=false
	push bool,
	// Host port to expose
	// +optional
	// +default=8080
	hostPort int,
) (*dagger.Service, error) {
	// Build the container image
	container, err := m.BuildImageWithKCL(ctx, source, repo, push)
	if err != nil {
		return nil, err
	}

	// Create final container with mounts and environment variables
	finalContainer := container.
		// Mount the profile file
		WithMountedFile("/tmp/"+profileFileName, profileFile).
		// Set environment variables
		WithEnvVariable("TEMPLATE_PROFILE_PATH", "/tmp/"+profileFileName).
		WithEnvVariable("TEMPLATES_DIR", "/tmp")

	// Return as a service
	svc := finalContainer.
		WithExposedPort(hostPort).
		AsService()

	return svc, nil
}
