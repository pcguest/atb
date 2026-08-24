.PHONY: check-versions hygiene-quick hygiene-full profile-fixtures goldens check-generated test-go coverage-check test-embed test-e2e test-all test-performance test-integration quality-evidence gate-gold-release deps-update deps-update-npm deps-audit-go deps-audit-npm deps-fix-npm deps-audit security-scan install-hooks install-noembed fuzz test-golden build

build:
	@echo "🔗 Building embedded ATB CLI..."
	cd web && npm ci && npm run build
	$(GOENV) go build -o ./atb ./cmd/atb
	@echo "✅ Built ./atb with embedded viewer"

test-golden:
	@echo "🔒 Running cross-language canonical-hash golden vectors..."
	$(GOENV) go test ./internal/hash/... -run TestGoldenVectors -count=1
	.venv/bin/python -m pytest sdk/python/tests/test_canonical_hash.py -x -q
	cd sdk/typescript && npm test -- --run canonical_hash
	@echo "✅ Golden vectors verified across Go, Python, and TypeScript"

GOTOOLCHAIN ?= go1.26.7
GOVERSION := $(shell GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOVERSION 2>/dev/null | tr ' ' '_')
GOCACHE ?= $(CURDIR)/.gocache/$(if $(GOVERSION),$(GOVERSION),default)
GOENV = GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN)
GO_PACKAGES = $$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go list ./... | grep -v '/node_modules/')
GO_COVER_PACKAGES = $$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./... | grep -v '^$$' | grep -v '/node_modules/')
GOSEC_DIRS = $$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go list -f '{{.Dir}}' ./... | grep -v '/node_modules/')
STATICCHECK ?= $(shell command -v staticcheck 2>/dev/null || printf '%s/bin/staticcheck' "$$(GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOPATH 2>/dev/null)")
TRIVY_VERSION ?= 0.73.0
TRIVY_IMAGE ?= ghcr.io/aquasecurity/trivy:0.73.0@sha256:7cced7cae583819fc7806d4cbc0dbbc7cad18b99f7d3e235192e6da8c091045c
GOSEC_VERSION ?= v2.27.1

profile-fixtures:
	$(GOENV) go run ./scripts/generate_profile_fixtures.go

# goldens regenerates every example/demo bundle a fresh clone needs for the
# local demo path and the docs links under examples/. These .atb bundles are
# gitignored (generated artefacts, not source); run this once after `make build`
# to materialise them. Pass/fail semantics are asserted by each generator.
goldens:
	@test -x ./atb || { echo "❌ ./atb not found — run 'make build' (or 'go build -o ./atb ./cmd/atb') first"; exit 1; }
	@echo "📦 Regenerating example + demo bundles..."
	$(GOENV) go run ./scripts/generate_profile_fixtures.go
	ATB_BIN=$(CURDIR)/atb bash examples/bundles/generate.sh
	ATB_BIN=$(CURDIR)/atb bash examples/bundles/demo-workflow/generate.sh
	$(GOENV) go run ./examples/bundles/incident-capture/
	@echo "✅ Regenerated examples/bundles/{profiles,project-bootstrap,demo-workflow,incident-capture}"

# check-generated regenerates the schema-driven bindings and fails if any of
# them drift from the committed output. Comparing against a pre-regen snapshot
# (rather than `git diff`) keeps the check correct on a dirty working tree.
check-generated:
	@echo "🔁 Checking generated bindings match schemas/event.v1.json..."
	@cp internal/event/types_generated.go /tmp/atb-gen-go.bak
	@cp sdk/python/atb/event_types_generated.py /tmp/atb-gen-py.bak
	@cp sdk/typescript/src/eventTypes_generated.ts /tmp/atb-gen-ts.bak
	$(GOENV) go generate ./internal/event/...
	@ok=1; \
	cmp -s internal/event/types_generated.go /tmp/atb-gen-go.bak || { echo "❌ internal/event/types_generated.go is out of date"; ok=0; }; \
	cmp -s sdk/python/atb/event_types_generated.py /tmp/atb-gen-py.bak || { echo "❌ sdk/python/atb/event_types_generated.py is out of date"; ok=0; }; \
	cmp -s sdk/typescript/src/eventTypes_generated.ts /tmp/atb-gen-ts.bak || { echo "❌ sdk/typescript/src/eventTypes_generated.ts is out of date"; ok=0; }; \
	rm -f /tmp/atb-gen-go.bak /tmp/atb-gen-py.bak /tmp/atb-gen-ts.bak; \
	if [ $$ok -ne 1 ]; then echo "   Run 'go generate ./internal/event/...' and commit the result."; exit 1; fi
	@echo "✅ Generated bindings in sync with schema"

check-versions:
	@ATB_SKIP_TAG_CHECK=1 /bin/bash scripts/check-versions.sh

test-go: profile-fixtures
	$(GOENV) go test $(GO_PACKAGES) -count=1

hygiene-quick: check-generated check-versions
	@echo "🧹 Running quick hygiene gate..."
	@unformatted="$$(gofmt -l $$(git ls-files '*.go'))"; \
		test -z "$$unformatted" || { printf '❌ Go files require gofmt:\n%s\n' "$$unformatted"; exit 1; }
	$(GOENV) go vet $(GO_PACKAGES)
	$(GOENV) $(STATICCHECK) $(GO_PACKAGES)
	$(MAKE) test-go
	cd web && npm run lint && npm run typecheck

hygiene-full: hygiene-quick
	@echo "🔍 Running full hygiene gate..."
	$(GOENV) go test $(GO_PACKAGES) -race
	$(GOENV) go test $(GO_COVER_PACKAGES) -coverprofile=coverage.out
	@$(MAKE) coverage-check
	cd web && npm run build
	@echo "⚡ Running performance tests..."
	@$(GOENV) go test ./test/performance -run=^$$ -bench=. -benchmem > /tmp/atb-performance-bench.txt
	@awk '/Benchmark/ {for (i=1; i<=NF; i++) if ($$i == "ns\\/op" && $$(i-1)+0 > 2000000000) {print "❌ Performance regression: >2s load time"; exit 1}}' /tmp/atb-performance-bench.txt
	@echo "✅ Hygiene gate passed"

coverage-check:
	@test -f coverage.out || { echo "❌ coverage.out missing; run 'make hygiene-full'"; exit 1; }
	@set -eu; \
		total="$$( $(GOENV) go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$$3); print $$3}' )"; \
		test -n "$$total" || { echo "❌ could not determine total coverage"; exit 1; }; \
		awk -v total="$$total" 'BEGIN { if (total + 0 < 80) { printf "❌ Total Go coverage %.1f%% is below 80%%\n", total; exit 1 } }'; \
		printf "✅ Total Go coverage %s%%\n" "$$total"

test-embed: hygiene-full
	@echo "🔗 Testing embed flow..."
	cd web && npm run build
	$(GOENV) go build -o /tmp/atb-rc ./cmd/atb
	@/tmp/atb-rc view --no-open --port 18888 > /tmp/atb-test-embed.log 2>&1 & echo $$! > /tmp/atb-test.pid
	@sleep 3
	@curl -f -I http://localhost:18888/view/ | grep -qi "content-security-policy" || (echo "❌ CSP missing"; kill $$(cat /tmp/atb-test.pid) 2>/dev/null || true; exit 1)
	@curl -f -s -X POST http://localhost:18888/api/v1/privacy/reveal -H "Content-Type: application/json" -d '{"seq":1}' -w "\nStatus: %{http_code}\n" | grep -q "401" || (echo "❌ Auth bypass"; kill $$(cat /tmp/atb-test.pid) 2>/dev/null || true; exit 1)
	@kill $$(cat /tmp/atb-test.pid) 2>/dev/null || true
	@rm -f /tmp/atb-test.pid
	@echo "✅ Embed + security smoke test passed"

test-e2e:
	@echo "🧪 Running E2E tests..."
	cd web && npm ci
	cd web && npm run build
	$(GOENV) go build -o /tmp/atb-e2e ./cmd/atb
	@test -f examples/quickstart/run.atb/bundle.atb || { echo "📦 Generating quickstart bundle (gitignored, absent on fresh checkouts)..."; ATB_BIN=/tmp/atb-e2e bash examples/quickstart/run.sh > /dev/null; }
	@/tmp/atb-e2e view --bundle examples/quickstart/run.atb/bundle.atb --session-token 0000000000000000000000000000000000000000000000000000000000000001 --no-open --port 18888 > /tmp/atb-e2e.log 2>&1 & echo $$! > /tmp/atb-e2e.pid
	@sleep 3
	@# ELECTRON_RUN_AS_NODE breaks the Cypress/Electron launcher when set by IDEs.
	@cd web && env -u ELECTRON_RUN_AS_NODE CYPRESS_BASE_URL=http://127.0.0.1:18888 npx cypress run --spec cypress/e2e/live-dashboard.cy.ts --browser firefox --env MOCK_API=false,SESSION_TOKEN=0000000000000000000000000000000000000000000000000000000000000001 || (echo "❌ Live E2E tests failed"; kill $$(cat /tmp/atb-e2e.pid) 2>/dev/null || true; rm -f /tmp/atb-e2e.pid; exit 1)
	@kill $$(cat /tmp/atb-e2e.pid) 2>/dev/null || true
	@rm -f /tmp/atb-e2e.pid
	@echo "✅ Live E2E tests passed"

test-performance:
	@echo "⚡ Running performance tests..."
	$(GOENV) go test ./test/performance ./cmd/atb -run=^$$ -bench=. -benchmem
	@echo "✅ Performance tests passed"

quality-evidence:
	@/bin/bash scripts/quality-evidence.sh

test-integration:
	@echo "🔁 Running integration tests..."
	$(GOENV) go test -tags=integration -count=1 -v ./test/integration/...

fuzz:
	go test ./internal/canonicalize/... -fuzz=FuzzMarshal -fuzztime=30s
	go test ./internal/bundle/... -fuzz=FuzzLoad -fuzztime=30s

test-all: hygiene-full
	@echo "✅ All tests passed"

## install-noembed: install CLI without embedded web UI (for go install compatibility)
install-noembed:
	$(GOENV) go install -tags noembed ./cmd/atb

gate-gold-release: test-all
	@echo "🏆 Running gold release gate..."
	@echo ""
	@echo "Step 1: Security scan..."
	@$(MAKE) security-scan
	@echo ""
	@echo "Step 2: Test coverage..."
	@$(MAKE) coverage-check
	@echo ""
	@echo "Step 3: E2E tests..."
	@$(MAKE) test-e2e
	@echo ""
	@echo "Step 4: Accessibility audit..."
	@cd web && env -u ELECTRON_RUN_AS_NODE npm run test:a11y || (echo "❌ A11y tests failed"; exit 1)
	@echo ""
	@echo "✅ All gold release gates passed"
	@echo "Ready to tag the current gold release"

deps-update:
	@echo "🔄 Updating Go dependencies..."
	$(GOENV) go get -u all
	$(GOENV) go mod tidy
	@echo "✅ Go dependencies updated"

deps-update-npm:
	@echo "🔄 Updating NPM dependencies..."
	cd web && npm update
	cd web && npm install
	@echo "✅ NPM dependencies updated"

deps-audit-go:
	@echo "🔍 Auditing Go dependencies..."
	@if command -v govulncheck >/dev/null; then \
		$(GOENV) govulncheck ./...; \
	else \
		echo "⚠️ govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@v1.5.0"; \
		$(GOENV) go list -m -u all | grep -v "github.com/pcguest/atb"; \
	fi

deps-audit-npm:
	@echo "🔍 Auditing NPM dependencies..."
	cd web && npm audit --audit-level=high

deps-fix-npm:
	@echo "🔧 Fixing NPM vulnerabilities..."
	@cd web && npm audit fix || (echo "⚠️ Some vulnerabilities require manual review (likely major upgrades)"; true)
	@echo "✅ NPM audit fix run complete"

deps-audit: deps-audit-go deps-audit-npm
	@echo "✅ Dependency audit complete"

install-hooks:
	@echo "🔧 Installing Git hooks..."
	@mkdir -p .githooks
	@chmod +x .githooks/pre-commit
	@git config core.hooksPath .githooks
	@echo "✅ Hooks installed at .githooks/"

security-scan:
	@echo "🔐 Running security scans..."
	@TRIVY_BIN="$$(command -v trivy || true)"; \
	TRIVY_FOUND_VERSION=""; \
	if [ -n "$$TRIVY_BIN" ]; then \
		TRIVY_FOUND_VERSION="$$($$TRIVY_BIN --version 2>/dev/null | awk 'NR == 1 { print $$2 }')"; \
	fi; \
	if [ "$$TRIVY_FOUND_VERSION" = "$(TRIVY_VERSION)" ]; then \
		"$$TRIVY_BIN" fs --skip-dirs .gocache --skip-dirs .gomodcache --skip-dirs .tmp --skip-dirs .venv --skip-dirs .venv-atb --skip-dirs web/node_modules --skip-dirs sdk/typescript/node_modules --scanners vuln --severity CRITICAL,HIGH --exit-code 1 --format json --output trivy-report.json .; \
	else \
		echo "⚠️ local Trivy version '$${TRIVY_FOUND_VERSION:-missing}' does not match $(TRIVY_VERSION); using pinned Docker image"; \
		docker run --rm -v "$$(pwd):/work" $(TRIVY_IMAGE) fs --skip-dirs .gocache --skip-dirs .gomodcache --skip-dirs .tmp --skip-dirs .venv --skip-dirs .venv-atb --skip-dirs web/node_modules --skip-dirs sdk/typescript/node_modules --scanners vuln --severity CRITICAL,HIGH --exit-code 1 --format json --output /work/trivy-report.json /work; \
	fi
	@GOSEC_BIN="$$(command -v gosec || true)"; \
	if [ -z "$$GOSEC_BIN" ] && [ -x "$$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOPATH 2>/dev/null)/bin/gosec" ]; then \
		GOSEC_BIN="$$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOPATH 2>/dev/null)/bin/gosec"; \
	fi; \
	GOSEC_FOUND_VERSION=""; \
	if [ -n "$$GOSEC_BIN" ]; then \
		GOSEC_FOUND_VERSION="$$(go version -m "$$GOSEC_BIN" 2>/dev/null | awk '$$1 == "mod" && $$2 == "github.com/securego/gosec/v2" { print $$3 }')"; \
	fi; \
	if [ "$$GOSEC_FOUND_VERSION" = "$(GOSEC_VERSION)" ]; then \
		$(GOENV) "$$GOSEC_BIN" $(GOSEC_DIRS); \
	else \
		echo "⚠️ local gosec version '$${GOSEC_FOUND_VERSION:-missing}' does not match $(GOSEC_VERSION); using pinned Docker install"; \
		docker run --rm -e GOFLAGS=-buildvcs=false -v "$$(pwd):/work" -w /work golang:1.26.7@sha256:45a5f7a810238aabcbad211d70b9ae082022d96f7c7259e94041ad1b933575ac sh -c 'go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) && /go/bin/gosec $$(go list -f "{{.Dir}}" ./... | grep -v "/node_modules/")'; \
	fi
	cd web && npm audit --audit-level=high
	cd sdk/typescript && npm audit --audit-level=high
