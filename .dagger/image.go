package main

import (
	"context"
	"dagger/dagger/internal/dagger"
)

// BuildImage builds a container image using ko
func (m *Dagger) BuildImage(
	ctx context.Context,
	src *dagger.Directory,
	// +optional
	// +default="ko.local"
	repo string,
	// +optional
	// +default="."
	buildArg string,
	// +optional
	// +default="v0.18.1"
	koVersion string,
	// +optional
	// +default="true"
	push string,
	// +optional
	// +default="GITHUB_TOKEN"
	tokenName string,
	// +optional
	token *dagger.Secret,
	// +optional
	// +default=false
	scan bool,
	// +optional
	// +default="HIGH,CRITICAL"
	scanSeverity string,
) (string, error) {
	imageRef, err := dag.Go().KoBuild(
		ctx,
		src,
		dagger.GoKoBuildOpts{
			Repo:      repo,
			BuildArg:  buildArg,
			KoVersion: koVersion,
			Push:      push,
			TokenName: tokenName,
			Token:     token,
		})

	if err != nil {
		return "", err
	}

	// Optionally scan the image after build
	if scan && push == "true" {
		scanReport, scanErr := m.ScanImage(ctx, imageRef, nil, nil, scanSeverity, "0.64.1")
		if scanErr != nil {
			return imageRef, scanErr
		}
		// Materialize the scan by getting its contents (forces execution)
		_, contentErr := scanReport.Contents(ctx)
		if contentErr != nil {
			return imageRef, contentErr
		}
	}

	return imageRef, nil
}

func (m *Dagger) ScanImage(
	ctx context.Context,
	imageRef string, // Fully qualified image reference (e.g., "ttl.sh/my-repo:1.0.0")
	// +optional
	registryUser *dagger.Secret,
	// +optional
	registryPassword *dagger.Secret,
	// +optional
	// +default="HIGH,CRITICAL"
	severity string,
	// +optional
	// +default="0.64.1"
	trivyVersion string,
) (*dagger.File, error) {

	return dag.Trivy().ScanImage(
		imageRef,
		dagger.TrivyScanImageOpts{
			RegistryUser:     registryUser,
			RegistryPassword: registryPassword,
			Severity:         severity,
			TrivyVersion:     trivyVersion,
		}), nil
}
