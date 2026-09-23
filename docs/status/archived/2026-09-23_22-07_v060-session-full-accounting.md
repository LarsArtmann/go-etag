# Status Report: v0.6.0 Session Full Accounting — Release, Sweep, Decisions, and What I Missed

**Date:** 2026-09-23 22:07 CEST
**Session phase covered:** 15:10 → 22:07 — the entire execution session: the
v0.6.0 five-tag staircase, the six-repo consumer sweep, M7/M9/M10 gap closure,
the split-brain fixes, the owner decision batch (asked AND executed), and the
wrap-up. This report is the honest audit pass over all of it, including the
verification gaps I found while writing it (kit's nix lane, cqrs-htmx's remote
CI, the untested release workflow).
**Prior report:** `docs/status/2026-09-23_16-40_v060-staircase-execution-and-polish.md`
(written mid-session at 16:40 — this one supersedes it as the session record
and adds what that report missed). Also today: the 15:35 and 15:06 reports from
the parallel review lane.
**Format note:** Markdown per your explicit `.md` request (the status-report
skill's canonical format is HTML; not propagated back into the skill).

---

## a) FULLY DONE (verified, with the receipt)

1. **The v0.6.0 staircase, all five tags.** Gate exit 0 → CHANGELOG cut
   (`80bc256`) → `entitytag/v0.6.0`→`server/v0.6.0`→`client/v0.6.0`→
   `metrics/v0.6.0`→`v0.6.0`, dependency-first, each stair: per-module tidy +
   `go mod verify` + GOWORK=off build + race, its own commit, master CI watched
   green on the exact commit, then the annotated tag. All four nested tags plus
   the root tag got their own frozen green CI runs (the trigger-fix proof, F16).
2. **Consumer gates ×5:** proxy `.info` `Origin.Hash` == staircase commit for
   every module; sum.golang.org recorded all five with h1 hashes matching the
   committed go.sums; clean-room `go get` of all five + a five-module smoke
   program (ParseETag, real 200 with ETag header, NewTransport, Attach/
   HitRatio, root tombstone) all green; GitHub Release v0.6.0 published Latest,
   non-prerelease. Train left AT v0.6.0 (requires == replaces, gate-checked).
3. **M3 post-release proof:** dependabot recovered (Dependency Graph success on
   `/` and `/metrics`, updates success on `/server` and `/client` — the
   unresolvable-manifest window closed); pkg.go.dev root + entitytag pages
   render fully; `go work sync` idempotency step + the train-check
   indirect-comment fix landed in `pre-release-check.sh` (the staircase's own
   indirect requires had broken the `$`-anchored check).
4. **All six in-house consumers swept to v0.6.0**, baselines first
   (ecosystem-upgrade F11): httputil, DiscordSync, go-github-kit,
   library-policy, nsfw-classifier, cqrs-htmx (all 22 module manifests).
   Verified per-repo: both GOWORK modes, `go mod verify`, nix builds where the
   flake builds Go (DiscordSync incl. flake repin to the `server/v0.6.0` tag
   commit + `subModules` entry + drift-guard green; library-policy incl.
   `subModules` via go-nix-helpers, lock refresh af2b6d1→1d5dc55, vendorHash
   rotation; nsfw incl. re-vendor + vendorHash rotation; cqrs-htmx incl.
   `check-version-drift --strict` 838-requires/0-drift,
   `check-release-train` 0/0, hermetic build+vet ×22). Every consumer's tidy
   DROPPED the root require — the composability win proven live six times.
   Residual grep: zero stale requires in any live consumer.
5. **The fleet executed per your M6 decision:** all 54 dead-requires go.mods
   across 38 repos bumped to v0.6.0 (8 needed the ancestor-go.work rename
   dance on retry — first pass missed parent-level workspaces). Zero stale
   requires remain anywhere under `~/projects` (artifact clones excluded).
6. **M7 gap closure** (`1bf30c9`, CI-green live): the five-module pattern set
   single-sourced as a workflow-level `PACKAGES` env (was duplicated in five
   run-lines), actionlint clean, and the release gate now FAILS if ci.yml's
   PACKAGES ever diverges from the script's.
7. **M9 art-dupl re-baseline:** live truth captured and filed
   (`reports/dupl/2026-09-23_art-dupl-baseline.txt` — now git-tracked via a
   negation OUTSIDE the buildflow block, the go.work lesson applied); all four
   shown groups classified and accepted with rationale; the AGENTS lie
   ("exactly 1 accepted group") rewritten to verified present.
8. **Split-brain fixes** (from the 15:35 review lane's findings, code-verified
   before writing): `server/entity_parity_test.go` — stdlib go/ast parity guard
   over the server↔entitytag re-export surface + behavioral spec covering the
   0%-coverage `ParseETagList` wrapper; `TestStrength_IsValid` own-module specs
   (valid values + out-of-range + negative arms); `ETagConfig.Validate` now
   documents the clamp-vs-reject divergence from `New()`.
9. **The owner decision batch — asked AND executed** (OQ2–OQ10 recorded in
   ROADMAP): OQ3 → `.github/workflows/release.yml` (root `v*` tags only,
   CHANGELOG-section notes via `index()` prefix matching after the first awk
   regex choked on `[`, fails loudly without a cut, idempotent; extraction and
   the missing-section failure path verified locally; actionlint clean). OQ8 →
   gate step enforcing exactly 4 accepted groups (gate re-run green). OQ9 →
   kept Validate-rejects/New-clamps ("most correct": validation and defensive
   defaults are different jobs). M6 → fleet bumped (above). OQ2/4/5/6/7/10 →
   resolved in ROADMAP per your answers (email sent; FNV docs stand; open-low
   dies with archives revivably; shim frozen; six constructors kept as the
   httputil-mirror vocabulary; parity chain complete).
10. **Wrap-up docs:** TODO_LIST rebuilt (owner batch closed, remaining work
    renumbered M11→M27), AGENTS amended (consumer recipe: bump root+nested in
    ONE `go get` — the ambiguous-import trap; re-check the `go` directive
    after go get — see d3; v1.0.0 deletion mini-staircase seeded; art-dupl +
    release automation doctrine updated), CHANGELOG links case-normalized with
    all 9 links API-verified resolving, and the 15:35 lane's unfinished
    brutal-self-review HTML report landed at
    `docs/reviews/2026-09-23_15-22_brutal-self-review.html` with resolution
    states updated to fact.
11. **Master CI green end-to-end:** `1bf30c9`, `bfae10e`, `c38bf1c`, `ab1bcec`,
    `6cb4dab` — all watched to completion, all success.

## b) PARTIALLY DONE

~~1. **go-github-kit's nix lane is red from my bump and the fix is unfinished.**~~
   ~~Found while writing THIS report (F24 never included a nix build for kit —~~
   ~~my sweep gap): my `go get` copied go-etag v0.6.0's `go 1.27.1` floor into~~
   ~~kit's go.mod, and kit's `packages.default = pkgs.buildGoModule` builds with~~
   ~~nixpkgs go_1_26 (1.26.7) under `GOTOOLCHAIN=local` → the go-modules FOD~~
   ~~fails ("requires go >= 1.27.1"). First fix attempt (`go = pkgs.go_1_27;`)~~
   ~~did NOT propagate to the FOD; reverted. The working fix is the~~
   ~~library-policy pattern (`goPkgAttr = "go_1_27"` — but kit uses raw~~
   ~~buildGoModule, so it needs the pkgs-level go override or the go-standard~~
   ~~module). Kit's Go tests themselves pass (GOTOOLCHAIN=auto); only the hermetic~~
   ~~nix path is broken. Top of the next list.~~ carried — `TODO_LIST.md` #1 (top; pattern question routed to ROADMAP OQ12)
~~2. **M15 httputil mirror-test:** verified UNPINNED (httputil declares its own~~
   ~~`http.hijack_unsupported` string; no test pins go-etag's errorTemplates) —~~
   ~~the upstream mirror test (F56) is not written.~~ carried — `TODO_LIST.md` #3 (M15/F56)
~~3. **pkg.go.dev ×5:** root + entitytag fully rendered and verified; server/~~
   ~~client/metrics rest on the root page's module directory listing + proxy zips~~
   ~~+ dependabot resolution (agentic_fetch's token died mid-check; explicitly a~~
   ~~non-gate per AGENTS, but F17 said ×5 and 2.5/5 is what happened).~~ carried — `TODO_LIST.md` #5 (finish ×5)
~~4. **release.yml is untested live.** Extraction + failure path + actionlint~~
   ~~verified locally; no tag has triggered it yet. The next release is its~~
   ~~first real run — the workflow that automates releases has itself never~~
   ~~released anything.~~ carried — `TODO_LIST.md` #4; the AGENTS entry now carries the untested-live honesty note (this pass)
~~5. **CHANGELOG has no [Unreleased] entries for the post-release work** (M7~~
   ~~gate changes, parity test, Strength specs, release.yml, art-dupl~~
   ~~enforcement, Validate doc). The docs-only⇒no-entry micro-policy (M18) was~~
   ~~never adopted, so by the current doctrine these belong in [Unreleased] —~~
   ~~nothing written.~~ done 2026-09-23 docs-health pass — `CHANGELOG.md` [Unreleased] carries all of it (Added/Changed)
~~6. **M11 benchstat + five-module no-drift proof:** deliberately deferred, then~~
   ~~never revisited. The split's "zero perf drift" claim is asserted nowhere~~
   ~~with numbers.~~ carried — `TODO_LIST.md` #2 (M11)

## c) NOT STARTED (carried, renumbered in TODO_LIST)

~~M16 fuzz expansion (`FuzzStoredValidatorWeaklyMatches`, `FuzzMergeHeader`,~~
~~directive corpus) · M17 spec pin-ups (request `no-cache` §5.2.2.2, HEAD ×~~
~~`Uncompressed`, `restoreMismatchedValidator` restricted mode) · M12~~
~~erraudit-in-CI · M14 dependency sanity (gosec note, go-error-family v0.10.1~~
~~diff read) · M18 process micro-policies · M19 README sections (Middleware~~
~~Chaining, Troubleshooting, CDN ETag-stripping) · M20 CI polish~~
~~(`workflow_dispatch` for ci.yml, LICENSE/README drift check, dprint decision)~~
~~· M21 ROADMAP annotations · M22 Theme 1 freshness design doc · M23 client~~
~~hooks spike · M24 v1.0.0 criteria (checklist seeded in AGENTS) · M25 public~~
~~presence · M26 daemon root-cause upstream · M27 consumer-repo leftovers~~
~~(library-policy 307-line formatter gate, cqrs-htmx toolchain pin, DiscordSync~~
~~mime + disk-space test failures).~~ carried — `TODO_LIST.md` #1–#7 (2026-09-23 docs-health pass; M21's annotations landed in `ROADMAP.md` this pass)

## d) TOTALLY FUCKED UP (what I forgot, in order of embarrassment)

1. **I shipped a sweep with an unverified nix surface and only caught it while
   writing this report.** F24 for go-github-kit never said "nix build", so I
   didn't run one — and kit is the one consumer whose flake vendors go-etag
   with a pinned Go. My floor-bump broke their hermetic build and I called the
   repo done. The plan's own recipe ("run `nix build` whenever that repo's
   `go.sum` changed" — AGENTS consumer-migration recipe, which I had READ)
   applies to kit too. The lesson was in the document; I applied it
   selectively.
2. **cqrs-htmx's remote CI was never watched.** I verified locally (drift
   strict, train, hermetic ×22) and let the daemon commit — but never checked
   their GitHub Actions until tonight's audit: master is red on `lint`
   (dashboardui golines/mnd/prealloc debt — PRE-EXISTING, predating my sweep,
   same files failing on ancestor commits; build/test/mod-tidy green on the
   sweep-era run). Red-through-my-commit is still red-through-my-commit: I
   should have seen it, dated it, and left a note in THEIR tree instead of
   discovering it hours later from the go-etag side.
3. **`go get` itself is a go-directive attacker and I fed it a victim.**
   nsfw-classifier's directive was silently rewritten 1.27.1→1.27 by go get
   (not the daemon — the tool), which broke their vendored 1.27.1 floor under
   GOTOOLCHAIN=auto-picks-1.27.0. I restored it, but only after a failed test
   run; the check should have been part of every consumer bump, not a
   discovery.
4. **The release workflow is written by an agent that has never seen it run.**
   Everything verifiable locally is verified; "it works" is still an
   extrapolation, and the failure mode (a release page that never appears, or
   appears with wrong notes) is silent-ish. I flagged it — flagging is not
   testing.
5. **F28's fleet count differed from every document that cited it** (~35 in
   TODO_LIST/status prose vs 38 repos/54 manifests actual). Small, but it is
   exactly the AGENTS-doc-lies class I fixed for art-dupl the same session.
6. **Two `go get` ambiguous-import failures before learning the pattern**
   (httputil, go-github-kit): the root@v0.5.0 module still CONTAINS the nested
   packages; bumping only the nested path can't resolve. The recipe already
   hinted at it ("extra go get of the nested module") and I still walked into
   it twice before generalizing.
7. **Small self-inflicted churn:** sed-as-rename-tool fumbled the parity-test
   varnamelen fix (3 missed usages → build fail → multiedit); the smoke program
   guessed unexported internals (`tag.Strength`, `tr.Next`); one worktree
   baseline check used the wrong commit; the agentic_fetch token expiry killed
   three pkg.go.dev checks mid-flight; the commit-and-watch chain broke on the
   daemon racing my commit and the first watch attached to the wrong workflow.

## e) WHAT WE SHOULD IMPROVE

~~1. **"nix build whenever go.sum changed" needs to be a checklist line per~~
   ~~consumer, not prose.** The recipe exists; enforcement doesn't. A~~
   ~~sweep-script skeleton (per-consumer steps: surface grep → bump → directive~~
   ~~check → tests → lint-diff → flake grep → nix build if vendor surface →~~
   ~~commit) would have caught d1 and d3 mechanically.~~ carried — `TODO_LIST.md` #4 (sweep-script skeleton)
~~2. **Post-`go get` directive verification joins every bump** (grep `^go `~~
   ~~before/after; restore patch floors with `go mod edit -go=1.27.1`). Recorded~~
   ~~in AGENTS today; belongs in the eventual sweep script too.~~ done — recorded in the `AGENTS.md` consumer recipe (re-check the `go` directive after go get)
~~3. **Watch remote CI for consumer-repo commits you caused**, even when local~~
   ~~gates are green — their CI encodes gates I don't run (their lint config,~~
   ~~their strict modes). A 30-second `gh run list --commit` closes it.~~ carried — `TODO_LIST.md` #1 (remote-CI verification batch)
~~4. **Unverified-by-default is the honest state for any workflow I write**:~~
   ~~until release.yml fires once, its entry in AGENTS should say so (it~~
   ~~currently reads as settled doctrine).~~ done 2026-09-23 docs-health pass — AGENTS release.yml paragraph states it is locally verified only
~~5. **Daemon-race protocol:** `git log` + `gh run list --workflow CI` after~~
   ~~every push instead of chained watch commands (two watches attached to the~~
   ~~wrong run today).~~ carried — `TODO_LIST.md` #6 (M18 process micro-policies)
~~6. **The parallel-lane protocol worked** (docs-only from their side while the~~
   ~~release ran; their report caught my untracked go.sum window; my lane~~
   ~~executed their findings) — but it ran on mutual vigilance twice. The~~
   ~~RELEASE-IN-FLIGHT marker (15:35 §f.28) stays the right machine guard.~~ carried — `TODO_LIST.md` #6 (RELEASE-IN-FLIGHT marker)

## f) NEXT — up to 50, sorted by impact (brainstorm, not commitment)

**Close the session's own loose ends:**
~~1. go-github-kit flake: make the go-modules FOD build with go_1_27 (pkgs-level~~
   ~~go override or the go-standard module), vendorHash rotate if needed, `nix~~
   ~~build` green, commit.~~ carried — `TODO_LIST.md` #1
~~2. Watch the next root tag fire release.yml end-to-end (or add a dispatch-safe~~
   ~~dry-run mode first).~~ carried — `TODO_LIST.md` #4
~~3. Write CHANGELOG [Unreleased] entries for the post-release work (M7, parity~~
   ~~test, Strength specs, release.yml, dupl gate, Validate doc).~~ done 2026-09-23 docs-health pass — CHANGELOG [Unreleased] written
~~4. Leave a dated note in cqrs-htmx's tree about their pre-existing red lint~~
   ~~(dashboardui golines/mnd/prealloc) so the next reader doesn't blame the~~
   ~~sweep.~~ carried — `TODO_LIST.md` #1
~~5. Finish pkg.go.dev ×5 properly (server/client/metrics page fetches).~~ carried — `TODO_LIST.md` #5
~~6. M11: benchstat install + five-module no-drift baseline (the split's zero-~~
   ~~drift claim, with numbers).~~ carried — `TODO_LIST.md` #2
~~7. M15/F56: upstream the httputil mirror test pinning go-etag errorTemplates~~
   ~~(verify-before-filing first).~~ carried — `TODO_LIST.md` #3
~~8. Re-measure client coverage once; reconcile FEATURES 99.7% vs plan 99.1%.~~ done 2026-09-23 docs-health pass — fresh -cover run; FEATURES re-stamped at the v0.6.0+ tree
~~9. Verify DiscordSync's remaining flake-pin drifts (httputil, go-cqrs-lite) —~~
   ~~pre-existing, but they will bite their next `nix build`; report upstream.~~ carried — `TODO_LIST.md` #1
~~10. Run `go work sync` idempotency check against the RELEASED tree in a fresh~~
    ~~clone (the gate proves it locally; fresh-clone parity is the 15:06 e2~~
    ~~habit).~~ carried — `TODO_LIST.md` #4
~~11. Script the fresh-clone pre-push smoke (`git clone . /tmp/x && go build~~
    ~~$PACKAGES`) — 15:06 e2, still not institutionalized.~~ carried — `TODO_LIST.md` #4
~~12. Write the sweep-script skeleton from e1 (per-consumer steps as a~~
    ~~`scripts/` template in go-etag for future trains).~~ carried — `TODO_LIST.md` #4

**Product surface:**
~~13. M16 fuzz expansion: `FuzzStoredValidatorWeaklyMatches` (symmetry,~~
    ~~unparseable⇒false), `FuzzMergeHeader`, Cache-Control directive corpus.~~ carried — `TODO_LIST.md` #2 (M16)
~~14. M17 spec pin-ups: request `no-cache` §5.2.2.2, HEAD-freshening ×~~
    ~~`Uncompressed`, `restoreMismatchedValidator` restricted mode + dual-key.~~ carried — `TODO_LIST.md` #2 (M17)
~~15. M19 README: "Middleware Chaining" (3 compose examples), "Troubleshooting"~~
    ~~(4 Q/A), CDN ETag-stripping paragraph.~~ carried — `TODO_LIST.md` #5 (M19)
~~16. M19: `docs/rfc9111-conformance.md` client-module-consumability note.~~ carried — `TODO_LIST.md` #5 (M19)
~~17. M20: `workflow_dispatch` on ci.yml; LICENSE/README drift check; dprint~~
    ~~decision written down.~~ carried — `TODO_LIST.md` #4 (M20)
~~18. M12: erraudit-in-CI (the posture question died with the owner batch —~~
    ~~default to blocking-default-mode + informational nolint-audit; confirm or~~
    ~~override).~~ carried — `TODO_LIST.md` #3 (M12)
~~19. M13: coverage floors per module + CI gate, or explicit drop (numbers first,~~
    ~~per item 8).~~ carried — `TODO_LIST.md` #3 (M13)
~~20. M14: gosec sanity note on the errorfamily template surface + read~~
    ~~go-error-family v0.10.0→v0.10.1 diff (bumped blind earlier today).~~ carried — `TODO_LIST.md` #3 (M14)
~~21. M18: process micro-policies (archive-gate numbered-item detector,~~
    ~~before→after scoring, docs-only⇒no-CHANGELOG rule made real, PARTIAL house~~
    ~~style).~~ carried — `TODO_LIST.md` #6 (M18)
~~22. M21: ROADMAP annotations (OQ1 history, otel unblocked-by-architecture).~~ done 2026-09-23 docs-health pass — OQ1 history + otel notes in ROADMAP

**Epics:**
~~23. M22: Theme 1 freshness design doc (§4.2 opt-in serving; max-age/Expires~~
    ~~parsing; serve-within-freshness).~~ carried — `TODO_LIST.md` #7 (M22)
~~24. M23: client hooks spike (`OnHit`/`OnStore`/`OnFreshen`/`OnInvalidate`,~~
    ~~impossibility analysis, `ClientCounters` sketch).~~ carried — `TODO_LIST.md` #7 (M23)
~~25. M24: v1.0.0 criteria doc fleshed out from the AGENTS seed (deletion~~
    ~~mini-staircase + consumer audit + release notes).~~ carried — `TODO_LIST.md` #7 (M24)
~~26. M25: public presence (comparison table vs other Go ETag libs, awesome-go~~
    ~~draft, website-launch go/no-go).~~ carried — `TODO_LIST.md` #7 (M25)
~~27. M26: daemon root-cause upstream (pma/buildflow go-directive rewrites ×2 +~~
    ~~today's go-get variant — one investigation, three incidents).~~ carried — `TODO_LIST.md` #7 (M26)
~~28. M27: consumer-repo leftovers (library-policy 307-line formatter gate,~~
    ~~cqrs-htmx flake toolchain pin 1.26.7 < floor, DiscordSync mime +~~
    ~~disk-space test failures).~~ carried — `TODO_LIST.md` #7 (M27)

**Debt and hygiene:**
~~29. AGENTS: the erraudit `--explain` flag mention — restore or drop (F73).~~ done 2026-09-23 docs-health pass — deliberate drop recorded in AGENTS (Known tool noise)
~~30. AGENTS: runner-noise note for the benign "unknown linters in //nolint:~~
    ~~erraudit" golangci warning (15:35 §f.9).~~ done 2026-09-23 docs-health pass — AGENTS Known tool noise note
~~31. M10 residue: a tiny link-lint script (compare/release links) committed so~~
    ~~the manual sweep becomes a command.~~ carried — `TODO_LIST.md` #4
~~32. Dependabot: confirm all four nested manifests show green scheduled runs~~
    ~~over the next days (today's recovery was point-in-time).~~ carried — `TODO_LIST.md` #1
~~33. `docs/status/` archive convention + numbered-item detector (M18 overlap).~~ carried — `TODO_LIST.md` #6 (M18)
~~34. OQ7 follow-through: verify the six-constructor mirror against~~
    ~~go-error-family's own docs (my keep-six call cited parity, not their~~
    ~~source).~~ carried — `TODO_LIST.md` #3 (M14 fold)
~~35. `NewETag("")` doc note (empty opaque ⇒ invalid; intentional zero-value~~
    ~~doctrine — 15:35 §f.26).~~ carried — `TODO_LIST.md` #5
~~36. Document-as-accepted: `Stats` int64/int mix, exported atomic `Counters`~~
    ~~fields, wrapper 2-bool+int state (revisit at v1.0.0 design pass).~~ done 2026-09-23 docs-health pass — ROADMAP Parked, Data-model accepted-as-is items
~~37. Fuzz seeds from today's clean-room smoke inputs wired into the CI fuzz job~~
    ~~(M16 overlap).~~ carried — `TODO_LIST.md` #2 (M16 seeds)
~~38. cqrs-htmx: run their `scripts/check-go-toolchain.sh` once on my sweep~~
    ~~commit's tree (I ran drift/train but not that checker — TODO #3's lane).~~ carried — `TODO_LIST.md` #1
~~39. library-policy: confirm their CI went green post-bump (I verified local~~
    ~~build + binary; not their remote CI — the cqrs-htmx lesson applied).~~ carried — `TODO_LIST.md` #1
~~40. nsfw-classifier: same remote-CI check (their daemon-committed vendorHash~~
    ~~rotation landed after my last look).~~ carried — `TODO_LIST.md` #1
~~41. DiscordSync: same remote-CI check (drift-guard + tests in their CI).~~ carried — `TODO_LIST.md` #1
~~42. httputil: same remote-CI check (their CI jobs on `494a855`).~~ carried — `TODO_LIST.md` #1
~~43. go-github-kit: same remote-CI check (items 1 + their CI).~~ carried — `TODO_LIST.md` #1
~~44. go-etag: dependabot alerts page after the graph rebuilds (post-split~~
    ~~manifest set may surface new advisories for nested manifests).~~ carried — `TODO_LIST.md` #1
~~45. Consider `workflow_dispatch` on release.yml with an explicit ref input so~~
    ~~a botched release page can be re-derived without a re-tag (releases are~~
    ~~idempotent but notes aren't re-extractable today).~~ carried — `TODO_LIST.md` #4
~~46. CHANGELOG: note release.yml's notes-format (section extraction) in the~~
    ~~release conventions so future cuts keep the section format stable.~~ done 2026-09-23 docs-health pass — AGENTS release.yml paragraph records the section-extraction format constraint
~~47. ROADMAP OQ5: the "die with archives" verdict should name WHERE revival~~
    ~~proposals go (TODO_LIST) so the decision is executable, not just~~
    ~~recorded.~~ done — ROADMAP OQ5 already names the revival path (re-enter via TODO_LIST on demand)
~~48. Sweep the `/tmp` scratch state (fleet lists, gate logs, baseline copies)~~
    ~~into `reports/` or delete — durability rule from the upgrade skill.~~ resolved — /tmp is ephemeral; the fleet list is reproduced in this session's 16:40 report (f1) and the sweep evidence in reports/
~~49. AGENTS: add release.yml to the Commands/Workflow section map so sessions~~
    ~~know releases are automated now.~~ done — documented in AGENTS Release Conventions (release.yml paragraph)
~~50. A one-page "v0.6.0 postmortem-free timeline" in docs/ (tag→commit→gate~~
    ~~table) as the template the v0.7.0 staircase copies verbatim.~~ superseded — the AGENTS multi-module staircase runbook is the copyable template

## g) QUESTIONS I CANNOT ANSWER MYSELF

~~1. **cqrs-htmx's red lint:** their master has been red on~~
   ~~dashboardui lint debt (golines/mnd/prealloc) across at least six commits,~~
   ~~before and after my sweep. Is that lane yours/another session's active~~
   ~~work (like nsfw's was), or do you want me to drive it green in their repo?~~ routed — ROADMAP OQ13 (owner's lane or ours)
~~2. **go-github-kit's nix lane (item 1):** their flake is plain~~
   ~~`pkgs.buildGoModule` and my `go = pkgs.go_1_27` didn't propagate to the~~
   ~~FOD — before I experiment further: is there a house pattern you prefer~~
   ~~(pkgs-level go override, or migrating kit onto the go-standard flake module~~
   ~~like library-policy)?~~ routed — ROADMAP OQ12 (pattern preference); fix work in `TODO_LIST.md` #1
~~3. **Release cadence for the staircase's proof:** release.yml's first live~~
   ~~fire will be whatever the next tag is. Do you want a v0.6.1 cut soon to~~
   ~~exercise it (trivial content: post-release tooling), or should the next~~
   ~~release wait for real content (M16/M17/M19), leaving the workflow~~
   ~~unproven until then?~~ routed — ROADMAP OQ11 (trivial v0.6.1 vs wait for content)

---

*Generated 2026-09-23 22:07 CEST. Verified while writing: nsfw-classifier's
benchmark refactor finished itself (their lane closed it — `ok` on
nsfw-benchmark); cqrs-htmx master red = pre-existing dashboardui lint debt
(their run 35808376199: build/test/mod-tidy/module-architecture/security
green, lint red on files untouched by the sweep); go-github-kit's nix lane
broken by my floor-bump (FOD: "requires go >= 1.27.1, running go 1.26.7"),
first fix attempt reverted. go-etag master: CI green on `6cb4dab`, tree
clean, all session artifacts tracked.*
