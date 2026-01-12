package main

import (
	"context"
	"dagger/dagger/internal/dagger"
)

func (m *Dagger) Build(
	ctx context.Context,
	src *dagger.Directory,
	// +optional
	// +default="1.25.5"
	goVersion string,
	// +optional
	// +default="linux"
	os string,
	// +optional
	// +default="amd64"
	arch string,
	// +optional
	// +default="main.go"
	goMainFile string,
	// +optional
	// +default="claim-machinery-api"
	binName string,
	// +optional
	// +default="bookworm"
	variant string,
	// +optional
	ldflags string,
	// +optional
	// +default=""
	packageName string,
) *dagger.Directory {

	binDir := dag.Go().BuildBinary(
		src,
		dagger.GoBuildBinaryOpts{
			GoVersion:   goVersion,
			Os:          os,
			Arch:        arch,
			GoMainFile:  goMainFile,
			BinName:     binName,
			Ldflags:     ldflags,
			PackageName: packageName,
			Variant:     variant,
		})
	return binDir
}
