# Notes for agents

## Code organization

Order functions from high-level entry points to lower-level helpers and implementation details.

## Coding Tips

When writring multi-line strings, use `utils.Dedent` from "github.com/iNecas/shemiq/internal/utils"

Avoid long lines, try to keep under 100, ideally uner 80 characters. But don't go the other
extreme and break short lines that would be short without.

Good:

```go
writeTestMarkdown(t,
    filepath.Join(shemiq, "archive", "old-file-path", "top-level.md"),
    withUUID)
directives = append(directives, allDirectives(child)...)
```

Bad:

```go
writeTestMarkdown(t, filepath.Join(shemiq, "archive", "old-file-path", "top-level.md"), withUUID)
directives = append(
               directives, allDirectives(child)...)
```
