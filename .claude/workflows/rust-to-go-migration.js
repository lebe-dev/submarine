export const meta = {
  name: 'rust-to-go-migration',
  description: 'Faithful 1-to-1 port of the submarine Rust CLI to Go 1.26 (urfave/cli + bubbletea), then build/vet/test until green',
  whenToUse: 'Run on the dev branch after MIGRATION-SPEC.md and the go.mod/internal scaffold exist. Ports every Rust module to Go and reconciles the whole tree with an iterative build-fix loop.',
  phases: [
    { title: 'Core', detail: 'port internal/subtitle (everyone depends on it)' },
    { title: 'Domains', detail: 'port backup/doctor/importer/rename/translationstatus/verify in parallel' },
    { title: 'CLI infra', detail: 'dto/jsonout/logging/utils + format enums (package cli)' },
    { title: 'Commands & TUI', detail: 'port 13 command handlers + bubbletea compare view in parallel' },
    { title: 'Wiring', detail: 'cmd/sm/main.go — urfave/cli command tree + dispatch; first full build' },
    { title: 'Tests', detail: 'port tests/*.rs integration tests' },
    { title: 'Build & fix', detail: 'iterate go build / vet / test, repairing cross-package mismatches until green' },
  ],
}

const REPO = '/Users/eugene/pro/submarine'
const SPEC = `${REPO}/MIGRATION-SPEC.md`

// ---- shared schemas ----
const PORT_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['pkg', 'filesWritten', 'buildOk', 'notes'],
  properties: {
    pkg: { type: 'string' },
    filesWritten: { type: 'array', items: { type: 'string' } },
    buildOk: { type: 'boolean', description: 'true if the self-build check passed' },
    notes: { type: 'string', description: 'anything the build-fix phase should know: deviations, TODOs, suspected cross-package risks' },
  },
}

const BUILD_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['buildOk', 'vetOk', 'testOk', 'remaining', 'progressed', 'filesChanged'],
  properties: {
    buildOk: { type: 'boolean', description: 'go build ./... exited 0' },
    vetOk: { type: 'boolean', description: 'go vet ./... exited 0' },
    testOk: { type: 'boolean', description: 'go test ./... exited 0' },
    remaining: { type: 'string', description: 'concise summary of remaining errors (empty if none)' },
    progressed: { type: 'boolean', description: 'true if this iteration reduced the error count vs what you started with' },
    filesChanged: { type: 'array', items: { type: 'string' } },
  },
}

const baseRules = `You are performing a FAITHFUL 1-to-1 port of the "submarine" Rust CLI to Go 1.26.
Read the binding contract FIRST and obey it exactly: ${SPEC}
Working dir: ${REPO}. Rust sources stay in place; you only ADD Go files.
1-to-1 means: identical algorithms, validation rules, error message strings, output text, and behavior. Do NOT refactor, improve, rename public concepts, or add features.
Dependencies are already in the module cache — offline 'go build' works. If go ever reports a missing module or go.sum entry, run 'go mod tidy' on a Bash call with dangerouslyDisableSandbox=true.`

// =================== Phase 1: Core (subtitle) ===================
phase('Core')
const core = await agent(
  `${baseRules}

MODULE: subtitle (the core domain — every other package depends on it; its Go API is PINNED in ${SPEC}).
Read these Rust sources and port them 1-to-1:
- ${REPO}/src/lib/subtitle/model.rs
- ${REPO}/src/lib/subtitle/ports.rs
- ${REPO}/src/lib/subtitle/service.rs

Write (package subtitle, dir ${REPO}/internal/subtitle/):
- model.go        (SubtitleError+kinds, validated newtypes, Subtitle, ParseTimestamp/FormatTimestamp, SubtitleUpdate, *Report structs — EXACTLY as pinned in the spec)
- ports.go        (the Service interface)
- service.go      (SubRipService incl. path-traversal protection and SRT parsing/writing)
- model_test.go   (port every #[cfg(test)] test from model.rs)
- service_test.go (port every #[cfg(test)] test from service.rs)

The pinned API in the spec is mandatory — downstream packages are written against those exact names/signatures.
Self-check: run 'cd ${REPO} && go build ./internal/subtitle/ && go vet ./internal/subtitle/ && go test ./internal/subtitle/' and fix until all pass.`,
  { label: 'port:subtitle', phase: 'Core', schema: PORT_SCHEMA },
)
log(`subtitle ported (buildOk=${core?.buildOk}). notes: ${core?.notes ?? ''}`)

// =================== Phase 2: Domains (parallel) ===================
phase('Domains')
const domains = [
  {
    pkg: 'backup', dir: 'internal/backup',
    rust: ['src/lib/backup/model.rs', 'src/lib/backup/ports.rs', 'src/lib/backup/service.rs'],
    go: ['model.go', 'ports.go', 'service.go', 'service_test.go'],
    hint: 'BackupService.CreateBackup(path) (*string, error): nil => file did not exist (no backup). Reproduce the timestamped backup filename format exactly.',
  },
  {
    pkg: 'doctor', dir: 'internal/doctor',
    rust: ['src/lib/doctor/model.rs', 'src/lib/doctor/ports.rs', 'src/lib/doctor/service.rs'],
    go: ['model.go', 'ports.go', 'service.go', 'service_test.go'],
    hint: 'Depends on internal/subtitle (read its .go files for exact signatures). Preserve diagnose/fix behavior, issue severities, backup filename format, and the empty-line/separator normalization rules verbatim.',
  },
  {
    pkg: 'importer', dir: 'internal/importer',
    rust: ['src/lib/import/model.rs', 'src/lib/import/ports.rs', 'src/lib/import/service.rs'],
    go: ['model.go', 'ports.go', 'service.go', 'service_test.go'],
    hint: 'Package is named "importer" (import is a Go keyword). CSV parsing uses encoding/csv with a configurable delimiter rune; anchored parsing mirrors the Rust line logic. Returns *subtitle.SubtitleError for the relevant variants (read internal/subtitle/model.go).',
  },
  {
    pkg: 'rename', dir: 'internal/rename',
    rust: ['src/lib/rename/model.rs', 'src/lib/rename/ports.rs', 'src/lib/rename/service.rs'],
    go: ['model.go', 'ports.go', 'service.go', 'service_test.go'],
    hint: 'No subtitle dependency. Port TemplateContext (builder methods -> Go With* methods or fields). Replace tera with text/template — rewrite the default template "{{ name }}{{ separator }}S{{ season }}{{ separator }}E{{ episode }}.srt" to valid Go template syntax and render the SAME output. Replace the glob crate with filepath.Glob + manual case-insensitive matching (no new deps). Preserve series-mode episode auto-increment and collision detection.',
  },
  {
    pkg: 'translationstatus', dir: 'internal/translationstatus',
    rust: ['src/lib/translation_status/model.rs', 'src/lib/translation_status/service.rs'],
    go: ['model.go', 'service.go', 'service_test.go'],
    hint: 'Depends on internal/subtitle. Keep the free function: func CheckTranslationStatus(...) with the same signature shape and the same progress/next-chunk math.',
  },
  {
    pkg: 'verify', dir: 'internal/verify',
    rust: ['src/lib/verify/model.rs', 'src/lib/verify/service.rs'],
    go: ['model.go', 'service.go', 'service_test.go'],
    hint: 'Depends on internal/subtitle. Keep the free function: func CompareSubtitles(...). Preserve match/mismatch/offset-detection logic and any percentage rounding exactly.',
  },
]

const domainResults = await parallel(domains.map((d) => () =>
  agent(
    `${baseRules}

MODULE: ${d.pkg}  (dir ${REPO}/${d.dir}/, package ${d.pkg})
Read & port these Rust sources 1-to-1:
${d.rust.map((r) => `- ${REPO}/${r}`).join('\n')}

The subtitle package is already ported — read ${REPO}/internal/subtitle/*.go for exact signatures of any types you reference.
Module-specific notes: ${d.hint}

Write (package ${d.pkg}): ${d.go.join(', ')}. Port every #[cfg(test)] test into *_test.go in the same package.
Self-check: 'cd ${REPO} && go build ./${d.dir}/ && go vet ./${d.dir}/ && go test ./${d.dir}/' — fix until all pass. Touch ONLY files under ${d.dir}/.`,
    { label: `port:${d.pkg}`, phase: 'Domains', schema: PORT_SCHEMA },
  )))
domainResults.filter(Boolean).forEach((r) => log(`${r.pkg}: buildOk=${r.buildOk} ${r.notes ? '— ' + r.notes : ''}`))

// =================== Phase 3: CLI infra ===================
phase('CLI infra')
const cliInfra = await agent(
  `${baseRules}

MODULE: cli infrastructure (dir ${REPO}/internal/cli/, package cli). NO command handlers here, NO urfave command tree here (that lives in cmd/sm/main.go to avoid an import cycle — see the spec's cycle note).
Read & port 1-to-1:
- ${REPO}/src/bin/cli/dto.rs         -> dto.go    (all DTO structs; JSON tags = snake_case serde names; omitempty where skip_serializing_if; format_duration_readable helper)
- ${REPO}/src/bin/cli/json_output.rs -> jsonout.go (OutputSuccess/OutputError + envelopes)
- ${REPO}/src/bin/cli/logging.rs     -> logging.go (log/slog; --log-level, --log-target console|file; lowercase messages)
- ${REPO}/src/bin/cli/utils.rs       -> utils.go  (shared helpers; read the Rust file to see exact functions, e.g. range parsing)
- the enums in ${REPO}/src/bin/cli/cli.rs (OutputFormat, ExportFormat, ImportFormat) -> format.go (with parse-from-string + String())

All domain packages and subtitle are already ported under ${REPO}/internal/ — read their .go files for exact signatures (e.g. internal/subtitle/model.go for SubtitleDto.FromSubtitle).
Port any #[cfg(test)] tests into *_test.go.
Self-check: 'cd ${REPO} && go build ./internal/cli/ && go vet ./internal/cli/ && go test ./internal/cli/' — fix until pass. Touch ONLY files directly under internal/cli/ (NOT internal/cli/cmd or internal/cli/output).`,
  { label: 'port:cli-infra', phase: 'CLI infra', schema: PORT_SCHEMA },
)
log(`cli infra ported (buildOk=${cliInfra?.buildOk}). notes: ${cliInfra?.notes ?? ''}`)

// =================== Phase 4: Commands & TUI (parallel) ===================
phase('Commands & TUI')
const cmdGroups = [
  { label: 'cmd:reads', files: ['get', 'info', 'describe'],
    hint: 'get: index or range (e.g. 120-123). info: stats DTO. describe: ALWAYS JSON; reproduce the full command schema output exactly.' },
  { label: 'cmd:mutate', files: ['set', 'add', 'delay'],
    hint: 'All support --dry-run and auto-backup. delay OFFSET allows leading hyphen (+100/-500). Preserve sample_before/after on delay.' },
  { label: 'cmd:io', files: ['import', 'export'],
    hint: 'import: --format csv|anchored, --reference, --delimiter, --dry-run, --force (confirmation prompt). export: --format anchored, range. Use internal/importer + internal/subtitle.' },
  { label: 'cmd:compare-domains', files: ['verify', 'translation_status'],
    hint: 'verify -> internal/verify.CompareSubtitles, optional --range. translation_status (alias ts) -> internal/translationstatus.CheckTranslationStatus, --reference/-r, --chunk-size.' },
  { label: 'cmd:complex', files: ['doctor', 'mass_rename'],
    hint: 'doctor: --fix mutates + backup. mass_rename: --dry-run/--force/--series-mode/--name/--season/--language/--separator/--file-template; uses internal/rename.' },
]

const tuiAgent = () => agent(
  `${baseRules}

MODULE: TUI compare view + its thin command handler.
Read & port 1-to-1:
- ${REPO}/src/bin/cli/output/compare.rs  -> ${REPO}/internal/cli/output/compare.go (package output)
- ${REPO}/src/bin/cli/cmd/compare.rs     -> ${REPO}/internal/cli/cmd/compare.go    (package cmd: thin HandleCompare that loads both files via subtitle service and calls output.RunTUI)

Replace ratatui+crossterm with charmbracelet/bubbletea + lipgloss (already cached; do NOT use bubbles). Map the ratatui immediate-mode UI onto bubbletea Model/Update/View:
- App state -> a bubbletea Model struct (same fields: two subtitle slices + filenames, selectedIndex, scrollOffset, mode Normal/JumpInput/SearchInput, inputBuffer, inputError, searchMatches, currentMatchIndex, shouldCenterOnNextRender, pendingGPress).
- Reproduce EXACTLY: navigation j/k/up/down, gg/G, r (random via math/rand/v2), ':' jump, '/' search, n/N next/prev match, q/Esc quit, the manual scroll/viewport math (calculateViewportHeight, updateScrollOffset, centerSelectedItem), side-by-side 50/50 panes, the status/help line text, error overlay, 60-char truncation, placeholder for missing rows, and the selected-row styling (blue bg / black fg / bold).
Read ${REPO}/internal/subtitle/*.go and ${REPO}/internal/cli/*.go for signatures.
Self-check: 'cd ${REPO} && go build ./internal/cli/output/ && go vet ./internal/cli/output/' until clean (the cmd/compare.go file builds in the final phase).`,
  { label: 'port:tui-compare', phase: 'Commands & TUI', schema: PORT_SCHEMA },
)

const cmdThunks = cmdGroups.map((g) => () =>
  agent(
    `${baseRules}

MODULE: command handlers (package cmd, dir ${REPO}/internal/cli/cmd/). Port these handlers 1-to-1, ONE Go file per command:
${g.files.map((f) => `- ${REPO}/src/bin/cli/cmd/${f}.rs  -> ${REPO}/internal/cli/cmd/${f}.go`).join('\n')}

Each Rust 'pub fn handle(...)' becomes an exported func named Handle<Command> (e.g. HandleGet, HandleSet, HandleMassRename, HandleTranslationStatus) in package cmd.
Notes: ${g.hint}
Everything you call is already ported: read internal/subtitle/*.go, internal/<domain>/*.go, and internal/cli/*.go for exact signatures (OutputFormat, OutputSuccess/OutputError, the DTOs, utils). Preserve EXACT user-facing text output, JSON shapes, error codes/messages, and the stable error 'code' strings (file_not_found, parse_error, timestamp_conflict, timestamp_overlap, etc.).
This is a shared package written by several agents in parallel: create ONLY your own ${g.files.map((f) => f + '.go').join('/')} files; do NOT edit peers' files or any shared file.
Syntax self-check each file with 'gofmt -e <file>.go' (the whole package is built in the final phase, so a full package build is expected to fail now if a peer file is missing — that's fine).`,
    { label: g.label, phase: 'Commands & TUI', schema: PORT_SCHEMA },
  ))

const cmdTuiResults = await parallel([tuiAgent, ...cmdThunks])
cmdTuiResults.filter(Boolean).forEach((r) => log(`${r.pkg}: files=${(r.filesWritten || []).length} ${r.notes ? '— ' + r.notes : ''}`))

// =================== Phase 5: Wiring (main.go) — first full build ===================
phase('Wiring')
const wiring = await agent(
  `${baseRules}

MODULE: program entry point + urfave/cli command tree.
Read & port 1-to-1:
- ${REPO}/src/bin/cli/cli.rs   (the clap command/flag/alias structure)
- ${REPO}/src/bin/cli/main.rs  (parse, init logging, dispatch to handlers, error handling, exit codes)
Write: ${REPO}/cmd/sm/main.go (package main).

Build the urfave/cli/v3 command tree mirroring cli.rs EXACTLY: command names, every flag (long names, value_name, defaults), the value-enums (--output text|json global, --format csv|anchored, --format anchored), the 'ts' alias for translation-status, allow-hyphen on delay OFFSET, the top-level --log-level/--log-target/--output flags, version, and the about text. Each command's Action calls the matching cmd.Handle* from package cmd (import ${REPO.replace('/Users/eugene/pro', '')}... use module path github.com/lebe-dev/submarine/internal/cli/cmd and .../internal/cli). Reproduce main.rs behavior: init slog logging from flags, dispatch, on handler error format/emit via cli.OutputError and os.Exit(1); when no command is given, emit the same no_command error and exit(1).

This is the FIRST full-tree build. Run 'cd ${REPO} && go build ./... ' and fix compile errors across the wiring AND reconcile any signature mismatches you introduced. Read internal/cli/*.go and internal/cli/cmd/*.go for the exact handler signatures. Return buildOk=true only if 'go build ./...' exits 0.`,
  { label: 'port:wiring', phase: 'Wiring', schema: PORT_SCHEMA },
)
log(`wiring done. buildOk=${wiring?.buildOk}. notes: ${wiring?.notes ?? ''}`)

// =================== Phase 6: Integration tests ===================
phase('Tests')
const testsPort = await agent(
  `${baseRules}

MODULE: integration tests. Port these Rust integration tests 1-to-1 into Go test files under ${REPO}/internal/cli/cmd/ (package cmd, files named <area>_integration_test.go):
- ${REPO}/tests/subtitle_integration_test.rs
- ${REPO}/tests/get_command_integration_test.rs
- ${REPO}/tests/add_command_integration_test.rs
- ${REPO}/tests/import_command_integration_test.rs
- ${REPO}/tests/translation_status_command_integration_test.rs

Use t.TempDir() instead of the tempfile crate; read fixtures from ${REPO}/test-data/ (paths relative to repo root — compute an absolute path or copy fixtures into the temp dir as the Rust tests do). Assert the same observable behavior (returned data, written file contents, error kinds). Call the ported handlers / services directly (package cmd can import internal/cli and the domains). Do NOT shell out to the binary unless the Rust test does.
Self-check: 'cd ${REPO} && go build ./... && go test ./internal/cli/cmd/ -run Integration' — fix your test files until they compile and pass (fix ONLY test files here; real-code bugs are handled in the next phase, but note them).`,
  { label: 'port:integration-tests', phase: 'Tests', schema: PORT_SCHEMA },
)
log(`integration tests ported. buildOk=${testsPort?.buildOk}. notes: ${testsPort?.notes ?? ''}`)

// =================== Phase 7: Build & fix loop ===================
phase('Build & fix')
let prevRemaining = null
let green = false
for (let i = 0; i < 12; i++) {
  const r = await agent(
    `${baseRules}

You are the reconciliation/repair pass for the Go port (iteration ${i + 1}).
Run, in order, from ${REPO}:
  1) go build ./...
  2) go vet ./...
  3) go test ./...
Fix as MANY errors as you can this iteration by editing the Go source/test files (follow ${SPEC}; keep the port faithful — fix real bugs, do not delete tests to make them pass, do not weaken assertions). Typical issues: cross-package signature mismatches between independently-written packages, wrong JSON tags, missing methods, import cycles, unused imports, slog API misuse, time.Duration/ms conversions, bubbletea API mismatches.
If 'go build' reports a missing module or go.sum entry, run 'go mod tidy' with dangerouslyDisableSandbox=true, then continue.
After fixing, RE-RUN all three and report the final state of THIS iteration. Set progressed=true if you reduced the number of errors compared to the start of this iteration.
${prevRemaining ? `Errors still outstanding from the previous iteration:\n${prevRemaining}` : ''}`,
    { label: `build-fix#${i + 1}`, phase: 'Build & fix', schema: BUILD_SCHEMA },
  )
  if (!r) { log(`iteration ${i + 1}: agent returned null, retrying`); continue }
  log(`iter ${i + 1}: build=${r.buildOk} vet=${r.vetOk} test=${r.testOk} progressed=${r.progressed}`)
  if (r.buildOk && r.vetOk && r.testOk) { green = true; break }
  if (!r.progressed && r.remaining && r.remaining === prevRemaining) {
    log(`no progress two iterations running — stopping the loop with remaining issues`)
    prevRemaining = r.remaining
    break
  }
  prevRemaining = r.remaining
}

return {
  green,
  remaining: prevRemaining || '',
  summary: green
    ? 'Go port builds, vets, and tests clean (go build/vet/test ./... all green).'
    : 'Port complete but not fully green — see remaining.',
}
