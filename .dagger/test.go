package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
)

func (m *Dagger) BuildAndTest(
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
	// +optional
	// +default="8080"
	port string,
) (string, error) {
	// Build the binary
	binDir := m.Build(
		ctx,
		src,
		goVersion,
		os,
		arch,
		goMainFile,
		binName,
		variant,
		ldflags,
		packageName,
	)

	// Create a container with the binary and start the API service
	// Include the source directory to provide required template files
	apiContainer := dag.Container().
		From("debian:bookworm-slim").
		WithDirectory("/app", src).
		WithDirectory("/app/bin", binDir).
		WithExposedPort(8080).
		WithEnvVariable("PORT", port).
		WithWorkdir("/app").
		WithEntrypoint([]string{"./bin/" + binName}).
		AsService()

	// Create test container that runs tests against the API
	testResults, err := dag.Container().
		From("curlimages/curl:latest").
		WithServiceBinding("api", apiContainer).
		WithExec([]string{
			"sh", "-c",
			fmt.Sprintf(`
			set -e

			# Colors
			BLUE='\033[0;34m'
			GREEN='\033[0;32m'
			YELLOW='\033[1;33m'
			NC='\033[0m'

			echo "${BLUE}========================================${NC}"
			echo "${BLUE}   Claim Machinery API Test Suite${NC}"
			echo "${BLUE}========================================${NC}"
			echo ""

			# Wait for API to be ready
			echo "${YELLOW}⏳ Waiting for API to be ready...${NC}"
			for i in {1..30}; do
				if curl -f http://api:%s/health > /dev/null 2>&1; then
					echo "${GREEN}✓ API is ready${NC}"
					break
				fi
				printf "."
				sleep 1
			done
			echo ""
			echo ""

			# Run health check
			echo "${BLUE}[1/2] Testing /health endpoint${NC}"
			HEALTH_RESPONSE=$(curl -s http://api:%s/health)
			if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
				echo "${GREEN}✓ Health check passed${NC}"
				echo "  Response: $HEALTH_RESPONSE"
			else
				echo "${RED}✗ Health check failed${NC}"
				exit 1
			fi
			echo ""

			# Run list templates endpoint
			echo "${BLUE}[2/2] Testing /api/v1/claim-templates endpoint${NC}"
			TEMPLATES_RESPONSE=$(curl -s http://api:%s/api/v1/claim-templates)
			TEMPLATE_COUNT=$(echo "$TEMPLATES_RESPONSE" | grep -o '"name"' | wc -l)
			echo "${GREEN}✓ Templates endpoint passed${NC}"
			echo "  Found $TEMPLATE_COUNT templates"
			echo ""

			echo "${BLUE}========================================${NC}"
			echo "${GREEN}  All tests passed! 🎉${NC}"
			echo "${BLUE}========================================${NC}"
			`, port, port, port),
		}).
		Stdout(ctx)

	if err != nil {
		return "", fmt.Errorf("tests failed: %w", err)
	}

	return testResults, nil
}
