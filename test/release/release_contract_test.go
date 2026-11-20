package release_test

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func readRepositoryFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	// Windows checkouts may have CRLF endings; contract checks match on "\n".
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func makeTarget(t *testing.T, makefile, target, nextTarget string) string {
	t.Helper()
	startMarker := "\n" + target + ":"
	start := strings.Index(makefile, startMarker)
	if start < 0 {
		t.Fatalf("Makefile target %s not found", target)
	}
	end := strings.Index(makefile[start+len(startMarker):], "\n"+nextTarget+":")
	if end < 0 {
		t.Fatalf("Makefile target %s after %s not found", nextTarget, target)
	}
	return makefile[start : start+len(startMarker)+end]
}

// workflowJobSection returns the YAML block for one job in a workflow. The job
// ends at the first of nextJobs that follows it, or at end of file.
func workflowJobSection(t *testing.T, workflow, jobName string, nextJobs ...string) string {
	t.Helper()
	startMarker := "\n  " + jobName + ":"
	start := strings.Index(workflow, startMarker)
	if start < 0 {
		t.Fatalf("workflow job %s not found", jobName)
	}
	end := len(workflow)
	for _, next := range nextJobs {
		if offset := strings.Index(workflow[start+len(startMarker):], "\n  "+next+":"); offset >= 0 {
			if candidate := start + len(startMarker) + offset; candidate < end {
				end = candidate
			}
		}
	}
	return workflow[start:end]
}

func TestEventV1SchemaMatchesFrozenDigest(t *testing.T) {
	root := repositoryRoot(t)
	schema, err := os.ReadFile(filepath.Join(root, "schemas", "event.v1.json"))
	if err != nil {
		t.Fatalf("read frozen event schema: %v", err)
	}
	manifest := strings.TrimSpace(readRepositoryFile(t, "schemas/event.v1.sha256"))
	got := fmt.Sprintf("%x  event.v1.json", sha256.Sum256(schema))
	if manifest != got {
		t.Fatalf("event.v1 schema digest mismatch\nmanifest: %s\nactual:   %s", manifest, got)
	}
}

func TestGoldReleaseGateHasNoSoftFailurePaths(t *testing.T) {
	makefile := "\n" + readRepositoryFile(t, "Makefile")
	goldGate := makeTarget(t, makefile, "gate-gold-release", "deps-update")

	for _, forbidden := range []string{
		"@-make security-scan",
		"Security scan warning (non-blocking)",
		"CYPRESS_MOCK_API",
		"Lighthouse skipped",
	} {
		if strings.Contains(goldGate, forbidden) {
			t.Errorf("gold release gate contains soft-failure path %q", forbidden)
		}
	}
	for _, required := range []string{
		"$(MAKE) build",
		"$(MAKE) test-all",
		"$(MAKE) security-scan",
		"$(MAKE) test-e2e",
		"npm run test:a11y",
	} {
		if !strings.Contains(goldGate, required) {
			t.Errorf("gold release gate does not require %q", required)
		}
	}

	e2e := makeTarget(t, makefile, "test-e2e", "test-performance")
	if strings.Contains(e2e, "npm install") {
		t.Error("test-e2e uses npm install instead of reproducible npm ci")
	}
	if !strings.Contains(e2e, "npm ci") {
		t.Error("test-e2e does not install dependencies with npm ci")
	}

	coverage := makeTarget(t, makefile, "coverage-check", "test-embed")
	if !strings.Contains(coverage, "set -eu;") {
		t.Error("coverage-check does not fail fast when the threshold command fails")
	}
}

func TestSecretScanFetchesPullRequestCommitsWithoutWriteCredentials(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/security.yml")
	secretScanStart := strings.Index(workflow, "\n  secret-scan:")
	goSecurityStart := strings.Index(workflow, "\n  go-security:")
	if secretScanStart < 0 || goSecurityStart <= secretScanStart {
		t.Fatal("security workflow secret-scan job boundaries not found")
	}
	secretScan := workflow[secretScanStart:goSecurityStart]
	for _, required := range []string{
		"contents: read",
		"+refs/pull/*/head:refs/remotes/origin/pr/*",
		"persist-credentials: false",
		"bash scripts/check-fetched-public-refs.sh origin",
		"bash scripts/audit-public-history.sh",
	} {
		if !strings.Contains(secretScan, required) {
			t.Errorf("secret scan does not require %q", required)
		}
	}
	for _, forbidden := range []string{"pull-requests: write", "GITLEAKS_ENABLE_COMMENTS", "GITLEAKS_ENABLE_UPLOAD_ARTIFACT"} {
		if strings.Contains(secretScan, forbidden) {
			t.Errorf("secret scan unexpectedly enables %q", forbidden)
		}
	}
}

func TestSecurityGateValidatesReleaseImagePlatformsWithoutPublishing(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/security.yml")
	imageScanStart := strings.Index(workflow, "\n  trivy-image:")
	if imageScanStart < 0 {
		t.Fatal("security workflow image scan job not found")
	}
	imageScan := workflow[imageScanStart:]
	for _, required := range []string{
		"platform: linux/amd64",
		"platform: linux/arm64",
		"runner: ubuntu-latest",
		"runner: ubuntu-24.04-arm",
		`--platform "${PLATFORM}"`,
		"--load",
		`bash scripts/verify-container-architecture.sh "${IMAGE_REF}" "${PLATFORM}"`,
		`docker run --rm --platform "${PLATFORM}"`,
		"/licenses/atb/LICENSE",
		"/licenses/atb/THIRD_PARTY_NOTICES",
		"/app/web/out/view/index.html",
		"severity: HIGH,CRITICAL",
		"exit-code: '1'",
	} {
		if !strings.Contains(imageScan, required) {
			t.Errorf("security image scan does not enforce %q", required)
		}
	}
	for _, forbidden := range []string{"docker/setup-qemu-action@", "docker push", "push: true", "push-by-digest=true"} {
		if strings.Contains(imageScan, forbidden) {
			t.Errorf("PR-safe security image scan contains publishing path %q", forbidden)
		}
	}
}

func TestDockerBuildPropagatesAndVerifiesTargetArchitecture(t *testing.T) {
	dockerfile := readRepositoryFile(t, "Dockerfile")
	for _, required := range []string{
		"FROM --platform=$BUILDPLATFORM node:",
		"FROM --platform=$BUILDPLATFORM golang:",
		"ARG TARGETOS\n",
		"ARG TARGETARCH\n",
		"GOOS=${TARGETOS} GOARCH=${TARGETARCH}",
	} {
		if !strings.Contains(dockerfile, required) {
			t.Errorf("Dockerfile does not enforce target architecture contract %q", required)
		}
	}
	for _, forbidden := range []string{"ARG TARGETOS=", "ARG TARGETARCH=", "GOOS=linux", "GOARCH=amd64"} {
		if strings.Contains(dockerfile, forbidden) {
			t.Errorf("Dockerfile hardcodes target build input %q", forbidden)
		}
	}

	verifier := readRepositoryFile(t, "scripts/verify-container-architecture.sh")
	for _, required := range []string{
		"{{.Os}}/{{.Architecture}}",
		`docker cp "$CONTAINER_ID:/app/atb"`,
		"header[:4] != b\"\\x7fELF\"",
		"62: \"amd64/x86-64\"",
		"183: \"arm64/AArch64\"",
		"Runtime success alone is insufficient",
	} {
		if !strings.Contains(verifier, required) {
			t.Errorf("container architecture verifier does not enforce %q", required)
		}
	}

	dockerPublish := readRepositoryFile(t, ".github/workflows/docker-publish.yml")
	buildJob := workflowJobSection(t, dockerPublish, "build", "publish")
	for _, required := range []string{"docker buildx build", "--load"} {
		if !strings.Contains(buildJob, required) {
			t.Errorf("unprivileged Docker build job does not enforce %q", required)
		}
	}
	if !strings.Contains(buildJob, `LOCAL_TAG="atb-build:`) {
		t.Error("Docker build job does not tag its local image with atb-build:")
	}
	if !strings.Contains(buildJob, `bash scripts/verify-container-architecture.sh "${LOCAL_TAG}" "${PLATFORM}"`) {
		t.Error("Docker build job does not verify the embedded executable architecture of the local atb-build image")
	}
	// The build job validates an image locally. It must not hold registry
	// credentials or take any publishing path.
	for _, forbidden := range []string{
		"docker/login-action",
		"secrets.DOCKERHUB",
		"--push",
		"push-by-digest=true",
	} {
		if strings.Contains(buildJob, forbidden) {
			t.Errorf("unprivileged Docker build job contains publishing path %q", forbidden)
		}
	}

	publishJob := workflowJobSection(t, dockerPublish, "publish")
	for _, required := range []string{"docker buildx build", "--push", "imagetools inspect"} {
		if !strings.Contains(publishJob, required) {
			t.Errorf("protected Docker publication job does not enforce %q", required)
		}
	}
	revalidated := strings.Index(publishJob, "Revalidate tag and SHA before publication")
	loggedIn := strings.Index(publishJob, "Log in to Docker Hub")
	pushed := strings.Index(publishJob, "--push")
	if revalidated < 0 || loggedIn < 0 || pushed < 0 {
		t.Fatal("Docker publication job must revalidate, log in, and push")
	}
	if revalidated > loggedIn {
		t.Error("Docker publication job logs in before revalidating the tag and SHA")
	}
	if revalidated > pushed {
		t.Error("Docker publication job pushes before revalidating the tag and SHA")
	}

	releaseCheck := readRepositoryFile(t, "scripts/release-check.sh")
	for _, required := range []string{"docker buildx version", "for platform in linux/arm64 linux/amd64", "verify-container-architecture.sh"} {
		if !strings.Contains(releaseCheck, required) {
			t.Errorf("local release preflight does not enforce %q", required)
		}
	}

	releaseRunbook := readRepositoryFile(t, "docs/maintainers/release.md")
	for _, required := range []string{"Docker Buildx", "binfmt/emulation", "emulated runtime pass does not"} {
		if !strings.Contains(releaseRunbook, required) {
			t.Errorf("release runbook does not document Docker architecture prerequisite %q", required)
		}
	}
}

func TestFounderAcceptanceRunbookTracksFlagshipIncident(t *testing.T) {
	runbook := readRepositoryFile(t, "docs/maintainers/local-acceptance.md")
	quickstart := readRepositoryFile(t, "docs/getting-started/quickstart.md")
	for _, required := range []string{
		"command -v python3.9",
		"sys.version_info[:2] != (3, 9)",
		`"$PYTHON39" -m venv .venv`,
		"make gate-gold-release",
		"make demo-incident",
		"run.atb/incident-demo/incident.atb",
		"incident-content-tampered.atb",
		"incident-order-tampered.atb",
		"incident-record-removed.atb",
		"npm pack",
		"go install -tags noembed ./cmd/atb",
	} {
		if !strings.Contains(runbook, required) {
			t.Errorf("local acceptance runbook does not require %q", required)
		}
	}
	if strings.Contains(quickstart, "run.atb/agent-incident-demo.atb") {
		t.Error("quickstart still references the obsolete incident bundle path")
	}
	if !strings.Contains(quickstart, "run.atb/incident-demo/incident.atb") {
		t.Error("quickstart does not reference the flagship incident bundle path")
	}
}

func TestReleaseCheckPreparesPythonBeforeCrossLanguageGoTests(t *testing.T) {
	script := readRepositoryFile(t, "scripts/release-check.sh")
	goTests := strings.Index(script, `echo "[1/7] Go tests"`)
	pythonInstall := strings.Index(script, `"$ATB_PYTHON_BIN" -m pip install -r sdk/python/requirements-dev.txt`)
	if goTests < 0 || pythonInstall < 0 {
		t.Fatal("release check must install the pinned Python test environment before Go tests")
	}
	if pythonInstall > goTests {
		t.Fatal("release check runs cross-language Go tests before installing Python dependencies")
	}
	for _, required := range []string{
		`PYTHON_BIN="${ATB_RELEASE_PYTHON:-python3.11}"`,
		`sys.version_info < (3, 11)`,
		`ATB_PYTHON_BIN="$VENV_DIR/bin/python"`,
		`ATB_PYTHON="$ATB_PYTHON_BIN"`,
		`trap cleanup EXIT`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("release check does not configure %q", required)
		}
	}
}

func TestGoldReleaseWorkflowInstallsPythonSDKDependencies(t *testing.T) {
	release := readRepositoryFile(t, ".github/workflows/release.yml")
	gateStart := strings.Index(release, "\n  gate-gold:")
	publishStart := strings.Index(release, "\n  publish:")
	if gateStart < 0 || publishStart <= gateStart {
		t.Fatal("release workflow gold gate job boundaries not found")
	}
	releaseGate := release[gateStart:publishStart]

	if !strings.Contains(releaseGate, "actions/setup-python@") {
		t.Error("release gold gate does not set up Python for cross-language tests")
	}
	if !strings.Contains(releaseGate, "python -m pip install -r ./sdk/python/requirements-dev.txt") {
		t.Error("release gold gate does not install pinned Python SDK dependencies")
	}
	if !strings.Contains(releaseGate, "go install golang.org/x/vuln/cmd/govulncheck@v1.5.0") {
		t.Error("release gold gate does not install pinned govulncheck")
	}
}

func TestWebReleaseBuildUsesHermeticBundler(t *testing.T) {
	packageJSON := readRepositoryFile(t, "web/package.json")
	if !strings.Contains(packageJSON, `"build": "next build --webpack"`) {
		t.Fatal("web production build must pin webpack instead of allowing Next.js to select a port-binding Turbopack worker")
	}
}

func TestReleaseWorkflowPublishesCompleteCanonicalAssets(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/release.yml")
	required := []string{
		"artifacts/cli/atb-linux-amd64",
		"artifacts/cli/atb-windows-amd64.exe",
		"artifacts/cli/LICENSE",
		"artifacts/cli/THIRD_PARTY_NOTICES",
		"artifacts/python/*",
		"artifacts/npm/*",
		"artifacts/web/*",
		"artifacts/sbom/*",
		"artifacts/provenance/*",
		"draft: true",
		"gh release edit \"${GITHUB_REF_NAME}\" --draft=false",
	}
	for _, value := range required {
		if !strings.Contains(workflow, value) {
			t.Errorf("release workflow does not publish required canonical asset or transition %q", value)
		}
	}
}

func TestReleaseArtifactsCarryApplicableLegalFiles(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/release.yml")
	for _, required := range []string{
		"make check-notices",
		"cp LICENSE THIRD_PARTY_NOTICES dist/release/",
		"cp LICENSE THIRD_PARTY_NOTICES web/out/",
		"package/LICENSE",
		"package/THIRD_PARTY_NOTICES",
		"artifacts/cli/LICENSE",
		"artifacts/cli/THIRD_PARTY_NOTICES",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("release workflow does not enforce legal artifact contract %q", required)
		}
	}

	dockerfile := readRepositoryFile(t, "Dockerfile")
	if !strings.Contains(dockerfile, "COPY LICENSE THIRD_PARTY_NOTICES /licenses/atb/") {
		t.Error("runtime container does not carry ATB licence and third-party notices")
	}

	pythonProject := readRepositoryFile(t, "sdk/python/pyproject.toml")
	if !strings.Contains(pythonProject, `license-files = ["LICENSE", "THIRD_PARTY_NOTICES"]`) {
		t.Error("Python package does not include both legal files")
	}
	typescriptPackage := readRepositoryFile(t, "sdk/typescript/package.json")
	for _, name := range []string{`"LICENSE"`, `"THIRD_PARTY_NOTICES"`} {
		if !strings.Contains(typescriptPackage, name) {
			t.Errorf("TypeScript package does not include %s", name)
		}
	}
}

func TestReleaseSecurityToolsAreVersionPinned(t *testing.T) {
	for _, name := range []string{
		"Makefile",
		".github/workflows/ci.yml",
		".github/workflows/security.yml",
	} {
		if strings.Contains(readRepositoryFile(t, name), "@latest") {
			t.Errorf("%s installs an unpinned security tool with @latest", name)
		}
	}
}

func TestCIWorkflowLintAndDockerPublicationInputsArePinnedAndProtected(t *testing.T) {
	ci := readRepositoryFile(t, ".github/workflows/ci.yml")
	if !strings.Contains(ci, "go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12") {
		t.Fatal("CI does not run the pinned workflow linter")
	}

	dockerPublish := readRepositoryFile(t, ".github/workflows/docker-publish.yml")
	for _, required := range []string{
		"environment: docker-release",
		"--push",
		"imagetools inspect",
	} {
		if !strings.Contains(dockerPublish, required) {
			t.Errorf("Docker publication does not enforce %q", required)
		}
	}
	if strings.Contains(dockerPublish, "push-by-digest=true") {
		t.Error("Docker publication still depends on the removed digest-artifact flow")
	}

	publishJob := workflowJobSection(t, dockerPublish, "publish")
	if !strings.Contains(publishJob, "EXPECTED_SHA") && !strings.Contains(publishJob, "inputs.expected_sha") {
		t.Error("Docker publication job does not revalidate expected_sha before pushing")
	}
}

func TestDockerPublicationIsManualAndVerifiesThePushedDigest(t *testing.T) {
	dockerPublish := readRepositoryFile(t, ".github/workflows/docker-publish.yml")

	// The trigger block must not create an automatic path from a tag to a
	// registry write.
	onStart := strings.Index(dockerPublish, "\non:\n")
	permissionsStart := strings.Index(dockerPublish, "\npermissions:\n")
	if onStart < 0 || permissionsStart <= onStart {
		t.Fatal("docker workflow trigger block boundaries not found")
	}
	triggerBlock := dockerPublish[onStart:permissionsStart]
	if !strings.Contains(triggerBlock, "workflow_dispatch:") {
		t.Error("docker workflow does not declare manual dispatch")
	}
	for _, forbidden := range []string{"\n  push:", "tags:"} {
		if strings.Contains(triggerBlock, forbidden) {
			t.Errorf("docker workflow is still reachable from a tag push (%q)", forbidden)
		}
	}

	publishJob := workflowJobSection(t, dockerPublish, "publish")
	pushed := strings.Index(publishJob, "--push")
	digestStart := strings.Index(publishJob, "Verify pushed digest per platform")
	if digestStart < 0 {
		t.Fatal("docker publication job does not re-verify the pushed digest")
	}
	if pushed < 0 || digestStart < pushed {
		t.Error("docker publication job verifies the digest before pushing")
	}
	for _, required := range []string{
		"docker buildx imagetools inspect",
		`--format '{{json .Manifest}}'`,
		`jq -r '.digest'`,
		`docker pull --platform "${PLATFORM}" "${IMAGE}@${DIGEST}"`,
		`bash scripts/verify-container-architecture.sh "${IMAGE}@${DIGEST}" "${PLATFORM}"`,
		"org.opencontainers.image.version",
		`PUBLISH_RUNNER_ARCH: ${{ runner.arch }}`,
		`X64) RUNNER_PLATFORM=linux/amd64`,
		`ARM64) RUNNER_PLATFORM=linux/arm64`,
		`if [ "${PLATFORM}" != "${RUNNER_PLATFORM}" ]; then`,
		"Skipped atb version execution",
	} {
		if !strings.Contains(publishJob, required) {
			t.Errorf("pushed-digest verification does not enforce %q", required)
		}
	}

	// The publish job runs on ubuntu-latest without QEMU/binfmt, so the pushed
	// binary may only be executed for the runner's native platform. The arm64
	// image is still verified by pull, OCI label, and embedded ELF architecture,
	// none of which execute the binary.
	guard := strings.Index(publishJob, `if [ "${PLATFORM}" != "${RUNNER_PLATFORM}" ]; then`)
	binaryRun := strings.Index(publishJob, `"${IMAGE}@${DIGEST}" version`)
	if guard < 0 || binaryRun < 0 {
		t.Fatal("pushed-digest verification does not guard the binary version check by runner platform")
	}
	if guard > binaryRun {
		t.Fatalf("pushed-digest verification executes the pushed binary before the native-platform guard")
	}
	if !strings.Contains(publishJob[guard:binaryRun], "continue") {
		t.Error("pushed-digest verification does not skip binary execution for non-native platforms")
	}

	release := readRepositoryFile(t, ".github/workflows/release.yml")
	releasePublish := workflowJobSection(t, release, "publish")
	if !strings.Contains(releasePublish, "\n    environment: release\n") {
		t.Error("release publish job does not use the release environment")
	}
	revalidated := strings.Index(releasePublish, "Revalidate checked-out commit and tag before publication")
	if revalidated < 0 {
		t.Fatal("release publish job has no HEAD/tag SHA revalidation step")
	}
	for _, required := range []string{
		"git rev-parse HEAD",
		`refs/tags/${TAG_NAME}^{commit}`,
	} {
		if !strings.Contains(releasePublish, required) {
			t.Errorf("release publish revalidation does not check %q", required)
		}
	}
	if firstCredential := strings.Index(releasePublish, "secrets."); firstCredential >= 0 && revalidated > firstCredential {
		t.Error("release publish job uses a credential before revalidating HEAD and tag")
	}
}

func TestSecurityScanExcludesRepositoryBuildCaches(t *testing.T) {
	makefile := readRepositoryFile(t, "Makefile")
	start := strings.Index(makefile, "\nsecurity-scan:")
	if start < 0 {
		t.Fatal("Makefile security-scan target not found")
	}
	securityTarget := makefile[start:]
	for _, dir := range []string{".gocache", ".tmp", ".tmp-cleanroom"} {
		flag := "--skip-dirs " + dir
		if count := strings.Count(securityTarget, flag+" "); count != 2 {
			t.Errorf("security-scan must apply %q to native and Docker Trivy commands; found %d", flag, count)
		}
	}
	if strings.Contains(securityTarget, "gosec ./...") {
		t.Error("security-scan recursively scans build caches with gosec ./...")
	}
	if !strings.Contains(makefile, "GOSEC_DIRS =") ||
		!strings.Contains(makefile, `go list -f '{{.Dir}}' ./...`) {
		t.Error("Makefile does not derive filesystem directories for gosec")
	}
	if !strings.Contains(makefile, `grep -v '/node_modules/'`) {
		t.Error("Makefile does not exclude dependency directories from gosec")
	}
	if !strings.Contains(securityTarget, `"$$GOSEC_BIN" $(GOSEC_DIRS)`) {
		t.Error("native gosec scan does not use the module package directories")
	}
	if !strings.Contains(securityTarget, `/go/bin/gosec $$(go list -f "{{.Dir}}" ./...`) {
		t.Error("Docker gosec scan does not derive the module package directories")
	}
}

func TestSecurityScanRejectsMismatchedLocalScannerVersions(t *testing.T) {
	makefile := readRepositoryFile(t, "Makefile")
	for _, required := range []string{
		"TRIVY_VERSION ?= 0.73.0",
		`[ "$$TRIVY_FOUND_VERSION" = "$(TRIVY_VERSION)" ]`,
		"GOSEC_VERSION ?= v2.27.1",
		`go version -m "$$GOSEC_BIN"`,
		`[ "$$GOSEC_FOUND_VERSION" = "$(GOSEC_VERSION)" ]`,
		"STATICCHECK_VERSION ?= v0.7.0",
		"GOVULNCHECK_VERSION ?= v1.5.0",
		`go version -m "$$GOVULN_BIN"`,
		`[ "$$GOVULN_FOUND_VERSION" != "$(GOVULNCHECK_VERSION)" ]`,
		`GOVULNCHECK_BIN="$$GOVULN_BIN"`,
	} {
		if !strings.Contains(makefile, required) {
			t.Errorf("security-scan does not enforce pinned scanner input %q", required)
		}
	}
}

func TestScannerBootstrapPinsGoAnalyzersAndIsolatesCaches(t *testing.T) {
	script := readRepositoryFile(t, "scripts/bootstrap-scanners.sh")
	for _, required := range []string{
		`readonly STATICCHECK_VERSION="v0.7.0"`,
		`readonly GOVULNCHECK_VERSION="v1.5.0"`,
		`readonly GO_VERSION="go1.26.11"`,
		`GOMODCACHE="$DEST_DIR/../go-mod-cache"`,
		`GOSUMDB="sum.golang.org"`,
		`honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}`,
		`golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("scanner bootstrap does not enforce %q", required)
		}
	}
}

func TestLocalGoAnalysisDoesNotUseUserCaches(t *testing.T) {
	makefile := readRepositoryFile(t, "Makefile")
	for _, required := range []string{
		"GOMODCACHE ?= $(CURDIR)/.gomodcache",
		"STATICCHECK_CACHE ?= $(CURDIR)/.tmp/staticcheck-cache/",
		"GOENV = GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) STATICCHECK_CACHE=$(STATICCHECK_CACHE)",
	} {
		if !strings.Contains(makefile, required) {
			t.Errorf("local analysis environment does not enforce %q", required)
		}
	}
}

func TestPreCommitUsesCanonicalGovulnTarget(t *testing.T) {
	hook := readRepositoryFile(t, ".githooks/pre-commit")
	if !strings.Contains(hook, "make govuln-scan") {
		t.Fatal("pre-commit hook does not invoke the canonical govuln target")
	}
	for _, forbidden := range []string{
		"command -v govulncheck",
		"scripts/govulncheck.sh",
		"govulncheck not installed",
	} {
		if strings.Contains(hook, forbidden) {
			t.Errorf("pre-commit hook independently discovers or invokes govulncheck via %q", forbidden)
		}
	}
}

func TestGoldGateChecksPreparedVersionMarkers(t *testing.T) {
	makefile := readRepositoryFile(t, "Makefile")
	if !strings.Contains(makefile, "check-versions:\n") {
		t.Fatal("Makefile has no local version-consistency target")
	}
	if !strings.Contains(makefile, "hygiene-quick: check-generated check-versions") {
		t.Fatal("hygiene-quick does not enforce version consistency")
	}

	script := readRepositoryFile(t, "scripts/check-versions.sh")
	for _, marker := range []string{
		"sdk/typescript/src/index.ts",
		"README.md",
		"SECURITY.md",
		"CHANGELOG.md",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("version check does not cover %s", marker)
		}
	}
}

func TestGoModulePathMatchesMajorRelease(t *testing.T) {
	goMod := readRepositoryFile(t, "go.mod")
	if !strings.HasPrefix(goMod, "module github.com/pcguest/atb/v2\n") {
		t.Fatal("go.mod module path must carry the /v2 major-version suffix")
	}

	toolsMod := readRepositoryFile(t, "tools/go.mod")
	if !strings.HasPrefix(toolsMod, "module github.com/pcguest/atb/tools\n") {
		t.Fatal("tools module path must remain independent of the root major-version suffix")
	}

	readme := readRepositoryFile(t, "README.md")
	if !strings.Contains(readme, "go install github.com/pcguest/atb/v2/cmd/atb@latest") {
		t.Error("README does not document the v2 CLI install path")
	}
	if strings.Contains(readme, "go install github.com/pcguest/atb/cmd/atb@latest") {
		t.Error("README still documents the v1 CLI install path")
	}

	installer := readRepositoryFile(t, "scripts/verify-published-release.sh")
	if !strings.Contains(installer, `MODULE="github.com/pcguest/atb/v2/cmd/atb"`) {
		t.Error("published-release verification still targets the v1 module path")
	}
}

func TestInstallGuidanceUsesVersionedModulePath(t *testing.T) {
	root := repositoryRoot(t)
	oldPath := "github.com/pcguest/atb/cmd/atb"
	skipDirs := map[string]bool{
		".git": true, ".gocache": true, ".venv": true, "build": true,
		"dist": true, "node_modules": true, ".next": true, "out": true,
		"testdata": true, "vendor": true,
	}
	textExtensions := map[string]bool{
		".go": true, ".js": true, ".json": true, ".md": true, ".mjs": true,
		".py": true, ".sh": true, ".toml": true, ".ts": true, ".tsx": true,
		".txt": true, ".yaml": true, ".yml": true,
	}
	checked := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if skipDirs[entry.Name()] || rel == "docs/releases" {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "CHANGELOG.md" || rel == "test/release/release_contract_test.go" {
			return nil
		}
		if !textExtensions[filepath.Ext(path)] && rel != "sdk/typescript/bin/atb" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > 1<<20 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		checked++
		if strings.Contains(string(data), oldPath) {
			t.Errorf("%s still advertises the v1 CLI module path", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}
	if checked == 0 {
		t.Fatal("no install guidance files were checked")
	}
}

func TestReleaseNpmPublishingUsesOIDCWithoutToken(t *testing.T) {
	release := readRepositoryFile(t, ".github/workflows/release.yml")
	publishJob := workflowJobSection(t, release, "publish")

	for _, required := range []string{
		"id-token: write",
		"npm install -g npm@11.15.0",
		"NPM_CONFIG_PROVENANCE",
		"npm stage publish",
		"--access public",
		"--provenance",
		"Stage npm package (OIDC)",
	} {
		if !strings.Contains(publishJob, required) {
			t.Errorf("npm publication does not enable OIDC staged trusted-publishing requirement %q", required)
		}
	}

	// The workflow must stage a version for maintainer approval, not publish it
	// directly. A direct `npm publish` would bypass the stage-only trusted
	// publisher contract.
	if strings.Contains(publishJob, `npm publish "$package_tarball"`) {
		t.Error("npm publication still contains a direct npm publish invocation instead of staged publishing")
	}

	// The runbook must state the official staged trusted-publishing minimums and
	// the pinned npm CLI version, so documentation cannot drift from the workflow.
	releaseRunbook := readRepositoryFile(t, "docs/maintainers/release.md")
	for _, required := range []string{
		"11.15.0",
		"22.14.0",
		"npm stage publish",
		"--allow-stage-publish",
	} {
		if !strings.Contains(releaseRunbook, required) {
			t.Errorf("release runbook does not document npm staged trusted-publishing requirement %q", required)
		}
	}

	// The publish job must not resurrect the long-lived token path: no token
	// environment mapping, no npm token secret, and no registry-url/scope setup
	// that writes a token-auth .npmrc.
	for _, forbidden := range []string{
		"NODE_AUTH_TOKEN",
		"secrets.NPM_TOKEN",
		"NPM_TOKEN",
		`registry-url: "https://registry.npmjs.org"`,
	} {
		if strings.Contains(publishJob, forbidden) {
			t.Errorf("npm publication still depends on token-based setup %q", forbidden)
		}
	}

	for _, name := range []string{
		".github/WORKFLOWS.md",
		"docs/maintainers/release.md",
		"docs/getting-started/configuration.md",
	} {
		if strings.Contains(readRepositoryFile(t, name), "NPM_TOKEN") {
			t.Errorf("%s still documents an NPM token secret", name)
		}
	}
}
