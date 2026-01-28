package main

import (
	"context"
	"dagger/dagger/internal/dagger"
)

// BuildBaseImage creates a base image with KCL pre-installed
func (m *Dagger) BuildBaseImage(ctx context.Context, platform dagger.Platform) (*dagger.Container, error) {
	return dag.Container(dagger.ContainerOpts{Platform: platform}).
			From("ghcr.io/kcl-lang/kcl:v0.12.3").
			WithExec([]string{"kcl", "--version"}), // Verify KCL is available
		nil
}

// BuildImageWithKCL builds the complete application image with KCL and optionally pushes it
func (m *Dagger) BuildImageWithKCL(
	ctx context.Context,
	// Source directory
	source *dagger.Directory,
	// Repository (e.g., ttl.sh/claim-machinery-api:latest)
	// +optional
	// +default="ttl.sh/claim-machinery-api:latest"
	repo string,
	// Push to registry
	// +optional
	// +default=false
	push bool,
) (*dagger.Container, error) {
	platform := dagger.Platform("linux/amd64")

	// Build the Go binary as a static binary using glibc-based image (compatible with Wolfi/KCL base)
	builder := dag.Container(dagger.ContainerOpts{Platform: platform}).
		From("golang:1.25-bookworm").
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "amd64").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"go", "mod", "download"}).
		WithExec([]string{"go", "build", "-trimpath", "-ldflags=-s -w", "-o", "/app/claim-machinery-api", "."})

	// Get the binary
	binary := builder.File("/app/claim-machinery-api")

	// Build base image with KCL
	baseImage, err := m.BuildBaseImage(ctx, platform)
	if err != nil {
		return nil, err
	}

	// Create final image with non-root user
	container := baseImage.
		WithExec([]string{"addgroup", "--gid", "65532", "--system", "nonroot"}).
		WithExec([]string{"adduser", "--uid", "65532", "--system", "--gid", "65532", "--home", "/app", "--no-create-home", "nonroot"}).
		WithFile("/usr/local/bin/claim-machinery-api", binary).
		WithDirectory("/app/config", dag.Directory()).
		WithExec([]string{"chown", "-R", "nonroot:nonroot", "/app"}).
		WithWorkdir("/app").
		WithExposedPort(8080).
		WithUser("nonroot").
		WithEntrypoint([]string{"/usr/local/bin/claim-machinery-api"})

	// Optionally push to registry
	if push {
		_, err := container.Publish(ctx, repo)
		if err != nil {
			return nil, err
		}
	}

	return container, nil
}
