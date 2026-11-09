# Review Fixes 250e545..HEAD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `subagent-driven-development` (recommended) or `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. fill the  `[ ]` with an `X` when it's done

---

## ⚠ REQUIRED — Model selection policy (binding, read before the first dispatch)

This plan is executed by a cold session that hands work to subagents. Every
dispatch **must name a model explicitly** in the `model` parameter — an omitted
model silently inherits the session model. When choosing models, load the
`selecting-models` skill, subject to these project-specific rules:

1. **Only `opencode-go/<model>` models.** Do not consider any other provider
   prefix, whatever the catalog or skill defaults suggest.
2. **Deliberately underestimate difficulty at first.** Assign tasks one
   difficulty level cheaper than the skill's table would say — even mid-tier
   (skill "T3 Integration") tasks start on `opencode-go/mimo-v2.6-flash`.
   Cheap-and-finished beats capable-and-never-needed.
3. **Calibrate from real usage after the first 2–3 subagents complete, then
   CONFIRM WITH THE USER before upgrading anything.**
   - Snapshot `opencode_console.get_usage({range: "24h"})` **before** the first dispatch.
   - Snapshot again after each of the first 2–3 subagents (implementer **and** its review) finishes.
   - `delta requests` = turns spent; `delta cost ÷ delta requests` = observed `$/turn` for that model.
   - Extrapolate the run: remaining spend ≈ Σ(task budget × observed $/turn), and check turns against the skill's per-difficulty budgets.
   - If the numbers or the output quality suggest a more capable model would be advisable: **stop and confirm with the user** before changing any assignment. The *first* tier upgrade is always a user decision; once the user approves a ladder, follow it automatically for the rest of the run.

### Baseline recorded at planning time (2026-11-08)

- 24 h usage: **78 requests, $0.1452** total; `opencode-go/mimo-v2.6-flash` alone: 77 requests, $0.1446 → **≈ $0.0019 observed per turn**.
- Budget: **$5 combined pool, $1.28 spent**, resets 2026-12-01 — keep dispatch prompts small and hand over file paths, not pasted contents. Re-snapshot before dispatching; these numbers age.

### Pre-chosen models for the first dispatch batch (Wave 1)

The first six dispatches — **both the implementer and the reviewer seat** — are pre-chosen as `opencode-go/mimo-v2.6-flash` (deliberate underestimate):

| # | Task | Implementer | Reviewer |
|---|---|---|---|
| 1 | T1 zip-slip guard | `opencode-go/mimo-v2.6-flash` | `opencode-go/mimo-v2.6-flash` |
| 2 | T2 pack-name sanitisation | `opencode-go/mimo-v2.6-flash` | `opencode-go/mimo-v2.6-flash` |
| 3 | T5 md5 unknown-state | `opencode-go/mimo-v2.6-flash` | `opencode-go/mimo-v2.6-flash` |
| 4 | T6 loadPackSync refactor | `opencode-go/mimo-v2.6-flash` | `opencode-go/mimo-v2.6-flash` |
| 5 | T7 atomic watcher | `opencode-go/mimo-v2.6-flash` | `opencode-go/mimo-v2.6-flash` |
| 6 | T8 `a.pkg` sweep | `opencode-go/mimo-v2.6-flash` | `opencode-go/mimo-v2.6-flash` |

Every remaining Wave 1 task (T9, T12, T13, T15, T22, T25–T31) also defaults to
`opencode-go/mimo-v2.6-flash` for both seats until calibration says otherwise.

### Per-task model guidance (defaults — underestimate first)

Difficulty labels below are the skill's tiers, renamed **D1–D4** to avoid
colliding with this plan's task ids T1–T32 (D1 transcription → D4 design/judgment).

| Difficulty | Plan tasks | Default model | Escalation trigger |
|---|---|---|---|
| D1–D2 (≤2 files, tests named, fully specified) | T1, T2, T3, T5, T9, T16, T18, T22, T25, T26, T27, T28, T29, T30, T31 | `opencode-go/mimo-v2.6-flash` | review finds real defects twice in a row |
| D2–D3 (multi-file, interfaces specified) | T4, T6, T7, T8, T11, T12, T14, T20, T24 | `opencode-go/mimo-v2.6-flash` (start here) | turns > 1.5× budget, or fix-loop round 3 |
| D3–D4 (integration, concurrency, security) | T10, T13, T17, T19, T21, T23 | `opencode-go/mimo-v2.6-flash` (start here — deliberate underestimate) | fix-loop round 3, or the user-approved ladder |
| Whole-branch review + escalated fixers | T32, any fix round ≥ 4 | **not pre-chosen** — propose to the user after calibration (`opencode-go/mimo-v2.6-pro` / `opencode-go/deepseek-v4-pro` are the next tiers) | always confirm first |

**Ladder** (use only after the user confirms the first upgrade):
`opencode-go/mimo-v2.6-flash` (0.14/0.28 per M) → `opencode-go/mimo-v2.6-pro` (0.435/0.87) → `opencode-go/deepseek-v4-pro` (0.66/1.98) → a most-capable model confirmed with the user (candidates: `opencode-go/glm-5.3`, `opencode-go/kimi-k3.1-pro`, `opencode-go/grok-4.7`, `opencode-go/qwen3.8-max`).

**Rules of thumb:**
- Reviewers scale with the **diff**, not the implementer: one-file diffs stay on flash; concurrency/security diffs and the whole-branch review go one tier up (only after calibration + user confirmation for the first upgrade).
- **Avoid:** `opencode-go/gemini-4-argon` as a default; free-tier models (`*-free`, `*-contributor`) on gated tasks — they burn turns and retries instead of dollars.
- **Context cliffs:** `opencode-go/claude-haiku-5-5` doubles in price above 100k tokens, `opencode-go/gpt-6-luna`/`gpt-5.6-luna` above 272k — keep dispatch prompts small, pass file paths.
- Record every dispatch in the ledger below as `Model: <id> — <reason>`.

### Dispatch ledger (fill in during execution)

| Task | Seat | Model | Requests (turns) | Cost USD | Notes |
|---|---|---|---|---|---|
| baseline snapshot | — | — | 78 | 0.1452 | 24 h usage at planning, 2026-11-08 |

---

**Goal:** Fix the security, correctness, race, performance, CI and documentation findings of the completed multi-dimension code review of git range `250e545..HEAD`, without regressing the 4 already-applied fixes or the behaviours the review marked as working-as-designed.

**Architecture:** Five phases, ordered by risk: security → races/locking → performance/behaviour → CI/tests → documentation. Every task is test-first where a test is practical; each phase ends with build, tests and `gofmt` green. One `git commit` per task; only the files a task names are staged.

**Tech Stack:** Go 1.27 (cgo via gcc/UCRT64), Fyne v2 (GUI), fsnotify, logrus, testify, GitHub Actions.

**Spec:** the review findings in the handoff from session `ses_e42a2826dffeP8rjRpmPfgJfhn` (every `file:line` below was re-verified against the current working tree; line numbers are current as of planning and shift as tasks apply — locate by function name).

## Global Constraints

- Build/test environment: `. .\setcc-gcc.ps1` first (gcc/UCRT64). Never set `CGO_CFLAGS` with gcc. `go test -race` works only with gcc and is slow → run **once**, in the final task.
- All text files LF, never CRLF. After every phase: `gofmt -l .` must print nothing, and `git diff --check` must be clean.
- Errors: new sentinels go in `pkg/ddpackage/error.go`; wrap with `errors.Join` or `dderrors.CausedBy(...)`.
- Logging: logrus with structured fields (`log.WithField`, `WithError`).
- UI mutations must run inside `fyne.Do(...)`. Binding `Set`/listeners are safe from any goroutine.
- New UI strings: `lang.X("key", "English fallback")` + `internal/gui/translation/en-US.json` only (Crowdin syncs the rest).
- Commits: stage named files only. **Never `git add -A` / `git add .`** — the tree holds untracked artifacts (`.opencode/`, `logs/`, `__debug_bin.exe`, …) and `AGENTS.md` must never be committed.
- Preserve (do not "fix"): `progress.go` throttle · watcher debounce + pause-queues-events + overflow rescan · `named/cache.go` generation counter · listener lifecycle in `gui.go` · in-place `resetTags`/`resetResourceMetadata` · `hujson.Standardize` everywhere · app clipboard for copy buttons · uitest DSL semantics · pixel-exact ddimage tests.
- The 4 already-applied uncommitted fixes (`pkg/ddpackage/pack.go` pack.json nil guard, `currentPackage()` conversions, resource_tree occurrences index, `waitForPendingClick`) are committed as-is in Task 0 and must not be reworked.

## Review Focus

1. **Hostile `.dungeondraft_pack` / `pack.json` inputs** — escaping res paths (`res://packs/evil/../../x`), pack names like `../../evil`, negative or multi-GB header integers must never write outside the chosen directory or panic/OOM. Owned by Tasks 1–3; their tests must fail before the fix.
2. **Non-English machines** — `newTestApp` does not pin locale `en`, so `TestAppSetNotAPackageContent` / `TestAppAddTagThroughTheTagPane` assert English strings against a localized app. Owned by Task 26.
3. **Watcher vs. package replacement during generation** — `pause`/`resume`/`discard` read `a.packageWatcher` while `teardownPackageWatcher` writes it; thumbnail generation mutates the file list concurrently with tests and with watcher updates. Owned by Tasks 7, 14; proven by the single `-race` run in Task 32.
4. **Roll-up selection semantics** — a resource shown under a selected tag branch is "visibly selected" but not "selected": deselect button, context-menu *Unselect* and fast second clicks all misbehave. Owned by Tasks 16–18; each has its own failing-first test.
5. **Async timing transitions** — filter debounce, async tag generation and in-place bulk-pane updates change when observable state settles. Owned by Tasks 19, 20, 24; their tests wait explicitly (never `time.Sleep` beyond existing helpers).

## Findings triaged out (verified during planning)

- `pkg/ddpackage/ddpackage.go:129-135` `SetAllow3rdParty` — **not reproducible**: it dereferences only its argument before `flLock.Lock()`, never `p.info`. No change.
- `pkg/ddpackage/pack.go:431-454` lock held across a full rescan — **accepted with documentation**: updates only run for unpacked packages, `Close()` is a no-op for them, and the AGENTS-documented invariant ("an update must not wait for the Fyne goroutine while it holds `packageUpdateLock`") must stay intact. Task 13 adds a comment stating the trade-off instead of shrinking the lock.
- Wall-clock helpers in `package_watcher_test.go` (`waitForWatcherIdle` 3-poll, 4×debounce sleep) — accepted; they sit inside `require.Eventually` with 10 s timeouts.
- Out of review scope (observed, not touched): `packPackage` does not pause the watcher while packing; GUI reads of `pkg.FileList().AsSlice()` in `RebuildTree` are unlocked (pre-existing pattern).

---

## Parallel execution schedule (subagent-driven)

Every task runs in its own git worktree on its own branch and merges back when
it is finished and verified. **Phases are not the unit of parallelism**: Phases
1–3 interleave the same hot files (`internal/gui/pack.go`, `package.go`,
`unpack.go`, `gui.go`, `resource_tree/resource_tree.go`), so three
phase-sized branches would collide at every merge. Tasks are the unit of
parallelism, subject to the chains below. Phases remain the *verification*
cadence (see below).

### Worktree protocol (every task)

1. Branch from current `main` (post-Task-0): `git worktree add ../DDGP-task-<NN>-<slug> -b fix/task-<NN>-<slug>`.
2. Implement and verify inside the worktree: `. .\setcc-gcc.ps1; go build ./...; go test ./<touched packages>/...; gofmt -l .` (must be empty).
3. **Before merge-back:** `git merge main` into the branch, resolve conflicts, re-run step 2.
4. Merge back: `git merge --no-ff fix/task-<NN>-<slug>`; the integrator runs `go build ./...` + the affected packages' tests on `main`, plus `gofmt -l .` and `git diff --check`.
5. `git worktree remove ../DDGP-task-<NN>-<slug>` and delete the branch.

Conflict classes:

- **chain** — same function, adjacent hunks, or same new test file: strictly serial, successor launches the moment its predecessor merges.
- **same file, distant hunks** — parallel allowed; git auto-merges; step 3 guarantees tests re-run on the merged result.
- **append-to-same-test-file** (e.g. two branches appending to `package_watcher_test.go`) — trivial conflict: keep both additions.

### Chains (serial — launch the successor as soon as its predecessor merges)

| Chain | Tasks | Why they cannot overlap |
|---|---|---|
| A | T1 → T3 | same file + same new test file (`pkg/ddpackage/unpack.go`, `unpack_test.go`) |
| B | T5 → T10 | both restructure `GetOrUpdateResourceMd5` and append to `ddpackage_test.go` |
| C | T2 → T4 → T14 | T2/T4 both rewrite `extractPackage`'s top; T4/T14 both edit `genThumbnails` |
| D | T6 → T11 → T20 | T6/T11 touch adjacent lines in `loadPack` (`unpack.go:59-84`); T11/T20 both edit `NewApp` |
| E | T7 → T21 → T24 | T21's new test needs T7's `a.watcher()`; T24 needs T21's exact-path `DiscardUnder` |
| F | T15 → T16 → T17 → T18 | same file regions + shared `resource_tree*_test.go`; T15's deletion at `:661` abuts T18's insertion |
| G | T9 → T21 | T9's pre-scan hunk (`pack.go:231`) is adjacent to T21's root-guard hunk (`:223`) — same function |

T19 additionally needs T8, T12 and T17 merged (it rewrites `buildBulkTagAddRow`, sits next to `saveTilesetMetadata`/`deselectResource`); T23 needs T8 and T11 (it rewrites `buildFilePreview`, where both sweep). Those are launch prerequisites, not chains — satisfied by the waves below plus protocol step 3.

### Launch waves

**Wave 1 — all launched in parallel immediately after Task 0** (18 worktrees):

> T1, T2, T5, T6, T7, T8, T9, T12, T13, T15, T22, T25, T26, T27, T28, T29, T30, T31

**Wave 2 — rolling launches as predecessors merge** (not a fixed batch):

| Task | Launches once |
|---|---|
| T3 | T1 merged |
| T10 | T5 merged |
| T4 | T2 merged |
| T11 | T6 merged |
| T14 | T4 merged |
| T16 | T15 merged |
| T20 | T11 merged |
| T21 | T7 **and** T9 merged |

**Wave 3 — after multi-file prerequisites merge:**

| Task | Launches once |
|---|---|
| T17 | T16 merged |
| T18 | T17 merged |
| T19 | T8, T12, T17 merged |
| T23 | T8, T11 merged |
| T24 | T21 merged |

**Free-floating (parallel with everything, merge any time):** T25, T29, T30, T31 — they touch only `build.yml`, `docs/design/*`, `en-US.json`, `CHANGELOG.md` (plus one distant doc-comment hunk in `resource_tree.go` from T29).

### Merge-review notes (soft dependencies)

- **gui.go struct-field block** (T6 `loadPacked` type, T7 watcher field, T11 `bgWork`, T20 filter fields) — edits land ~10–20 lines apart: parallel allowed, expect possible trivial field-insertion conflicts at merge-back.
- **package_watcher_test.go** — T7 edits helpers, T14 edits `thumbnailCount`, T21/T28 append tests, T24 edits one assertion: distant except the two appends, which merge by keeping both.
- **gui_test.go** — T4 adds a test, T26 pins a line: distant, auto-merges.
- **pkg/ddpackage/pack.go** — T9 and T21 are chained (G); every other pack.go task is distant from both.
- New test files are unique per task/chain (`package_save_test.go` for T12, `package_pane_test.go` for T19, `package_preview_test.go` for T23, `pack_md5_test.go` for T27) so parallel branches never append to the same new file.

### Concurrency budget

Wave 1's theoretical maximum is 18 concurrent worktrees. Practical
recommendation: **6–8 subagents at a time**, prioritising the chain heads that
gate Waves 2–3 (T1, T2, T5, T6, T7, T8) over the isolated doc/CI tasks, so the
pipeline refills itself as chains complete.

### Verification cadence

- **per task:** build + touched-package tests + `gofmt -l .` in the worktree, repeated after `git merge main`, and again on `main` after merge-back.
- **per wave boundary** (no open branch left touching a compiled package): full `go test ./...`.
- **once at Task 32:** `go test -race ./internal/gui/... ./pkg/ddpackage/...` (gcc). If any Phase 2 task was resolved through a conflict at merge-back, run the race suite once immediately after that merge as well.

---

## Phase 0 — Baseline

### Task 0: Commit the baseline (pre-applied fixes + this plan)

**Files:** stage exactly: `pkg/ddpackage/pack.go`, `internal/gui/pack.go`, `internal/gui/package.go`, `internal/gui/resource_tree/resource_tree.go`, `internal/gui/resource_tree/resource_tree_node_test.go`, `docs/plans/2026-11-08-review-fixes.md`

> The plan document must be committed here: git worktrees do not carry untracked
> files, so every task worktree reads the plan (and its Global Constraints) from
> the branch it starts from. `AGENTS.md` is local-only and will NOT exist in
> worktrees — subagent prompts must carry the Global Constraints section.

- [x] **Step 1: Verify the baseline is green**

Run: `. .\setcc-gcc.ps1; go build ./...; go test ./pkg/ddpackage/... ./internal/gui/resource_tree/...; gofmt -l .`
Expected: build OK, tests PASS, `gofmt -l .` prints nothing.

- [x] **Step 2: Commit the four verified fixes as-is**

```powershell
git add pkg/ddpackage/pack.go internal/gui/pack.go internal/gui/package.go internal/gui/resource_tree/resource_tree.go internal/gui/resource_tree/resource_tree_node_test.go docs/plans/2026-11-08-review-fixes.md
git commit -m "fix: guard nil pack.json entry, lock package access, index selection occurrences, fix click test race"
```

> This is the baseline every worktree branches from. No parallel work starts before it lands.

---

## Phase 1 — Security & correctness one-liners

### Task 1: C1 — zip-slip guard in extraction

**Files:**
- Modify: `pkg/ddpackage/error.go`, `pkg/ddpackage/unpack.go` (`extractFileList` ~:122, `ExtractFile` ~:182)
- Test: `pkg/ddpackage/unpack_test.go` (new)

**Interfaces:**
- Produces: `var ErrPathEscape = errors.New("resource path escapes the target directory")`; `func joinUnderBase(base, rel string) (string, error)` (returns `ErrPathEscape` for empty/absolute/volume-name `rel` or a result not under `base`).

- [x] **Step 1: Write the failing test**

```go
func TestJoinUnderBaseRejectsEscapingPaths(t *testing.T) {
	// rel "../../outside/x.png", "..", "", "C:\\evil", "/abs" -> ErrPathEscape
}
func TestJoinUnderBaseAcceptsNestedPaths(t *testing.T) {
	// rel "textures/objects" -> joined under base; "a/../b" cleans to "b" inside base
}
```

- [x] **Step 2: Run to verify it fails** — `go test ./pkg/ddpackage/ -run TestJoinUnderBase -v` → FAIL (undefined).

- [x] **Step 3: Implement in `pkg/ddpackage/unpack.go`**

Body of `joinUnderBase`: reject `rel == "" || filepath.IsAbs(rel) || filepath.VolumeName(rel) != ""`; `joined := filepath.Join(base, rel)`; `if ok, _ := utils.PathIsSub(base, joined); !ok → return "", ErrPathEscape`. Add `ErrPathEscape` to `error.go`.

- [x] **Step 4: Guard both sinks**

- `extractFileList`: replace `path := filepath.Join(outDirPath, filepath.Dir(fi.Path))` with `path, err := joinUnderBase(outDirPath, filepath.Dir(fi.Path))`; on error log with `logrus` fields (`resPath`, `path`) and return `dderrors.CausedBy(ErrPathEscape, err)` — extraction of a hostile pack must abort, not skip silently.
- `ExtractFile`: `filePath, err := joinUnderBase(outPath, fileNameFull)` before the `os.Create` check; wrap with `dderrors.CausedBy(ErrPathEscape, err)` and field `unpackedFile`.

- [x] **Step 5: Run tests** — `go test ./pkg/ddpackage/ -run 'TestJoinUnderBase|TestExtract' -v` → PASS.

- [x] **Step 6: Commit** — `git add pkg/ddpackage/error.go pkg/ddpackage/unpack.go pkg/ddpackage/unpack_test.go; git commit -m "fix: reject resource paths that escape the extraction directory"`

### Task 2: C2 — pack name used as extraction target

**Files:**
- Modify: `internal/gui/unpack.go` (`extractPackage` ~:254-275)
- Test: `internal/gui/unpack_test.go` (new)

**Interfaces:**
- Produces: `func unpackTargetPath(base, packName string) string`

- [x] **Step 1: Write the failing test** — table-driven `TestUnpackTargetPath`: names `"../../evil"`, `".."`, `""`, `"a/../../b"`, `"C:\\evil"`, `"My Pack"` → result is a single segment under `base` (`filepath.Rel(base, result)` has no `..` and is not absolute); normal names pass through unchanged.

- [x] **Step 2: Run to verify it fails** — `go test ./internal/gui/ -run TestUnpackTargetPath -v` → FAIL (undefined).

- [x] **Step 3: Implement `unpackTargetPath` in `internal/gui/unpack.go`**

`seg := filepath.Base(filepath.Clean(strings.TrimSpace(packName)))`; if `seg` is `"."`, `".."`, `""` or a separator → `seg = "pack"`; return `filepath.Join(base, seg)`. Note in the commit: names containing separators now become one segment (behaviour change, only for names that previously nested or escaped).

- [x] **Step 4: Use it in `extractPackage`**

Capture once at function top: `pkg := a.currentPackage()`; `if pkg == nil { log.WithError(...).Warn(...); return }`; `targetPath := unpackTargetPath(path, pkg.Name())`; pass `pkg` to `pkg.ExtractPackageProgress(...)` (replaces the two direct `a.pkg` reads at :267/:275).

- [x] **Step 5: Run tests** — `go test ./internal/gui/ -run TestUnpackTargetPath -v` → PASS; full `go test ./internal/gui/` still green.

- [x] **Step 6: Commit** — `git add internal/gui/unpack.go internal/gui/unpack_test.go; git commit -m "fix: sanitize pack name before using it as extraction directory"`

### Task 3: DoS bounds for packed headers

**Files:**
- Modify: `pkg/ddpackage/unpack.go` (`getFileList` ~:498-525, `readPackedFileFromPackage` ~:216-233)
- Test: `pkg/ddpackage/unpack_test.go`

**Interfaces:**
- Produces: `const maxResPathBytes int32 = 4096`; `func checkResPathLength(n int32) error`; `func checkPackedReadBounds(offset, size, fileSize int64) error` (both return errors, no panics).

- [x] **Step 1: Write the failing tests** — `TestCheckResPathLength`: `0`, `-1`, `4097` → error; `1`, `4096` → nil. `TestCheckPackedReadBounds`: negative size, size beyond `fileSize-offset`, negative offset → error; in-bounds → nil.

- [x] **Step 2: Run to verify they fail** — `go test ./pkg/ddpackage/ -run 'TestCheckResPathLength|TestCheckPackedReadBounds' -v` → FAIL.

- [x] **Step 3: Implement the checks**

- `getFileList`: after `binary.Read` of `filePathLength`, `if err := checkResPathLength(filePathLength); err != nil { log with FileNum/filePathLength; return dderrors.CausedBy(ErrInvalidPackage, err) }` **before** `make([]byte, filePathLength)`.
- `readPackedFileFromPackage`: before `make([]byte, info.Size)`, determine file size — `end, err := r.Seek(0, io.SeekEnd)` then seek back to `info.Offset` (keep the existing offset seek semantics); `checkPackedReadBounds(info.Offset, info.Size, end)` → on error `return nil, dderrors.CausedBy(ErrReadPacked, err)`.

- [x] **Step 4: Run tests** — `go test ./pkg/ddpackage/` → PASS.

- [x] **Step 5: Commit** — `git add pkg/ddpackage/unpack.go pkg/ddpackage/unpack_test.go; git commit -m "fix: bound packed header path lengths and read sizes"`

### Task 4: GUI dialog/lifecycle one-liners

**Files:**
- Modify: `internal/gui/pack.go` (`genThumbnails` ~:772, `packPackage` ~:834), `internal/gui/gui.go` (`setErrContent` ~:687), `internal/gui/about.go` (`withAlpha` ~:247)
- Test: `internal/gui/pack_test.go` (new), `internal/gui/gui_test.go`, `internal/gui/about_test.go`

- [x] **Step 1: Write three failing tests**

1. `TestGenThumbnailsWithoutPackageHidesProgressDialog` (in `pack_test.go`): `a := newTestApp(t)` (no package) → `a.genThumbnails()` → `require.Eventually(a.window.Canvas().Overlays().Top() == nil)` — today the dialog stays open forever (FAIL).
2. `TestSetErrContentReenablesButtons` (in `gui_test.go`): `a.disableButtons.Set(true)` → `a.setErrContent("msg", errors.New("x"))` → `disabled, _ := a.disableButtons.Get(); assert.False(disabled)` — today stays `true` (FAIL).
3. `TestWithAlphaKeepsRGBOfSemiTransparentColors` (in `about_test.go`): `withAlpha(color.NRGBA{200, 100, 50, 128}, 200)` must equal `color.NRGBA{200, 100, 50, 200}` — today premultiplied (FAIL).

- [x] **Step 2: Run to verify they fail** — `go test ./internal/gui/ -run 'TestGenThumbnailsWithoutPackage|TestSetErrContent|TestWithAlpha' -v` → FAIL.

- [x] **Step 3: Fix `genThumbnails` nil path** — in the `pkg == nil` branch add `fyne.Do(func() { progressDlg.Hide() })` beside `a.disableButtons.Set(false)`.

- [x] **Step 4: Fix `packPackage` goroutine** — capture `pkg := a.currentPackage()` at function top with a nil guard (mirrors Task 2); inside the goroutine wrap `progressDlg.Hide()`, the error dialog and the success dialog in `fyne.Do(...)` exactly like `genThumbnails` does at :801-817; use the captured `pkg` for `PackPackageProgress` and the message fields (replaces reads at :850/:865/:873/:889). Do the same `fyne.Do` wrapping for `progressDlg.Hide()` and both dialogs in `extractPackage`'s goroutine (`internal/gui/unpack.go` ~:274-300).

- [x] **Step 5: Fix `setErrContent` and `withAlpha`** — `setErrContent`: add `a.disableButtons.Set(false)` before `a.setMainContent(msgContent)` (every caller is a failed load). `withAlpha`: `n := color.NRGBAModel.Convert(c).(color.NRGBA); n.A = alpha; return n`.

- [x] **Step 6: Run tests** — `go test ./internal/gui/` → PASS.

- [x] **Step 7: Commit** — stage `internal/gui/pack.go internal/gui/unpack.go internal/gui/gui.go internal/gui/about.go internal/gui/pack_test.go internal/gui/gui_test.go internal/gui/about_test.go`; `git commit -m "fix: hide progress dialogs on every exit path, re-enable buttons after load errors, un-premultiply withAlpha"`

### Task 5: md5 unknown-state reports success with an empty hash

**Files:** Modify: `pkg/ddpackage/ddpackage.go` (`GetOrUpdateResourceMd5` ~:563-568); Test: `pkg/ddpackage/ddpackage_test.go` (new)

- [x] **Step 1: Write the failing test** — `TestGetOrUpdateResourceMd5UnknownPackageState`: `pkg := ddpackage.NewPackage(log)` (mode is `PackageModeUnloaded`), `fi := &structures.FileInfo{Path: "...", ResPath: "res://packs/x/y.png"}` with invalid md5 → call `GetOrUpdateResourceMd5` → callback gets `md5 == ""` and `err != nil`; `fi.Md5.IsValid()` stays false. Today: callback gets the md5 of the empty string and `err == nil` (FAIL).

- [x] **Step 2: Run to verify it fails** — `go test ./pkg/ddpackage/ -run TestGetOrUpdateResourceMd5UnknownPackageState -v` → FAIL.

- [x] **Step 3: Implement** — in the `else` branch (~:563): log with fields as today, then `callback("", dderrors.CausedBy(ErrPackageNotLoaded, errors.New("unknown package state")))` and `return` — never fall through to `hash.Sum`.

- [x] **Step 4: Run tests** — `go test ./pkg/ddpackage/` → PASS.

- [x] **Step 5: Commit** — `git add pkg/ddpackage/ddpackage.go pkg/ddpackage/ddpackage_test.go; git commit -m "fix: report an error for md5 hashing in an unknown package state"`

### Task 6: `loadPackSync` returns tag/metadata errors (dead `err = nil` removed)

**Files:**
- Modify: `internal/gui/unpack.go` (`loadPackSync` ~:89-119, `loadPack` ~:40-85), `internal/gui/gui.go` (`loadPacked` field ~:132)
- Test/Helpers: `internal/gui/z_spec_helpers_test.go` (`blockPackedLoad` ~:76-90)

**Interfaces:**
- Produces: `type packedLoadResult struct { Pkg *ddpackage.Package; TagsErr, MetadataErr error }`; `loadPacked func(path string, progress func(p float64, curRes string)) (packedLoadResult, error)` (mirrors `unpackedLoadResult`).

- [x] **Step 1: Change the signature** — `loadPackSync` returns `packedLoadResult`; delete the two `showErrorDialog` calls and both dead `err = nil` lines; return `TagsErr`/`MetadataErr` on the result.
- [x] **Step 2: Move the dialogs to `loadPack`** — after `loader(...)` succeeds: `if result.TagsErr != nil { a.showErrorDialog(dderrors.CausedBy(errors.New(lang.X("package.tags.error", "Failed to read tags")), result.TagsErr)) }` and the metadata equivalent, then `a.setPackContent(result.Pkg)` (same shape as `loadUnpackedPath` ~:165-175). Update the `loadPacked` field type and `blockPackedLoad`.
- [x] **Step 3: Run tests** — `. .\setcc-gcc.ps1; go build ./...; go test ./internal/gui/` → PASS (existing suite covers the load paths).
- [x] **Step 4: Commit** — `git add internal/gui/unpack.go internal/gui/gui.go internal/gui/z_spec_helpers_test.go; git commit -m "refactor: loadPackSync reports tag and metadata errors like loadUnpackedSync"`

**Phase 1 gate:** `. .\setcc-gcc.ps1; go build ./...; go test ./...; gofmt -l .; git diff --check` → all clean.

---

## Phase 2 — Races & locking

### Task 7: atomic `packageWatcher` field

**Files:** Modify: `internal/gui/gui.go` (field ~:110), `internal/gui/pack.go` (`setupPackageWatcher` ~:456, `pause/resume/discard/teardown` ~:486-524); Tests: `internal/gui/gui_test.go`, `internal/gui/package_watcher_test.go`

**Interfaces:**
- Field becomes `packageWatcher atomic.Pointer[packageWatcher]` (type and field share the name — legal Go).
- Produces: `func (a *App) watcher() *packageWatcher` (loads the pointer; returns nil when none).

- [x] **Step 1: Swap the field and accessors** — `setupPackageWatcher`: `a.packageWatcher.Store(watcher)`. `teardownPackageWatcher`: `if w := a.packageWatcher.Swap(nil); w != nil { w.Close() }`. `pause/resume/discardPackageWatcherChanges`: `if w := a.watcher(); w != nil { w.Pause()/Resume()/DiscardUnder(dir) }`. Each call may act on a replaced instance — that is safe: `Resume` on a never-paused watcher is a no-op, `Pause` on a closed watcher is inert (closed ⇒ timers stopped).
- [x] **Step 2: Update test references** — replace `a.packageWatcher` reads in `gui_test.go` (`assert.Nil`) and `package_watcher_test.go` (`require.NotNil`, `.DiscardUnder`, `.overflow`, `.Pending`) with `a.watcher()`.
- [x] **Step 3: Run tests** — `go test ./internal/gui/` → PASS.
- [x] **Step 4: Commit** — `git add internal/gui/gui.go internal/gui/pack.go internal/gui/gui_test.go internal/gui/package_watcher_test.go; git commit -m "fix: guard the package watcher with an atomic pointer"`

### Task 8: sweep the remaining direct `a.pkg` reads

**Files:** Modify: `internal/gui/package.go` (:174, :263, :377, :531, :569, :623, :1047), `internal/gui/pack.go` (:645, :665, :666), `internal/gui/tag_gen.go` (:127), `internal/gui/tag_sets.go` (:39, :50, :78, :118, :188, :196, :209, :210, :214), `internal/gui/unpack.go` (:196)

- [x] **Step 1: Convert every listed read** to `a.currentPackage()` (mechanical swap; `nil` behaves exactly like today's field read). For `buildFilePreview` and `buildPackageTree` capture `pkg := a.currentPackage()` once at the top and use the local. (Tasks 2 and 4 already covered `extractPackage` and `packPackage` — do not touch those again.)
- [x] **Step 2: Run tests** — `go build ./...; go test ./internal/gui/` → PASS.
- [x] **Step 3: Commit** — `git add internal/gui/package.go internal/gui/pack.go internal/gui/tag_gen.go internal/gui/tag_sets.go internal/gui/unpack.go; git commit -m "fix: read the package through currentPackage everywhere in the gui"`

### Task 9: pre-scan under `flLock` in `updateFromPaths`

**Files:** Modify: `pkg/ddpackage/pack.go` (`updateFromPaths` ~:223-247)

- [x] **Step 1: Move the `p.fileList.Find` block** (the `os.Stat` error branch at :231-242) under `p.flLock.RLock()/RUnlock()` — a read lock held only around the `Find`, not across the `os.Stat` loop; the write lock at :259 stays where it is. Do not hold `RLock` while calling `ListDir`.
- [x] **Step 2: Run tests** — `go test ./pkg/ddpackage/` → PASS (the `-race` run in Task 32 is the real proof).
- [x] **Step 3: Commit** — `git add pkg/ddpackage/pack.go; git commit -m "fix: guard the removal pre-scan of the file list with flLock"`

### Task 10: md5 single-flight, published under `flLock`

**Files:** Modify: `pkg/ddpackage/ddpackage.go` (`GetOrUpdateResourceMd5` ~:526-571, struct fields ~:60); Test: `pkg/ddpackage/ddpackage_test.go`

**Interfaces:**
- New fields on `Package`: `md5Mu sync.Mutex`, `md5Inflight map[string][]func(string, error)`, `pkgFileMu sync.Mutex`.
- Produces: `func (p *Package) computeMd5(fi *structures.FileInfo, key string)` (unexported worker).

**Lock order rule (must not be violated):** `flLock` may be held while acquiring `md5Mu`; never the reverse. `md5Mu` is never held while doing I/O or invoking callbacks.

- [x] **Step 1: Write the failing tests**
  1. `TestGetOrUpdateResourceMd5SingleFlight`: one unpacked file, 8 concurrent `GetOrUpdateResourceMd5` calls → all 8 callbacks fire with the identical correct hash; the file is read once (count via a `logrus` hook or by asserting callbacks fire in one batch — simplest: assert every callback receives `hex(md5(filebytes))` and no error; the single-flight structure is asserted by `TestGetOrUpdateResourceMd5QueuesCallbacksBehindAnInflightComputation` if needed).
  2. `TestGetOrUpdateResourceMd5PublishedUnderLock`: `go vet`-clean by construction; rely on `-race` (Task 32) for the lock proof — the test still asserts `fi.Md5.IsValid()` only after the callback runs.

- [x] **Step 2: Run to verify they fail (or race)** — `go test -race ./pkg/ddpackage/ -run TestGetOrUpdateResourceMd5 -v` → FAIL/race today.

- [x] **Step 3: Implement**
  - `GetOrUpdateResourceMd5`: under `md5Mu` — if `fi.Md5.IsValid()` copy `hash := fi.Md5.String()`, unlock, `callback(hash, nil)`; else if `key := fi.ResPath` has inflight entries, append the callback and return; else create the entry and start `go p.computeMd5(fi, key)`.
  - `computeMd5`: run the existing mode branches (unpack read / packed read guarded by `pkgFileMu` around seek+read inside `readPackedFileFromPackage`); on success `p.flLock.Lock(); fi.Md5.UpdateWith(sum); p.flLock.Unlock()`; then under `md5Mu` pop the waiter list, unlock, invoke every callback with the result (or the error — including the unknown-state error from Task 5).

- [x] **Step 4: Run tests** — `go test ./pkg/ddpackage/` → PASS; `go test -race ./pkg/ddpackage/` → PASS.
- [x] **Step 5: Commit** — `git add pkg/ddpackage/ddpackage.go pkg/ddpackage/ddpackage_test.go; git commit -m "fix: single-flight md5 computation, publish hashes under flLock"`

### Task 11: background-work tracker without WaitGroup misuse

**Files:** Modify: `internal/gui/gui.go` (field ~:139-144, `NewApp`), `internal/gui/pack.go` (:127/:129), `internal/gui/unpack.go` (:59/:61), `internal/gui/package.go` (:298/:300); Tests: `internal/gui/z_spec_helpers_test.go` (:113, :136), new `internal/gui/background_test.go`

**Interfaces:**
- Produces: `type backgroundTracker struct { mu sync.Mutex; cond *sync.Cond; count int }` with `Start()`, `Done()`, `Wait()`; App field `bgWork backgroundTracker` (cond created in `NewApp` via `sync.NewCond(&app.bgWork.mu)`).

- [x] **Step 1: Write the failing test** — `TestBackgroundTrackerConcurrentStartAndWait`: goroutine loop calling `Start()`/`Done()` while the main goroutine calls `Wait()` in a loop for a bounded number of rounds — today's `sync.WaitGroup` pattern panics/races with `Add` concurrent with `Wait`; the tracker must complete without panic (FAIL today if written against `bgWG` — write it against `bgWork` after the rename, verifying semantics: `Wait` returns only when `count == 0`).
- [x] **Step 2: Implement** `Start` (`count++`), `Done` (`count--; if count == 0 { cond.Broadcast() }`), `Wait` (`for count > 0 { cond.Wait() }`).
- [x] **Step 3: Replace all `a.bgWG` uses** — `a.bgWG.Add(1)` → `a.bgWork.Start()`; `defer a.bgWG.Done()` → `defer a.bgWork.Done()`; test helpers `a.bgWG.Wait()` → `a.bgWork.Wait()`; delete the `bgWG` field, update its doc comment.
- [x] **Step 4: Run tests** — `go test ./internal/gui/` → PASS.
- [x] **Step 5: Commit** — `git add internal/gui/gui.go internal/gui/pack.go internal/gui/unpack.go internal/gui/package.go internal/gui/z_spec_helpers_test.go internal/gui/background_test.go; git commit -m "fix: track background work with a condvar so Add can not race Wait"`

### Task 12: save timers stopped (and flushed) on package replace

**Files:** Modify: `internal/gui/gui.go` (`replacePackage` ~:288), `internal/gui/package.go` (save callbacks ~:540-585); Test: `internal/gui/package_save_test.go` (new — unique to this task so it can run in parallel)

**Interfaces:**
- Produces: `func (a *App) stopSaveTimersFor(old *ddpackage.Package)` (stops `tagSaveTimer` + `resSaveTimers`; a timer stopped before firing performs its save synchronously on `old`).

- [x] **Step 1: Write the failing test** — `TestReplacePackageFlushesPendingTagSave`: load fixture, `a.pkg.Tags().Tag("pending", ...)`, `a.saveUnpackedTags()` (schedules 500 ms timer), immediately `a.resetPkg()` → `require.Eventually` the tags file on disk contains `pending` (today the timer survives and writes after close / races — FAIL or flaky).
- [x] **Step 2: Implement `stopSaveTimersFor`** — tag timer: `if t := a.tagSaveTimer; t != nil && t.Stop() { if err := old.SaveUnpackedTags(); err != nil { log.WithError(err)... } }; a.tagSaveTimer = nil`. Resource timers: for each key, `Stop()` true → save synchronously: suffix `.dungeondraft_wall` → `old.SaveUnpackedWall(key)`, else `old.SaveUnpackedTileset(key)`; clear the map. (A timer that already fired is left alone — its callback holds the same `old` and writes are harmless for unpacked packages.)
- [x] **Step 3: Call it from `replacePackage`** — before `old.Close()`, only when `old != nil && old != pkg`.
- [x] **Step 4: Drop the redundant `fyne.Do` wrappers** around `a.showErrorDialog(err)` in the three save callbacks (the helper already queues — review finding "double-queued layering").
- [x] **Step 5: Run tests** — `go test ./internal/gui/` → PASS.
- [x] **Step 6: Commit** — `git add internal/gui/gui.go internal/gui/package.go internal/gui/package_save_test.go; git commit -m "fix: flush pending tag and metadata saves when the package is replaced"`

### Task 13: `updateFromPaths` — dialog built on the Fyne goroutine, disk I/O off it

**Files:** Modify: `internal/gui/pack.go` (`updateFromPaths` ~:203-247, `showPackageUpdateDialog` ~:283-307, `packageUpdateContent` ~:325-350)

- [x] **Step 1: Build the dialog inside `fyne.Do`** — `showPackageUpdateDialog` creates the bindings outside (safe), then one `fyne.Do(func(){ d.dialog = dialog.NewCustomWithoutButtons(...); d.dialog.Show() })` that also builds the content (`widget.NewProgressBar().Bind`, `widget.NewActivity().Start` currently run on the watcher goroutine). `hide()` already runs inside `fyne.Do`, queued after construction ⇒ FIFO ordering keeps `d.dialog` non-nil when `hide` reads it. `packageUpdateHook` still fires synchronously after `showPackageUpdateDialog` returns (tests read only the plain `progress` field).
- [x] **Step 2: Move `LoadTags`/`LoadResourceMetadata` off the Fyne goroutine** — in `updateFromPaths`, run both on the update goroutine **before** the `fyne.Do` (still inside `packageUpdateLock` ⇒ `replacePackage` waits ⇒ no use-after-close): collect `tagsErr`/`metaErr`, then `if tagsErr != nil { a.showErrorDialog(...) }` (the helper queues itself — call it *outside* the `fyne.Do`), and the `fyne.Do` only does `defer dlg.hide()` + `a.packageUpdated.Set(...)`.
- [x] **Step 3: Document the lock scope** — extend the `onPackageChanged` comment: the full rescan intentionally stays under `packageUpdateLock` (updates are unpacked-only, `Close()` is a no-op for them, and an update must never wait for the Fyne goroutine while holding the lock — see AGENTS.md). This closes the "consider limiting the critical section" finding as *accepted with rationale*.
- [x] **Step 4: Run tests** — `go test ./internal/gui/` → PASS (package_update + watcher suites cover dialog show/hide).
- [x] **Step 5: Commit** — `git add internal/gui/pack.go; git commit -m "fix: build the update dialog on the Fyne goroutine and load tags off it"`

### Task 14: serialize thumbnail generation with package updates

**Files:** Modify: `internal/gui/pack.go` (`genThumbnails` ~:772-832); Test: `internal/gui/package_watcher_test.go` (`thumbnailCount` ~:553)

- [x] **Step 1: Hold `packageUpdateLock` for the generation task** — first line in the goroutine: `a.packageUpdateLock.Lock(); defer a.packageUpdateLock.Unlock()` (then `pause`/`defer resume` as today). This excludes watcher updates from `GenerateThumbnailsProgress`'s file-list writes and makes `replacePackage` wait for the generation instead of closing the package under it. Deadlock check (documented in the code comment): nothing the generation calls (`fyne.Do`, binding `Set`, `pause`, `discard`) waits for `packageUpdateLock` while holding a lock the update path needs — `packageWatcher.report` releases `pw.mu` before invoking `onUpdate`.
- [x] **Step 2: Lock in the test** — `thumbnailCount` takes `a.packageUpdateLock` for the whole read (`currentPackage()` + `FileList().AsSlice()` iteration).
- [x] **Step 3: Run tests** — `go test ./internal/gui/ -run 'TestThumbnail|TestPackageWatcher' -v` → PASS; full `go test ./internal/gui/` → PASS.
- [x] **Step 4: Commit** — `git add internal/gui/pack.go internal/gui/package_watcher_test.go; git commit -m "fix: serialize thumbnail generation with package updates"`

**Phase 2 gate:** `. .\setcc-gcc.ps1; go build ./...; go test ./...; gofmt -l .; git diff --check` → clean. (Full `-race` runs once, in Task 32.)

---

## Phase 3 — Performance & behaviour

### Task 15: resource tree indexes + dead code + scale test

**Files:** Modify: `internal/gui/resource_tree/resource_tree.go` (`RebuildTree` ~:298, `ParentOf` ~:466, `IsVisiblySelected` ~:496, `updateVisibleSelection` ~:535, delete `startClickSequence` ~:661-670, struct fields ~:158); Test: `internal/gui/resource_tree/resource_tree_test.go`

**Interfaces:**
- New fields: `parentOf map[widget.TreeNodeID]widget.TreeNodeID`, `visibleSet map[widget.TreeNodeID]bool`.
- `ParentOf` becomes an O(1) lookup; `IsVisiblySelected` becomes `t.visibleSet[uid]` (nil-safe: missing key is false).

- [x] **Step 1: Write the failing test** — `TestSelectionAtScale`: build a package with 500 resources, each tagged with 3 tags (create in a temp dir + `BuildFileList`, tag via `pkg.Tags().Tag`), `RebuildTree()`; select every tag branch; assert `GetSelectedResources()` returns exactly 500 entries (each resource once), `IsVisiblySelected` is true for every occurrence uid, `ParentOf(child)` returns the tag branch for a sample. (Correctness at scale — deliberately **no** wall-clock assertions, per the review's flake warning.)
- [x] **Step 2: Run to verify it fails** — `go test ./internal/gui/resource_tree/ -run TestSelectionAtScale -v` → FAIL (undefined indexes) — actually first run proves it passes *without* indexes too; write the test first, then implement indexes, then re-run (the test guards behaviour, not speed).
- [x] **Step 3: Implement the indexes** — at the end of `RebuildTree` (after the node maps are complete): build `parentOf` by iterating `nodeTree` (child → parent, first wins); `updateVisibleSelection` stores its local `visibleSet` into `t.visibleSet` (and `Select`/`Unselect`/`restoreClickSelection` paths already call it). Delete `startClickSequence` (no callers — dead code).
- [x] **Step 4: Run tests** — `go test ./internal/gui/resource_tree/` → PASS.
- [x] **Step 5: Commit** — `git add internal/gui/resource_tree/resource_tree.go internal/gui/resource_tree/resource_tree_test.go; git commit -m "perf: index parents and the visible selection, drop dead click helper"`

### Task 16: `GetSelectedResources` computes `Selected` in a second pass

**Files:** Modify: `internal/gui/resource_tree/resource_tree.go` (`GetSelectedResources` ~:833-864); Test: `internal/gui/resource_tree/resource_tree_test.go`

- [x] **Step 1: Write the failing test** — `TestGetSelectedResourcesReportsDirectlySelectedResources`: fixture with tag `rock` on `Rock.png`; in **path mode** `Select(resPathUID)`, then `Select(tagBranchUID)` (branch popped first by the LIFO stack ⇒ first-seen uid is the branch child) → entry for Rock must have `Selected == true` (today: false ⇒ FAIL). Second case: only the branch selected → Rock `Selected == false`, and a resource with no direct selection under another selected branch → `false`.
- [x] **Step 2: Run to verify it fails** — `go test ./internal/gui/resource_tree/ -run TestGetSelectedResourcesReports -v` → FAIL.
- [x] **Step 3: Implement** — after the collection loop (before the sort), second pass: `resources[i].Selected = t.selectedIndexOfData(resources[i].Uid) != -1` (data-based, so any occurrence uid counts as "on its own"; a roll-up branch carries different data and yields `false`).
- [x] **Step 4: Run tests** — `go test ./internal/gui/resource_tree/` → PASS.
- [x] **Step 5: Commit** — `git add internal/gui/resource_tree/resource_tree.go internal/gui/resource_tree/resource_tree_test.go; git commit -m "fix: determine direct selection of resources in a second pass"`

### Task 17: context-menu *Unselect* walks to the selected ancestor

**Files:** Modify: `internal/gui/resource_tree/resource_tree.go` (`contextMenuItems` ~:985, new method near `Unselect` ~:738), `internal/gui/package.go` (`deselectResource` ~:1009); Test: `internal/gui/resource_tree/resource_tree_test.go`

**Interfaces:**
- Produces: `func (t *ResourceTree) UnselectIncludingAncestors(uid widget.TreeNodeID)` — if the node (by uid or data) is selected → `Unselect(uid)`; otherwise unselect the first selected ancestor from `AllParentsOf`.

- [x] **Step 1: Write the failing test** — `TestUnselectIncludingAncestorsUnselectsTheSelectedBranch`: `Select(tagBranchUID)` → `UnselectIncludingAncestors(childUID)` → branch no longer selected (today `Unselect(child)` silently no-ops ⇒ FAIL). Plus: `contextMenuItems()` on that roll-up-selected child returns exactly one item labelled `lang.X("tree.context.unselect", "Unselect")` (the label is asserted; the handler is a one-line delegation covered by the helper test).
- [x] **Step 2: Run to verify it fails** — `go test ./internal/gui/resource_tree/ -run 'TestUnselectIncludingAncestors|TestContextMenu' -v` → FAIL.
- [x] **Step 3: Implement** the helper; use it in the context-menu *Unselect* action and replace the body of `deselectResource` in `package.go` with a call to it (keeps the two paths in sync). Do **not** change `Unselect` itself (ctrl-unselect of a roll-up child must stay a no-op — you cannot represent "branch selected except one child").
- [x] **Step 4: Run tests** — `go test ./internal/gui/resource_tree/; go test ./internal/gui/` → PASS (bulk deselect spec tests included).
- [x] **Step 5: Commit** — `git add internal/gui/resource_tree/resource_tree.go internal/gui/resource_tree/resource_tree_test.go internal/gui/package.go; git commit -m "fix: unselect via the selected ancestor for roll-up selected nodes"`

### Task 18: fast second click flushes the pending click instead of cancelling it

**Files:** Modify: `internal/gui/resource_tree/resource_tree.go` (`MouseDown` ~:925-972, new helper next to `cancelPendingClick` ~:653); Test: `internal/gui/resource_tree/resource_tree_node_test.go`

**Interfaces:**
- Produces: `func (t *ResourceTree) flushPendingClick()` — if a debounced click is pending, apply it now (`selectOnly` + `endClickSequence`), else no-op.

- [x] **Step 1: Write the failing test** — `TestFastSecondClickKeepsTheFirstClick`: select node B; `MouseDown` plain on A (pending); immediately `MouseDown` ctrl on B (within `DoubleTapDelay`); after the timers settle, selection must be exactly `[A]` (first click flushed, then ctrl-unselect of B) — today the pending A is cancelled and the selection ends empty (FAIL). Use the existing `waitForPendingClick` helper from Task 0's fix for settling.
- [x] **Step 2: Run to verify it fails** — `go test ./internal/gui/resource_tree/ -run TestFastSecondClickKeepsTheFirstClick -v` → FAIL.
- [x] **Step 3: Implement `flushPendingClick`** — under `clickMu`: if `clickTimer == nil` return; capture `lastClickUID`, `cancelPendingClickLocked()`, unlock, then call `applyPendingClick(uid)` (it re-checks `lastClickUID`, which was not cleared, then `selectOnly` + `endClickSequence`).
- [x] **Step 4: Call it in `MouseDown`** — in the ctrl and shift branches *before* `endClickSequence()`; in the new-click-sequence branch *before* cloning `clickSelection` (so the restored double-click selection includes the flushed click).
- [x] **Step 5: Run tests** — `go test ./internal/gui/resource_tree/` → PASS (all existing click/double-click tests included).
- [x] **Step 6: Commit** — `git add internal/gui/resource_tree/resource_tree.go internal/gui/resource_tree/resource_tree_node_test.go; git commit -m "fix: apply a pending debounced click when a new click sequence starts"`

### Task 19: bulk pane updates in place (pane rebuild only on kind/resource change)

**Files:** Modify: `internal/gui/package.go` (`buildPackageTreeAndInfoPane` closure ~:136-158, `buildBulkOpsPane` ~:790, `buildBulkTagsPane` ~:813, `buildBulkResourceList` ~:900, `updateBulkResourceNode` ~:965, `buildBulkTagAddRow` ~:1024); Test: `internal/gui/package_pane_test.go` (new — unique to this task so it can run in parallel)

**Interfaces:**
- `buildBulkOpsPane(tree *resource_tree.ResourceTree, resources []resource_tree.SelectedResource, editable bool) (fyne.CanvasObject, *bulkPane)`
- `type bulkPane struct { … }` holding `resources`, `byUID map[widget.TreeNodeID]resource_tree.SelectedResource`, `ids binding.StringList` (+ its `[]string`), `updateTags func()`, the bound tag list, `tree`, `editable`.
- `func (b *bulkPane) setResources(resources []resource_tree.SelectedResource)` — replaces slice + map, rebuilds the id strings, `ids.Reload()`, calls `updateTags()`.
- `updateBulkResourceNode(b *bulkPane, uid string, obj fyne.CanvasObject)` (drops the index param; resolves via `byUID`).
- Closure state in `buildPackageTreeAndInfoPane`: `paneMode` (`none`/`single`/`bulk`), `singleInfo *structures.FileInfo`, `bulk *bulkPane`.

- [x] **Step 1: Write the failing test** — `TestBulkPaneIsRebuiltOnlyWhenTheKindChanges` (in `package_pane_test.go`): load fixture via `newSpecApp`-style helper; `tree.Select(resA)` → capture `content := a.ui.Get(nameResourcePane).(*fyne.Container).Objects[0]`; `tree.Select(resB)` (now bulk) → content changed, `a.ui.Get(nameBulkOpsPane) != nil`; `tree.Select(resC)` → content **identical pointer** (today rebuilt each time ⇒ FAIL).
- [x] **Step 2: Run to verify it fails** — `go test ./internal/gui/ -run TestBulkPaneIsRebuiltOnlyWhenTheKindChanges -v` → FAIL.
- [x] **Step 3: Restructure the bulk pane around `bulkPane`** — resource list becomes `widget.NewListWithData(b.ids, createBulkResourceNode, update)`; tag-remove button, add-tag button and `commonTags` read `b.resources` at invocation time (fixes stale captures too).
- [x] **Step 4: Rewrite the `OnSelectionChanged` closure** — `len(selected)==0`: rebuild `defaultPreview` only if mode ≠ none. Single `*ResourceFileInfoNode`: rebuild `buildInfoPane` unless mode == single **and** `singleInfo == data.Info` (skip; the selection only changed occurrences). Otherwise (bulk): if mode == bulk && `bulk != nil` → `bulk.setResources(tree.GetSelectedResources())` and return **without** `a.ui.Reset()`; else build the pane via `buildBulkOpsPane` and rebuild. `a.ui.Reset()` only on the rebuild branches.
- [x] **Step 5: Run tests** — `go test ./internal/gui/` → PASS, especially `z_spec_bulk_ops_test.go` (it selects resources incrementally, exercising `setResources`).
- [x] **Step 6: Commit** — `git add internal/gui/package.go internal/gui/package_pane_test.go; git commit -m "perf: update the bulk pane in place instead of rebuilding it per selection"`

### Task 20: debounce the filter entry

**Files:** Modify: `internal/gui/gui.go` (App fields, `NewApp`, `clean`), `internal/gui/package.go` (`buildPackageTree` listener ~:182-190); Test: `internal/gui/z_spec_resource_tree_test.go` (`TestSpecResourceFilter` ~:216)

**Interfaces:**
- New App fields: `filterDebounce time.Duration` (set to `const defaultFilterDebounce = 200 * time.Millisecond` in `NewApp`), `filterTimer *time.Timer`, `filterMu sync.Mutex` guarding the timer (the listener runs on the Fyne goroutine, `clean`/rebuild on others).

- [x] **Step 1: Update the spec test to wait** — in `TestSpecResourceFilter`, replace each immediate `assert` after `entry.SetText(...)` with `require.Eventually` polling `shownResources(t, tree)` against the expected slice (exact for the single-result cases, `ElementsMatch` for the multi-result ones). First run against current code → PASS (baseline), then the debounce lands and the same waits keep it green.
- [x] **Step 2: Implement the debounce** — in `buildPackageTree`: stop any previous `filterTimer` at the top (content is being replaced). Listener body: parse/validate as today (the validator already shows the error text), then under `filterMu`: stop the old timer, `filterTimer = time.AfterFunc(a.filterDebounce, func() { fyne.Do(func() { f, err := Parse(filter); if err != nil { return }; resourceTree.SetFilter(f); resourceTree.OpenBranch(PackageTreeRootID) }) })`. The `fyne.Do` closure reads the `filter` closure variable on the Fyne goroutine — no cross-goroutine read. `clean()`: stop the timer under `filterMu`.
- [x] **Step 3: Run tests** — `go test ./internal/gui/ -run 'TestSpecResourceFilter|TestSpec' -v` → PASS; full package PASS.
- [x] **Step 4: Commit** — `git add internal/gui/gui.go internal/gui/package.go internal/gui/z_spec_resource_tree_test.go; git commit -m "perf: debounce the resource tree filter entry"`

### Task 21: watcher caching, symlink safety, root check

**Files:** Modify: `internal/gui/package_watcher.go` (`watchNewFolder` ~:231-263, `cleanupWatchList` ~:213-229, `Watch` ~:265-293, struct), `pkg/ddpackage/pack.go` (`updateFromPaths` ~:223-257); Tests: `internal/gui/package_watcher_test.go`, `pkg/ddpackage/pack_test.go` (new)

**Interfaces:**
- New on `packageWatcher`: `root string`, `watched map[string]bool`, `watchMu sync.Mutex`; helper `func (pw *packageWatcher) isWatched(path string) bool`; `DiscardUnder` gains exact-path matching (`path == clean(dir)` counts as under it) so a single file can be discarded.

- [x] **Step 1: Write the failing tests**
  1. `TestUpdateFromPathsIgnoresPathsOutsideTheRoot` (`pkg/ddpackage/pack_test.go`): create a file outside the unpacked dir, call `UpdateFromPaths([]string{outsideFile})` → file list unchanged (FAIL today — the outside file gets indexed).
  2. `TestWatcherDoesNotWatchSymlinksLeavingThePackage`: create a symlink inside the fixture pointing to a temp dir outside it (`t.Skip` if `os.Symlink` fails — Windows needs developer mode), touch the target, `waitForWatcherIdle` → `assert.False(a.watcher().isWatched(symlinkPath))` and the outside file is not indexed (FAIL today — `os.Stat` follows the link and the dir gets watched).
  3. Keep the whole existing watcher suite green (it is the regression net for the cache).

- [x] **Step 2: Run to verify they fail** — `go test ./internal/gui/ -run TestWatcherDoesNotWatchSymlinks -v; go test ./pkg/ddpackage/ -run TestUpdateFromPathsIgnores -v` → FAIL.
- [x] **Step 3: Implement the watcher cache** — `Watch(dir)` stores `root` and records every added path in `watched`. `watchNewFolder`: `os.Lstat` (no follow); if symlink → `filepath.EvalSymlinks` + `utils.PathIsSub(pw.root, resolved)` or skip; if not a dir or already `watched` → return; add self + subdirs (skipping symlinks leaving root) under `watchMu`, recording in `watched`. `cleanupWatchList`: `Lstat` for existence + `watched` membership instead of `slices.Contains(WatchList(), …)`; delete from `watched` on remove. All `WatchList()` calls are gone.
  **Amended during execution (ruling R-T21, user-confirmed):** the cache desyncs permanently on delete-then-recreate inside the debounce window (or while paused), because the OS watch dies with the old directory while `watched[path]` stays true. Two additions, both implemented: (a) clear `watched[path]` on a `Remove`/`Rename` event **in `watch()` before `enqueue`** — i.e. before any debounce/pause gate, so it works while paused — so a recreate re-adds the OS watch; (b) `cleanupWatchList`'s `Remove` failure is logged **unless** it is `fsnotify.ErrNonExistentWatch`/`os.ErrNotExist` (fsnotify already dropped the entry when it dispatched the deletion event, so the old guard warned on every real folder deletion); genuine failures still warn. Test: `TestWatcherWatchesAFolderAgainAfterItWasDeletedAndRecreated`.
- [x] **Step 4: Root check in `pkg/ddpackage.updateFromPaths`** — right after `filepath.Abs`: `if isSub, _ := utils.PathIsSub(p.unpackedPath, absPath); !isSub { continue }` (keeps the existing inner check at :231 harmless).
- [x] **Step 5: Run tests** — `go test ./internal/gui/ -run TestPackageWatcher -v` → PASS; `go test ./pkg/ddpackage/` → PASS.
- [x] **Step 6: Commit** — `git add internal/gui/package_watcher.go internal/gui/package_watcher_test.go pkg/ddpackage/pack.go pkg/ddpackage/pack_test.go; git commit -m "fix: cache the watch set, skip symlinks leaving the pack root, ignore outside paths"`

### Task 22: cap `RepeatHorizontally` allocations

**Files:** Modify: `pkg/ddimage/ddimage.go` (`RepeatHorizontally` ~:401, `wallPathThumbnail` ~:439); Test: `pkg/ddimage/ddimage_test.go`

**Interfaces:**
- Produces: `const maxRepeatPixels int64 = 1 << 22` (≈4.19 M pixels ≈ 16 MiB of NRGBA — an order of magnitude more than any thumbnail needs).

- [x] **Step 1: Write the failing test** — `TestRepeatHorizontallySkipsAllocationsBeyondTheBudget`: 10×10 source, `RepeatHorizontally(img, maxRepeatPixels/100+1)` must return the **same** image (identity) — today it allocates width×height and returns a new image (FAIL; the allocation here is chosen to be ~16 MB, safe to run pre-fix).
- [x] **Step 2: Run to verify it fails** — `go test ./pkg/ddimage/ -run TestRepeatHorizontallySkips -v` → FAIL.
- [x] **Step 3: Implement** — first line of `RepeatHorizontally`: `if int64(width)*int64(bounds.Dy()) > maxRepeatPixels { return img }` (use `int64` math so huge widths can not overflow). In `wallPathThumbnail`, clamp `targetWidth` with `int64` math to `maxRepeatPixels / max(bounds.Dy(), 1)` before the `int` conversion, so a pathological height can not produce an overflowing/negative width.
- [x] **Step 4: Add the regression test for the pathological path** — `TestWallThumbnailOfVeryTallTexture`: 10×100000 image → `WallThumbnail` completes and the result width ≤ `maxThumbnailWidth` (written **after** the fix — pre-fix it would OOM).
- [x] **Step 5: Run tests** — `go test ./pkg/ddimage/` → PASS (all existing pixel-exact tests unchanged).
- [x] **Step 6: Commit** — `git add pkg/ddimage/ddimage.go pkg/ddimage/ddimage_test.go; git commit -m "fix: cap the repeat allocation for extreme texture aspect ratios"`

### Task 23: file preview size gate

**Files:** Modify: `internal/gui/package.go` (`buildFilePreview` ~:262-345); Test: `internal/gui/package_preview_test.go` (new — unique to this task so it can run in parallel)

**Interfaces:**
- Produces: `const maxTextPreviewBytes int64 = 4 << 20` (4 MiB).

- [x] **Step 1: Write the failing tests** — `TestFilePreviewSizeGate`: build a fixture dir (via `testutil.DefaultFixture().Write`) plus `data/long.json` (>200 lines, small) and `data/huge.json` (>4 MiB, few lines); load with `loadUnpackedSync`, then for each: `a.buildFilePreview(info)` → long file shows `nameTooLargeLabel` (line-count rule), huge file shows `nameTooLargeLabel` (size rule — today it reads 4 MiB first ⇒ FAIL on the huge case by construction: assert the gate is hit, i.e. the tooLarge branch is taken), and the fixture's small `data/default.dungeondraft_tags` shows `nameTextPreview`. Confirm `.json` is in `DefaultValidExt()` (it is) so the files are indexed.
- [x] **Step 2: Run to verify it fails** — `go test ./internal/gui/ -run TestFilePreviewSizeGate -v` → FAIL.
- [x] **Step 3: Implement** — in the non-image branch only: determine `size := info.Size`; if `size <= 0 && info.Path != ""` → `os.Stat` (unpacked files carry `Size == 0`; packed files carry the header size). If `size > maxTextPreviewBytes` → take the existing `tooLarge` branch **without** calling `LoadResource`. Otherwise read as today and replace `strings.Split(textContent, "\n")` with `bytes.Count(fileData, []byte("\n"))+1 > 200`. The image branch keeps reading unconditionally (a preview needs the bytes).
- [x] **Step 4: Run tests** — `go test ./internal/gui/` → PASS.
- [x] **Step 5: Commit** — `git add internal/gui/package.go internal/gui/package_preview_test.go; git commit -m "fix: gate the text preview on file size before reading"`

### Task 24: tag generation off the Fyne goroutine + discard its own writes

**Files:** Modify: `internal/gui/tag_gen.go` (`generateTags` ~:36-46, generate button ~:460-491), `internal/gui/package_watcher.go` (`DiscardUnder` — exact-path match from Task 21); Test: `internal/gui/package_watcher_test.go` (`TestTagGenerationUpdatesTheTagsWithoutSpammingTheWatcher` ~:568)

- [x] **Step 1: Tighten the existing test** — change `assert.LessOrEqual(t, updated-count, 1, …)` to `assert.Equal(t, 0, updated-count, "the watcher must not report the tags file the generation itself wrote")` (FAIL after the discard fix is absent — it currently reports exactly that one update).
- [x] **Step 2: Implement the discard** — inside `generateTags`, after `pkg.GenerateTagsProgress(...)` and **before** `defer a.resumePackageWatcher()` runs: `a.discardPackageWatcherChanges(filepath.Join(pkg.UnpackedPath(), "data", "default.dungeondraft_tags"))` (requires Task 21's exact-path matching in `DiscardUnder` — if Task 21's ordering differs, do the `DiscardUnder` extension here).
- [x] **Step 3: Run to verify** — `go test ./internal/gui/ -run TestTagGenerationUpdatesTheTags -v` → PASS after the fix (write the tightened assertion first, watch it fail, then apply Steps 2).
- [x] **Step 4: Move the button handler to a goroutine** — in the generate button's `OnTapped` (currently synchronous at :463-490): keep `progressDlg.Show()` + `a.disableButtons.Set(true)` + `generateBtn.Disable()` on the Fyne goroutine, then `go func() { a.generateTags(generator, progress.Report); fyne.Do(func() { progressDlg.Hide(); a.disableButtons.Set(false); generateBtn.Enable(); doneDlg.Show() }) }()` (same shape as `genThumbnails`; the `doneDlg.SetOnClosed → genTagsDlg.Hide` wiring stays inside the `fyne.Do`).
- [x] **Step 5: Run tests** — `go test ./internal/gui/` → PASS.
- [x] **Step 6: Commit** — `git add internal/gui/tag_gen.go internal/gui/package_watcher_test.go; git commit -m "fix: generate tags off the Fyne goroutine and discard the tags file it writes"`

**Phase 3 gate:** `. .\setcc-gcc.ps1; go build ./...; go test ./...; gofmt -l .; git diff --check` → clean.

---

## Phase 4 — CI & test coverage

### Task 25: workflow hardening

**Files:** Modify: `.github/workflows/build.yml`

- [x] **Step 1: Release waits for tests** — `release:` → `needs: [build, test]` (currently `needs: build` at :174).
- [x] **Step 2: Deterministic Makefile tab + verification** — replace the sed line (:117) with a shell-expanded real tab and fail the job if no tab landed:
  ```bash
  TAB=$'\t'
  sed -i "/^install:/a\\${TAB}install -Dm00755 usr/local/bin/\$(Exec)-cli \$(DESTDIR)\$(PREFIX)/bin/\$(Exec)-cli" ./install/dungeondraft-packager/Makefile
  grep -q $'^\tinstall -Dm00755' ./install/dungeondraft-packager/Makefile
  ```
  (Double quotes expand `${TAB}` to a literal tab; `\$` keeps `$(Exec)` literal for make; the `grep` is the hard guarantee the recipe line starts with a TAB.)
- [x] **Step 3: Test job extras** — after `go test ./...`:
  - exclude the scratch dir: `go list ./... | grep -v '/testing$' | xargs go test`
  - `test -z "$(gofmt -l .)"` and `go vet ./...` (fix any pre-existing vet findings in the same commit if trivial; otherwise report them before proceeding).
- [x] **Step 4: Add a `race` job** — same setup as `test` (Go 1.27.x, apt deps, `CGO_ENABLED: 1`) running `go test -race ./internal/gui/... ./pkg/ddpackage/...` (gcc on ubuntu supports the race runtime; this is the job that would have caught every Phase 2 finding). `release` does **not** wait for `race` (keeps tag releases fast; `test` remains the gate).
- [x] **Step 5: Validate YAML** — `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/build.yml'))"` or a careful visual check of indentation; commit and let CI confirm on push.
- [x] **Step 6: Commit** — `git add .github/workflows/build.yml; git commit -m "ci: gate releases on tests, verify the Makefile tab, add vet/gofmt/race jobs"`

### Task 26: pin the locale in `newTestApp`

**Files:** Modify: `internal/gui/gui_test.go` (`newTestApp` ~:36)

- [x] **Step 1: Add the pin** — `a.app.Preferences().SetString("locale", "en")` between `NewAppWithFyneApp` and `a.setup()`, with the same comment as `newSpecApp` (z_spec_helpers_test.go:24-26).
- [x] **Step 2: Run to verify** — `go test ./internal/gui/ -run 'TestAppSetNotAPackageContent|TestAppAddTagThroughTheTagPane' -v` → PASS (they already pass on an English machine; the pin makes them locale-independent — verify by `Set-Item Env:LANG`-style reasoning or code review).
- [x] **Step 3: Commit** — `git add internal/gui/gui_test.go; git commit -m "test: pin the locale of newTestApp to English"`

### Task 27: package test gaps (md5 round-trip, filter group collapse)

**Files:** Test: `pkg/ddpackage/pack_md5_test.go` (new — unique to this task so it can run in parallel with T21's `pack_test.go`), `pkg/ddpackage/filter/filter_test.go`

- [x] **Step 1: md5 round-trip** — `TestPackWritesMd5ThatLoadReadsBack`: write fixture pack (reuse `testutil.DefaultFixture` + `PackPackageProgress` to a temp file), `LoadFromPackedPath` → for a sample resource assert `fi.Md5.IsValid()` and it equals `hex(md5(fileBytes))` read from the unpacked source; also assert `readPackedFileFromPackage` accepts it (md5 verification path).
- [x] **Step 2: filter group collapse** — in `filter_test.go`, read the new logic at `filter.go:585` ("filter out empty filters before deciding whether the group collapses") and assert: `"(rock)"` behaves identically to `"rock"`; a group whose filters are all empty parses without error and matches nothing extra; nested `((a))` collapses; `"(rock tree)"` keeps OR semantics. Run: `go test ./pkg/ddpackage/filter/ -v`.
- [x] **Step 3: Run tests** — `go test ./pkg/ddpackage/...` → PASS.
- [x] **Step 4: Commit** — `git add pkg/ddpackage/pack_md5_test.go pkg/ddpackage/filter/filter_test.go; git commit -m "test: md5 round-trip through the pack header and filter group collapse"`

### Task 28: gui test gaps (pack.json ID rescan, report throttle, layouts)

**Files:** Test: `internal/gui/package_watcher_test.go`, `internal/gui/package_update_test.go`, `internal/gui/layouts/layouts_test.go` (new)

- [x] **Step 1: pack.json ID change forces a full rescan** — `TestPackageWatcherRescansWhenPackIDChanges`: like `TestPackageWatcherDetectsEditedPackJSONInPackageRoot` but change `"id": "testpack"` to a new id; via `packageUpdateHook` assert `dlg.progress == true` (rescan), and after `waitForPackageUpdate` assert every resource now lives under `res://packs/<newid>/…` and `pkg.ID()` is the new id. (Covers `reloadPackInfo`'s `oldID != pkg.ID()` return, previously untested.)
- [x] **Step 2: report throttle** — `TestPackageUpdateDialogReportThrottles`: build `packageUpdateDialog{progress: true, activityStr: binding.NewString(), activityProgress: binding.NewFloat()}`, attach a counting listener to `activityProgress`, call `d.report(0.5, "x", 0)` 100× → exactly 1 notification; `d.report(0.51, …)` → a second. Mirrors `progress_test.go`.
- [x] **Step 3: layouts unit tests** — `layouts_test.go` with `test.NewApp()` in `TestMain`: for `paddedLayout`, `leadingCenterLayout`, `stackedLayout`, `fixedSizeLayout` assert child positions/sizes from `Layout(objs, size)` and `MinSize` (including the invisible-child and empty-object cases). Run: `go test ./internal/gui/layouts/ -v`.
- [x] **Step 4: Run tests** — `go test ./internal/gui/` → PASS.
- [x] **Step 5: Commit** — `git add internal/gui/package_watcher_test.go internal/gui/package_update_test.go internal/gui/layouts/layouts_test.go; git commit -m "test: pack id rescan, update report throttle, layout units"`

**Phase 4 gate:** `. .\setcc-gcc.ps1; go test ./...; gofmt -l .; git diff --check` → clean.

---

## Phase 5 — Documentation

### Task 29: `docs/design/named_ui.md` past tense + filter-help names

**Files:** Modify: `docs/design/named_ui.md`, `internal/gui/resource_tree/resource_tree.go` (doc comment on `GetSelectionAndChildData` ~:795)

- [x] **Step 1: Rewrite "The problem" as history** — the section (:6-20) describes `uiRegions` (`internal/gui/regions.go`, which no longer exists) in present tense: convert to past tense ("The parts of the interface **were** addressed by name. They **were** kept in `uiRegions`, a struct …", "…therefore **meant** three edits"), keeping the code sample as a historical illustration. Fix any other present-tense references to `uiRegions` in the file (the "trade-offs" section mentions "the race on `uiRegions` disappears" — that one is already fine).
- [x] **Step 2: Names-table note** — add a line under the names table: `filterHelpSectionList` and `filterHelpScroll` live in the filter-help dialog overlay and are **not** resolvable through `a.ui`; the spec tests build a second `named.Cache` rooted at `popup.Content` (`z_spec_filter_help_test.go:43-52`).
- [x] **Step 3: Fix the Go doc comment** — `GetSelectionAndChildData`: state that it returns the data of the selected resources **and the tag branches** they are shown under (it appends `ResourceTagBranch` data too).
- [x] **Step 4: Verify** — re-read for consistent tense; no code changes → `go build ./...` still green (comment-only edit).
- [x] **Step 5: Commit** — `git add docs/design/named_ui.md internal/gui/resource_tree/resource_tree.go; git commit -m "docs: describe uiRegions as history, note filter-help name resolution"`

### Task 30: spec wording + translation typo

**Files:** Modify: `docs/design/ui_spec.md`, `internal/gui/translation/en-US.json` (:133)

- [x] **Step 1: `ui_spec.md`** — `:11` `with a "about" ion button` → `with an "about" icon button`; then sweep the whole file for missing/wrong articles (`a info` → `an info`, `a output` → `an output`, `a Open Pack` → `an Open Pack`, `a edit …` → `an edit …`, insert missing articles where a noun phrase has none). Wording only — the tests assert `lang.X` strings, not this file.
- [x] **Step 2: `en-US.json`** — `"task.updatePackage.text": "Updating pack recources ..."` → `"Updating pack resources ..."`. Only the English catalog (Crowdin syncs the rest).
- [x] **Step 3: Verify** — `go test ./internal/gui/ -run TestSpec -v` → PASS (spec wording is not asserted anywhere).
- [x] **Step 4: Commit** — `git add docs/design/ui_spec.md internal/gui/translation/en-US.json; git commit -m "docs: fix icon-button wording and the pack resources typo"`

### Task 31: CHANGELOG corrections + entries for this fix round

**Files:** Modify: `CHANGELOG.md`

- [x] **Step 1: Fix the two wrong existing lines** —
  - v2.1.0 feature bullet (:33-35): the filter entry shows the syntax **in its placeholder** only (`tree.filter.toolTip` is plain text) — drop "and in its tool tip".
  - v2.0.4 heading (:104): link target `releases/tag/v2.0.3` → `releases/tag/v2.0.4`.
- [x] **Step 2: Add an `## [Unreleased]` section at the `<!-- insertion marker -->`** (above v2.1.0 — v2.1.0 has no tag yet, but its heading carries a date, so new work gets its own section) with a `### Bug Fixes` list covering the user-visible items of this plan: zip-slip / hostile-pack bounds, pack-name sanitisation, stuck progress dialogs + buttons after load errors, tag generation off the UI thread with no redundant reload, debounced filter, roll-up deselect/context-menu/fast-click fixes, save flush on package replace, md5 single-flight + unknown-state error, preview size gate, repeat-allocation cap, `IsEmpty` fix (`pkg/structures/stack.go` — the only public-API bug fix of the range, previously undocumented). Keep the existing house style (lowercase, wrapped at ~90 cols).
- [x] **Step 3: Verify** — re-read against Keep-a-Changelog structure; no code changes.
- [x] **Step 4: Commit** — `git add CHANGELOG.md; git commit -m "docs: correct filter tooltip and v2.0.4 links, log the review fixes"`

---

### Task 32: final verification & plan commit

- [x] **Step 1: Full build + suite** — `. .\setcc-gcc.ps1; go build ./...; go test ./...` → PASS.
- [x] **Step 2: gofmt + whitespace + LF** — `gofmt -l .` (empty), `git diff --check` (clean). LF is enforced by `.gitattributes`; if `git diff --check` reports `CR` at EOL anywhere, fix the file to LF before committing.
- [x] **Step 3: Single race run** — `. .\setcc-gcc.ps1; go test -race ./internal/gui/... ./pkg/ddpackage/...` → PASS (minutes; gcc required). Any race reported → fix in the owning task's area, re-run once.
- [x] **Step 4: Plan document** — the plan was committed in Task 0; if any schedule amendment was made during execution, commit it now (`git add docs/plans/2026-11-08-review-fixes.md; git commit -m "docs: sync the review-fixes plan with execution"`).
- [x] **Step 5: Report** — list commits per phase and paste the final test/race/gofmt outputs for the user.

---

## Execution amendments (recorded during the run — committed with Step 4)

Every change below was confirmed with the human before it was applied; each
supersedes the corresponding task text above. The full ledger (including
deferred Minor findings for the final review) lives in
`docs/sdd/2026-11-08-review-fixes/progress.md`.

| Ruling | What changed |
|---|---|
| **R-T10** | The brief's `computeMd5` publish-under-`flLock` **deadlocked** `UpdateResources(Md5:true)` (it holds `flLock` across `innerWg.Wait()`). `UpdateResources` now snapshots the file list under `flLock` and releases it before spawning workers; added `TestUpdateResourcesMd5DoesNotDeadlock` (no existing test called `UpdateResources`). |
| **R-T11** | `backgroundTracker.Done()` gained a **panic on a mis-paired call** (`sync: negative backgroundTracker counter…`), restoring WaitGroup's fail-fast that the mandated body dropped. |
| **R-T16** | The brief's test scenario was **unsatisfiable** (path mode has no `tag://` branches; its case 1 already passed and the fix was a no-op there). Test rewritten in **tag mode** with a resource under ≥2 tags — RED at HEAD, GREEN with the fix. |
| **R-T20** | The brief's debounce design **crashed the test suite** (`concurrent map read and map write`: under the test driver `fyne.Do` runs the rebuild on the timer goroutine). Added a `filterMu` fence around the rebuild and the test's polling conditions; `require.EventuallyWithT` replaces `require.Eventually`. |
| **R-T22** | The absolute `maxRepeatPixels` budget was **rejected** in favour of an input ratio: `maxRepeatRatio = 64`, and an over-limit request is **clamped to `63 × srcWidth`** (one source-repeat below the limit) rather than returning unchanged — "thumbnails might not be perfect but they are still closer". `wallPathThumbnail` unchanged. Tall-texture test uses 10×10000 (not 10×100000) to keep the suite light. |
| **R-T24** | The plan's strict-zero watcher assertion is **accepted as non-deterministic by construction** (the discard races fsnotify delivery): stress it at Task 32 with `-count=10`; the optional product fix is deferred. |
| **R-T12** | `stopSaveTimersFor`'s lock-free map iteration was a **crash-class race** (reproduced: `fatal error: concurrent map iteration and map write`). Added `saveTimerLock` guarding every access to `tagSaveTimer`/`resSaveTimers`, saves performed outside the lock. |
| **R-T13** | Tags/metadata now load off the Fyne goroutine while UI readers of `pkg.Tags()` hold no lock — **accepted with a recorded rationale** (plan/AGENTS.md document lock-free UI reads as an accepted invariant; window bounded by the modal dialog; same class pre-exists for pack info). Flagged to the `-race` run and the final review. |
| **R-T21** | The `watched` cache desynced permanently on delete-recreate inside the debounce (or while paused). Amended Step 3 (see above): clear the entry on `Remove`/`Rename` in `watch()` before `enqueue`, and stop warning for already-gone watches. |
| **R-T28** | `leadingCenterLayout.MinSize` omits `l.padding` while `Layout` adds it — a real bug the new tests pin. **Reported, not fixed** (test-only task): recorded in the task report and in the test comment; production fix deferred to the final review. |
| **R-FLAKE** | Two watcher-test flakes were investigated (`investigation-watcher-double-report.md`): verdict **PRE-EXISTING** (deferred NTFS directory metadata → spurious fsnotify `Write` on directories), no task in this round caused it. **Ad-hoc task FL1 added** (not in the original 33): test-harness hardening only — `sync.Once` around `close(running)` and the rescan hook scoped inside the `packageUpdateLock` closure. The optional product fix (skipping bare `Write`-on-directory events) is deferred. |
| **R-LADDER** | After calibration the human approved the first tier upgrade: reviewers of **concurrency/security diffs** run on `opencode-go/mimo-v2.6-pro` (T10, T14, T20, T24 and their re-reviews); everything else stayed on `opencode-go/mimo-v2.6-flash`. |
