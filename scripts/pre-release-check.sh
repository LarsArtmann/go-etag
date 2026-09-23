#!/usr/bin/env bash
# pre-release-check.sh — the one-command local release gate for go-etag.
#
# Encodes every locally checkable step of the release runbook (AGENTS.md,
# Release Conventions) so a release cannot ship with skipped steps again
# (v0.4.0 skipped `go mod verify` and the nolint audit). Run this on the
# exact commit you intend to tag; it must exit 0 before the CHANGELOG cut.
#
# Not covered here (stay manual): CI green on the exact commit, the
# annotated tag + push, proxy.golang.org / sum.golang.org verification,
# the clean-room `go get`, and the GitHub Release.

set -euo pipefail

# Every go / golangci-lint invocation needs the auto toolchain: local go is
# older than the go.mod directive and the persisted env pins GOTOOLCHAIN=local
# (AGENTS.md, Commands). Never change the persisted env instead.
export GOTOOLCHAIN=auto

step() { printf '\n==> %s\n' "$*"; }

fail() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}

step "build"
go build ./... || fail "go build"

step "vet"
go vet ./... || fail "go vet"

step "race tests"
go test -race -count=1 ./... || fail "go test -race"

step "lint (0 issues required)"
golangci-lint run || fail "golangci-lint run"

step "format clean"
fmt_diff="$(mktemp)"
if ! golangci-lint fmt --diff >"$fmt_diff" 2>&1; then
	golangci-lint fmt
	fail "golangci-lint fmt rewrote files - re-run the script after formatting"
fi

if [ -s "$fmt_diff" ]; then
	cat "$fmt_diff"
	fail "golangci-lint fmt --diff is not empty - run 'golangci-lint fmt' and commit"
fi

rm -f "$fmt_diff"

step "error audit (type-aware)"
erraudit_out="$(mktemp)"
if ! GOEXPERIMENT=jsonv2 erraudit ./... --type-aware --enforce-go-error-family \
	--enforce-samber-oops --enforce-generic-return >"$erraudit_out" 2>&1; then
	grep -v '\[feature:logger\]' "$erraudit_out" || true
	fail "erraudit"
fi

rm -f "$erraudit_out"

step "nolint audit (path, not ./...)"
GOEXPERIMENT=jsonv2 erraudit nolint-audit . || fail "erraudit nolint-audit"

step "module verification"
go mod verify || fail "go mod verify"

step "no replace directives"
if grep -qE '^[[:space:]]*replace[[:space:]]' go.mod; then
	fail "go.mod carries a replace directive - releases must build from the module proxy"
fi

step "no pseudo-version requires"
if grep -qE 'v[0-9]+\.[0-9]+\.[0-9]+-[0-9]{14}-[0-9a-f]{12}' go.mod; then
	fail "go.mod pins a pseudo-version - require a tagged release instead"
fi

step "toolchain pin parity (go.mod vs CI)"
go_directive="$(awk '$1 == "go" { print $2; exit }' go.mod)"
pin_count="$(grep -c "GOTOOLCHAIN: go${go_directive}" .github/workflows/ci.yml || true)"
if [ "$pin_count" -lt 3 ]; then
	found="$(grep -h 'GOTOOLCHAIN:' .github/workflows/ci.yml | sort -u | tr '\n' ' ')"
	fail "CI GOTOOLCHAIN pins do not match the go directive ${go_directive} (found: ${found}) - the daemon-relaxation class of drift"
fi

printf '\nALL LOCAL GATES GREEN.\n'
printf 'Remaining runbook steps (manual): CI green on this exact commit, annotated tag + push,\n'
printf 'proxy .info hash + sum.golang.org, clean-room go get, GitHub Release as Latest.\n'
