#!/usr/bin/env bash
# pre-release-check.sh — the one-command local release gate for go-etag.
#
# Encodes every locally checkable step of the release runbook (AGENTS.md,
# Release Conventions) so a release cannot ship with skipped steps again
# (v0.4.0 skipped `go mod verify` and the nolint audit). Run this on the
# exact commit you intend to tag; it must exit 0 before the CHANGELOG cut.
#
# Multi-module repo since the 2026-09-23 split: five modules (root tombstone,
# entitytag, server, client, metrics) joined by the committed go.work. Every
# gate loops over the module roots; the bans and parity checks cover every
# go.mod. Not covered here (stay manual): CI green on the exact commit, the
# annotated bottom-up tag staircase + push, proxy.golang.org / sum.golang.org
# verification per module tag, the clean-room `go get` per module, and the
# GitHub Release.

set -euo pipefail

# Every go / golangci-lint invocation needs the auto toolchain: local go is
# older than the go.mod directive and the persisted env pins GOTOOLCHAIN=local
# (AGENTS.md, Commands). Never change the persisted env instead.
export GOTOOLCHAIN=auto

MODULES=("." client entitytag metrics server)
PACKAGES="./... ./client/... ./entitytag/... ./metrics/... ./server/..."
GOMODS=(go.mod client/go.mod entitytag/go.mod metrics/go.mod server/go.mod)

step() { printf '\n==> %s\n' "$*"; }

fail() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}

step "build + vet + race tests (per module, workspace joined by go.work)"
for mod in "${MODULES[@]}"; do
	(
		cd "$mod"
		go build ./... || exit 1
		go vet ./... || exit 1
		go test -race -count=1 ./... || exit 1
	) || fail "build/vet/race in $mod"
done

step "lint (0 issues required, per module)"
golangci-lint run || fail "golangci-lint run (root)"
for mod in client entitytag metrics server; do
	(
		cd "$mod"
		golangci-lint run --config ../.golangci.yml
	) || fail "golangci-lint run in $mod"
done

step "format clean (per module)"
fmt_diff="$(mktemp)"
golangci-lint fmt --diff >"$fmt_diff" 2>&1 || {
	golangci-lint fmt
	fail "golangci-lint fmt rewrote files (root) - re-run the script after formatting"
}
[ -s "$fmt_diff" ] && {
	cat "$fmt_diff"
	fail "golangci-lint fmt --diff is not empty (root) - run 'golangci-lint fmt' and commit"
}
for mod in client entitytag metrics server; do
	(
		cd "$mod"
		golangci-lint fmt --diff --config ../.golangci.yml
	) >"$fmt_diff" 2>&1 || {
		(cd "$mod" && golangci-lint fmt --config ../.golangci.yml)
		fail "golangci-lint fmt rewrote files in $mod - re-run the script after formatting"
	}
	[ -s "$fmt_diff" ] && {
		cat "$fmt_diff"
		fail "golangci-lint fmt --diff is not empty in $mod - run 'golangci-lint fmt' and commit"
	}
done
rm -f "$fmt_diff"

step "error audit (type-aware, per module: erraudit is module-scoped)"
for mod in "${MODULES[@]}"; do
	erraudit_out="$(mktemp)"
	(
		cd "$mod"
		GOEXPERIMENT=jsonv2 erraudit ./... --type-aware --enforce-go-error-family \
			--enforce-samber-oops --enforce-generic-return
	) >"$erraudit_out" 2>&1 || {
		grep -v '\[feature:logger\]' "$erraudit_out" || true
		fail "erraudit in $mod"
	}
	rm -f "$erraudit_out"
done

step "nolint audit (nested modules only: the root module has no nolint directives, and a root-level scan misjudges nested-module directives as stale)"
for mod in client entitytag metrics server; do
	(
		cd "$mod"
		GOEXPERIMENT=jsonv2 erraudit nolint-audit .
	) || fail "erraudit nolint-audit in $mod"
done

step "module verification (per module)"
for mod in "${MODULES[@]}"; do
	(
		cd "$mod"
		go mod verify
	) || fail "go mod verify in $mod"
done

step "go.work tracked (the daemon's gitignore regeneration class)"
[ -n "$(git ls-files go.work)" ] ||
	fail "go.work is NOT tracked - the 2026-09-23 red-CI class: CI, lint, and fuzz resolve sibling modules through it. Check .gitignore's negation lines outside the buildflow markers."

step "no replace directives (any go.mod)"
for gomod in "${GOMODS[@]}"; do
	if grep -qE '^[[:space:]]*replace[[:space:]]' "$gomod"; then
		fail "$gomod carries a replace directive - releases must build from the module proxy (go.work carries the dev replacements instead)"
	fi
done

step "no pseudo-version requires (any go.mod)"
for gomod in "${GOMODS[@]}"; do
	if grep -qE 'v[0-9]+\.[0-9]+\.[0-9]+-[0-9]{14}-[0-9a-f]{12}' "$gomod"; then
		fail "$gomod pins a pseudo-version - require a tagged release instead"
	fi
done

step "toolchain pin parity (every go.mod vs CI, exact match)"
go_directives="$(for gomod in "${GOMODS[@]}"; do
	awk '$1 == "go" { print $2; exit }' "$gomod"
done | sort -u)"
[ "$(printf '%s\n' "$go_directives" | wc -l)" -eq 1 ] ||
	fail "go directives differ across go.mod files: $(echo "$go_directives" | tr '\n' ' ')"
go_directive="${go_directives//$'\n'/}"
pin_count="$(grep -cE "GOTOOLCHAIN: go${go_directive//./\\.}$" .github/workflows/ci.yml || true)"
if [ "$pin_count" -lt 3 ]; then
	found="$(grep -h 'GOTOOLCHAIN:' .github/workflows/ci.yml | sort -u | tr '\n' ' ')"
	fail "CI GOTOOLCHAIN pins do not exactly match the go directive ${go_directive} (found: ${found}) - the daemon-relaxation class of drift"
fi

step "internal version train in sync (go.mod requires == go.work replaces)"
train_versions="$(grep -hE '^[[:space:]]*github\.com/larsartmann/go-etag' "${GOMODS[@]}" |
	awk '{ print $1, $2 }' | sort -u)"
train_count="$(printf '%s\n' "$train_versions" | wc -l)"
[ "$train_count" -ge 1 ] || fail "no internal requires found"
while read -r modpath version; do
	dir="${modpath#github.com/larsartmann/go-etag}"
	dir="${dir#/}"
	[ -n "$dir" ] || dir="."
	grep -qE "^	${modpath//./\\.} ${version//./\\.} => \./${dir#./}" go.work ||
		fail "go.work replace for $modpath at $version is missing or drifted"
	grep -qE "^	\./${dir#./}$" go.work ||
		fail "go.work use entry for ./$dir is missing"
done <<<"$train_versions"
while read -r modpath version; do
	for gomod in "${GOMODS[@]}"; do
		if grep -qE "^[[:space:]]*${modpath//./\\.} v" "$gomod"; then
			grep -qE "^[[:space:]]*${modpath//./\\.} ${version//./\\.}$" "$gomod" ||
				fail "$gomod requires $modpath at a version other than $version - the train must move together"
		fi
	done
done <<<"$train_versions"

printf '\nALL LOCAL GATES GREEN.\n'
printf 'Remaining runbook steps (manual): CI green on this exact commit, the bottom-up tag\n'
printf 'staircase (entitytag -> server -> client -> metrics -> root) + push, proxy .info hash\n'
printf '+ sum.golang.org per module tag, clean-room go get per module, GitHub Release as Latest.\n'
