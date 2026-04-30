---
name: Add AllCaps Removal Rule
overview: Add a new configurable subtitle rule `RemoveLineIfAllCaps` that removes individual all-caps lines (not whole cues), wired through config, transform pipeline, and tests without changing default behavior compatibility.
todos:
  - id: config-wireup
    content: Add RemoveLineIfAllCaps field, defaults, description, and abbreviated label in rules config
    status: completed
  - id: transform-logic
    content: Implement all-caps line predicate and ApplyAll integration for line-level removals
    status: completed
  - id: tests-config
    content: Extend config tests for default, parse, and effective description coverage
    status: completed
  - id: tests-transform
    content: Add transform tests for predicate behavior and end-to-end line removal semantics
    status: completed
  - id: verify
    content: Run targeted Go tests and lint checks for touched files
    status: completed
isProject: false
---

# Add `RemoveLineIfAllCaps` Rule

## Goal
Introduce a new boolean rule that removes subtitle lines when the full line is uppercase text, while preserving existing behavior and rule ordering conventions.

## Scope and behavior
- Remove only matching **line(s)** within a cue.
- Do **not** remove entire cue unless all lines become empty after filtering (existing pipeline behavior).
- Strict all-caps behavior: remove any all-caps line, including single-word acronyms (e.g., `FBI`).
- "All-caps" detection rule:
  - line must contain at least one letter;
  - every letter in the line must be uppercase;
  - digits/punctuation/symbols/whitespace are ignored for casing checks.

## Planned file changes
- Update [`K:/Develop/subtitle-sanitizer/internal/rules/config.go`](K:/Develop/subtitle-sanitizer/internal/rules/config.go)
  - Add `RemoveLineIfAllCaps bool` to `Config` with JSON key `removeLineIfAllCaps`.
  - Add default value in `DefaultConfig()` (set to `false` for backward compatibility).
  - Include this flag in `DescribeEffective()` output list.
  - Add a new abbreviated rule constant in `AbbreviatedRuleDescription` (e.g., `RuleRemoveLineIfAllCaps`).

- Update [`K:/Develop/subtitle-sanitizer/internal/transform/transform.go`](K:/Develop/subtitle-sanitizer/internal/transform/transform.go)
  - Add helper predicate for all-caps line detection using Unicode letter checks.
  - In `ApplyAll`, add a conditional block for `conf.RemoveLineIfAllCaps` that:
    - splits cue text by lines,
    - removes lines matching all-caps predicate,
    - rejoins non-empty remaining lines,
    - marks rule as applied when at least one line is removed.
  - Place the new block with other line-removal rules (before delimiter cleanup) to keep semantics consistent.

- Update tests in [`K:/Develop/subtitle-sanitizer/internal/rules/config_test.go`](K:/Develop/subtitle-sanitizer/internal/rules/config_test.go)
  - Validate default config value.
  - Validate JSON parsing for `removeLineIfAllCaps`.
  - Validate `DescribeEffective()` includes the new rule when enabled.

- Update tests in [`K:/Develop/subtitle-sanitizer/internal/transform/transform_test.go`](K:/Develop/subtitle-sanitizer/internal/transform/transform_test.go)
  - Add focused tests for all-caps predicate edge cases (letters + punctuation/numbers).
  - Add `ApplyAll` tests for:
    - all-caps line removed,
    - mixed-case line preserved,
    - multiline cue where only one line is removed,
    - cue fully emptied by removals and dropped,
    - applied-rules log includes new abbreviation.

## Verification plan
- Run targeted Go tests for modified packages:
  - `go test ./internal/rules ./internal/transform`
- If needed, run broader sanitization flow tests:
  - `go test ./internal/sanitize ./internal/wasmbridge`
- Confirm no lints introduced in touched files.

## Compatibility and risk handling
- Keep default disabled (`false`) to avoid behavior change for existing users.
- Reuse existing line-splitting and cue cleanup patterns from current rules to minimize regressions.
- Avoid changing rule precedence outside inserting this new optional check.