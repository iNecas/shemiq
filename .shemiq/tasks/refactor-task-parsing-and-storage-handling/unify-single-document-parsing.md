# Unify single-document parsing
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 59d0fd91-5c0b-4cf7-bfef-c93ec52113ed
:::

## Context

Introduce the private, section-based parser described by the parent task. The parser extracts structure and precise source locations from one Markdown document; it does not interpret task semantics or access the filesystem.

The current code has two overlapping parsing paths: `internal/task/metadata.go` scans directives and keeps repair offsets, while `internal/task/outline.go` rescans headings to identify task entries. Neither excludes fenced examples. This task provides their eventual replacement, but deliberately does **not** migrate either path yet. The scoped-store and validation subtasks will integrate the new parser and remove the legacy paths later.

## Interview

Approved decisions:

- Introduce the parser independently, with focused tests. Leave existing callers and legacy parsing unchanged until later subtasks migrate them.
- Recognize conventional ATX headings and backtick/tilde fences within the limited Markdown subset below; do not introduce a full Markdown parser.
- Return a synthetic root and nested sections, with directives attached structurally and fields preserved in source order.
- Preserve byte spans, diagnostic lines, and insertion information needed by future repairs.
- Syntax errors are fatal and return no document; semantic interpretation remains outside the parser.
- Retain the existing strict directive-field grammar, including rejecting blank lines inside directives.

## Implementation plan

### Scope and private API

- Add `internal/task/parse.go` in the existing package. Use a private entry point such as `parseMarkdownDocument(path string, data []byte) (*parsedDocument, error)` and distinct private types so the new parser can coexist with the existing `parseDocument`, `document`, and `directive` definitions.
- The path is diagnostic context only. Do not read files, resolve references, discover projects, normalize filesystem paths, or create tasks here.
- Return the first syntax error with its path and one-based line number, and no parsed document. Do not expose partially parsed results.
- Do not migrate commands, refinement, validation, or repair code in this task. Temporary coexistence is intentional, not a permanent parallel API.

### Document and source model

- Retain the original bytes and a synthetic root section covering the document. Treat those bytes as unchanged source data; do not normalize line endings before calculating offsets.
- Each heading section records its heading level, title, heading position/span, overall section span, directly contained directives, and child sections. A section runs from its heading to the next heading of equal or lower level, or EOF, including its descendants. Skipped heading levels do not create synthetic intermediate sections.
- Attach each directive to the innermost current section. Directives before any heading belong to the root. Headingless and status-only directives must remain accessible by traversing the tree.
- Record structural containment only. The future store determines heading adjacency, primary tasks, and entries under `## Tasks`; the parser must retain enough heading/directive boundary information to make those decisions from the original bytes, including checking intervening blank lines.
- Use zero-based, half-open byte spans and one-based diagnostic lines. Retain directive boundaries, opening/closing positions, the insertion offset immediately before the closing line, and the preferred line ending for inserted fields.
- Preserve the existing insertion-line-ending behavior: prefer the line ending immediately before the closer, with the existing document-style/LF fallback. A closer at EOF without a final newline is valid.
- Preserve every field occurrence in source order, including its key, trimmed value, located key/field position, and exact value span. Duplicate keys must not overwrite earlier occurrences. Empty values need a valid zero-length value span suitable for a later targeted replacement.
- Unknown keys, duplicates, empty values, invalid types/statuses, missing UUIDs, and invalid UUIDs are not syntax errors. Do not produce semantic `Issue`s or impose required fields here.

### Markdown recognition

- Recognize only column-zero ATX headings at levels 1–6. Require whitespace after the opening hashes unless the heading is empty. Strip surrounding title whitespace and conventional optional closing hashes; retain inline Markdown formatting as literal title text.
- Ignore backtick and tilde fenced blocks. Recognize conventional fence openers with at least three matching fence characters and up to three leading spaces. A closer must use the same character, be at least as long as the opener, and follow conventional closing-fence rules. Content within fences cannot create headings, directives, or Shemiq syntax errors.
- An unclosed fence extends through EOF and is not a Shemiq syntax error.
- Indented headings/directives and Setext headings are ordinary text. Do not interpret inline formatting, lists, blockquotes, or other Markdown structures beyond recognizing the agreed headings and fences.
- Recognize fences only outside directives; a fence-looking line inside a directive cannot hide malformed field syntax.

### Directive syntax and errors

- Outside fences, an opener is exactly column-zero `:::shemiq`. Any other column-zero line beginning with that prefix is a malformed opener.
- A directive closer is exactly column-zero `:::`. A stray closer outside a directive is ordinary text.
- Within a directive, require one `key: value` field per line. Retain the current key grammar: an ASCII letter followed by ASCII letters, digits, `_`, or `-`. Split at the first colon; empty values and additional colons in values are allowed.
- Blank lines inside directives, malformed fields, nested directives, malformed openers, and unterminated directives are fatal syntax errors. Unterminated-directive diagnostics identify the opening line; other errors identify the offending line.
- An empty directive is syntactically valid; requirements concerning its metadata belong to later semantic interpretation.

### Minimal verification

Add a small set of focused parser tests, preferably table-driven where useful:

- Section nesting, skipped levels, section boundaries, directive ownership, and root-level metadata; retain duplicate/unknown fields and empty values without semantic findings.
- ATX title handling and exclusion of backtick/tilde fenced examples, including indented fences and a fence extending to EOF; indented examples remain text.
- Fatal syntax errors with located diagnostics and no partial document.
- Byte and value offsets against original source, including LF, CRLF, mixed line endings, and a closing line at EOF. Simulate a targeted metadata edit using the retained positions and verify that unrelated bytes are unchanged.

Run `go test ./...` without migrating or changing existing command tests. The new parser is not yet reachable through commands, so focused parser tests are the appropriate coverage at this stage. Later integration work will exercise it through existing end-to-end refinement and repair tests.

### Handoff to sibling tasks

- **Introduce the scoped store and task model:** consume this tree, recognize heading-adjacent task definitions, and interpret metadata into tasks and located conversion issues. Migrate refinement and retire the legacy outline parser there.
- **Consolidate validation and repairs:** traverse all directives, including root/metadata-only directives; use preserved field occurrences and offsets for semantic findings and targeted edits. Migrate validation and retire legacy directive parsing there.
- Keep both sibling tasks in place. Do not implement store loading, semantic conversion, cross-document checks, or production repair orchestration as part of this task.

## Implementation notes

- Added private `parseMarkdownDocument` in `internal/task/parse.go`. It consumes supplied bytes only, retains the diagnostic path unchanged, and returns no document on the first syntax error. Existing parsing paths and callers are unchanged.
- Added a synthetic root and nested `parsedSection` tree. Sections retain complete heading lines, descendant-inclusive spans, directly owned directives, and children without synthetic intermediate heading levels. Original bytes and heading/directive boundaries remain available for future adjacency checks.
- `sourceSpan` records zero-based, half-open byte offsets and one-based diagnostic lines. Directives retain opening/closing spans, insertion offsets, and preferred line endings. Ordered `parsedField` occurrences retain full-line, key, and trimmed-value spans, including zero-length empty values; duplicates and invalid/unknown metadata remain uninterpreted.
- Recognizes the agreed column-zero ATX headings and conventional backtick/tilde fences. Fenced examples and indented Markdown examples cannot create metadata; unclosed fences extend to EOF. Directive fields retain the strict existing grammar, including fatal blank lines and fence-looking malformed fields.
- Added four focused parser tests covering section ownership/boundaries, heading/fence recognition, fatal located diagnostics, and original-byte offsets. The edit simulation replaces a value and inserts a field, verifies unrelated bytes remain unchanged, and reparses the result across LF, CRLF, mixed endings, empty values, and an EOF closer.
- Verification: `go test ./...` and `go vet ./...` passed. No command tests were changed.
- Handoff: the scoped store can consume `parsedDocument.root` for task conversion and heading adjacency, then retire the legacy outline parser. Validation can traverse every directive and use ordered fields/spans for findings and targeted edits. The new parser reuses the existing syntax-only `metadataKeyPattern` and `lineEnding` helpers in `metadata.go`; retain or relocate these helpers when retiring legacy directive parsing. Store loading, semantics, and production repair orchestration remain for the sibling tasks.
