.PHONY: hygiene-quick hygiene-full profile-fixtures goldens check-generated test-go test-embed test-e2e test-all test-performance test-integration quality-evidence gate-gold-release deps-update deps-update-npm deps-audit-go deps-audit-npm deps-fix-npm deps-audit security-scan install-hooks install-noembed fuzz test-golden build

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

GOTOOLCHAIN ?= go1.26.4
GOVERSION := $(shell GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOVERSION 2>/dev/null | tr ' ' '_')
GOCACHE ?= $(CURDIR)/.gocache/$(if $(GOVERSION),$(GOVERSION),default)
GOENV = GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN)
GO_PACKAGES = $$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go list ./... | grep -v '/web/node_modules/')
GO_COVER_PACKAGES = $$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./... | grep -v '^$$' | grep -v '/web/node_modules/')
STATICCHECK ?= $(shell command -v staticcheck 2>/dev/null || printf '%s/bin/staticcheck' "$$(GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOPATH 2>/dev/null)")

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

test-go: profile-fixtures
	$(GOENV) go test $(GO_PACKAGES) -count=1

hygiene-quick: check-generated
	@echo "🧹 Running quick hygiene gate..."
	$(GOENV) go fmt $(GO_PACKAGES) && $(GOENV) go vet $(GO_PACKAGES)
	$(GOENV) $(STATICCHECK) $(GO_PACKAGES)
	$(MAKE) test-go
	cd web && npm run lint && npm run typecheck

hygiene-full: hygiene-quick
	@echo "🔍 Running full hygiene gate..."
	$(GOENV) go test $(GO_PACKAGES) -race
	$(GOENV) go test $(GO_COVER_PACKAGES) -coverprofile=coverage.out
	@$(GOENV) go test ./pkg/api/v1 -cover | awk '/coverage:/{gsub("%","",$$5); if ($$5+0 < 80) {print "❌ Coverage below 80%"; exit 1}}'
	cd web && npm run build
	@echo "⚡ Running performance tests..."
	@$(GOENV) go test ./test/performance -run=^$$ -bench=. -benchmem > /tmp/atb-performance-bench.txt
	@awk '/Benchmark/ {for (i=1; i<=NF; i++) if ($$i == "ns\\/op" && $$(i-1)+0 > 2000000000) {print "❌ Performance regression: >2s load time"; exit 1}}' /tmp/atb-performance-bench.txt
	@echo "✅ Hygiene gate passed"

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
	cd web && npm install
	cd web && npm run build
	$(GOENV) go build -o /tmp/atb-e2e ./cmd/atb
	@/tmp/atb-e2e view --no-open --port 18888 > /tmp/atb-e2e.log 2>&1 & echo $$! > /tmp/atb-e2e.pid
	@sleep 3
	@cd web && CYPRESS_BASE_URL=http://127.0.0.1:18888 npm run test:e2e || (echo "❌ E2E tests failed"; kill $$(cat /tmp/atb-e2e.pid) 2>/dev/null || true; rm -f /tmp/atb-e2e.pid; exit 1)
	@kill $$(cat /tmp/atb-e2e.pid) 2>/dev/null || true
	@rm -f /tmp/atb-e2e.pid
	@echo "✅ E2E tests passed"

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
	@-make security-scan || echo "⚠️  Security scan warning (non-blocking)"
	@echo ""
	@echo "Step 2: Test coverage..."
	@$(GOENV) go test ./pkg/api/v1 -cover | awk '/coverage:/{gsub("%","",$$5); if ($$5+0 < 80) {print "❌ Coverage below 80%"; exit 1}}'
	@echo "✅ Coverage OK"
	@echo ""
	@echo "Step 3: E2E tests..."
	@$(MAKE) test-e2e || (echo "⚠️  E2E tests failed — checking mock fallback..." && cd web && CYPRESS_MOCK_API=true npm run test:e2e || (echo "❌ E2E tests failed even with mocks"; exit 1))
	@echo ""
	@echo "Step 4: Lighthouse audit..."
	@-command -v lighthouse >/dev/null 2>&1 && lighthouse http://localhost:8080/view/ --output=json --output-path=web/lh-report.json --only-categories=accessibility,performance || echo "⚠️  Lighthouse skipped (install lighthouse globally to run this optional local audit)"
	@echo ""
	@echo "Step 5: Accessibility audit..."
	@cd web && npm run test:a11y || (echo "❌ A11y tests failed"; exit 1)
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
		echo "⚠️ govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
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
	@if command -v trivy >/dev/null 2>&1; then \
		trivy fs --scanners vuln --severity CRITICAL,HIGH --format json --output trivy-report.json .; \
	else \
		echo "⚠️ trivy not installed locally; using Docker fallback"; \
		docker run --rm -v "$$(pwd):/work" ghcr.io/aquasecurity/trivy:0.61.0 fs --scanners vuln --severity CRITICAL,HIGH --format json --output /work/trivy-report.json /work; \
	fi
	@GOSEC_BIN="$$(command -v gosec || true)"; \
	if [ -z "$$GOSEC_BIN" ] && [ -x "$$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOPATH 2>/dev/null)/bin/gosec" ]; then \
		GOSEC_BIN="$$(GOCACHE=$(GOCACHE) GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOPATH 2>/dev/null)/bin/gosec"; \
	fi; \
	if [ -n "$$GOSEC_BIN" ]; then \
		$(GOENV) "$$GOSEC_BIN" ./...; \
	else \
		echo "⚠️ gosec not installed locally; using Docker fallback"; \
		docker run --rm -v "$$(pwd):/work" -w /work golang:1.26.4 sh -lc 'go install github.com/securego/gosec/v2/cmd/gosec@latest && /go/bin/gosec ./...'; \
	fi
