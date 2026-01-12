package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
	"strconv"
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
	// normalize port for container exposure
	expPort, errConv := strconv.Atoi(port)
	if errConv != nil || expPort <= 0 {
		expPort = 8080
	}
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
		WithExposedPort(expPort).
		WithEnvVariable("PORT", port).
		WithEnvVariable("ENABLE_TEST_ROUTES", "1").
		// Uncomment to see JSON logs from API during tests
		// WithEnvVariable("LOG_FORMAT", "json").
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
			echo "${BLUE}[1/3] Testing /health endpoint${NC}"
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
			echo "${BLUE}[2/3] Testing /api/v1/claim-templates endpoint${NC}"
			TEMPLATES_RESPONSE=$(curl -s http://api:%s/api/v1/claim-templates)
			TEMPLATE_COUNT=$(echo "$TEMPLATES_RESPONSE" | grep -o '"name"' | wc -l)
			echo "${GREEN}✓ Templates endpoint passed${NC}"
			echo "  Found $TEMPLATE_COUNT templates"
			echo ""

			# Version endpoint + Request-ID propagation
			echo "${BLUE}[3/4] Testing /version endpoint and X-Request-ID${NC}"
			REQ_ID="dagger-test-reqid-1"
			VERSION_RESPONSE=$(curl -s -H "X-Request-ID: $REQ_ID" http://api:%s/version)
			if echo "$VERSION_RESPONSE" | grep -q '"version"'; then
				echo "${GREEN}✓ Version endpoint responded${NC}"
				echo "  Response: $VERSION_RESPONSE"
			else
				echo "${RED}✗ Version endpoint failed${NC}"
				exit 1
			fi
			# Check header echo
			RESP_HEADERS=$(curl -s -D - -o /dev/null -H "X-Request-ID: $REQ_ID" http://api:%s/health)
			if echo "$RESP_HEADERS" | grep -i '^X-Request-ID:' | grep -q "$REQ_ID"; then
				echo "${GREEN}✓ X-Request-ID propagated in response headers${NC}"
			else
				echo "${RED}✗ X-Request-ID not found in response headers${NC}"
				echo "$RESP_HEADERS" | sed -n '1,20p'
				exit 1
			fi
			echo ""

			# Panic simulation to verify JSON error body with requestId
			echo "${BLUE}[4/4] Testing panic recovery JSON and requestId${NC}"
			REQ_ID2="dagger-test-reqid-2"
			# Capture headers and body without failing on 500
			HFILE=$(mktemp)
			BFILE=$(mktemp)
			curl -s -D "$HFILE" -o "$BFILE" -H "X-Request-ID: $REQ_ID2" "http://api:%s/__test/panic?msg=boom"
			STATUS=$(head -1 "$HFILE" | awk '{print $2}')
			if [ "$STATUS" != "500" ]; then
				echo "${RED}✗ Expected HTTP 500, got $STATUS${NC}"
				echo "--- Headers ---"
				sed -n '1,20p' "$HFILE"
				echo "--- Body ---"
				sed -n '1,40p' "$BFILE"
				exit 1
			fi
			if ! grep -qi '^X-Request-ID: ' "$HFILE"; then
				echo "${RED}✗ X-Request-ID header missing on panic response${NC}"
				sed -n '1,20p' "$HFILE"
				exit 1
			fi
			if ! grep -q "$REQ_ID2" "$BFILE"; then
				echo "${RED}✗ requestId not present in JSON error body${NC}"
				sed -n '1,40p' "$BFILE"
				exit 1
			fi
			echo "${GREEN}✓ Panic recovery returned JSON with requestId and 500${NC}"
			echo ""

			echo "${BLUE}========================================${NC}"
			echo "${GREEN}  All tests passed! 🎉${NC}"
			echo "${BLUE}========================================${NC}"
			`, port, port, port, port, port, port),
		}).
		Stdout(ctx)

	if err != nil {
		return "", fmt.Errorf("tests failed: %w", err)
	}

	return testResults, nil
}
