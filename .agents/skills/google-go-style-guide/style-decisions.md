# Go Style Decisions

Normative style decisions for Go. See [references.md](references.md) for the
official source.

If this file conflicts with [guide.md](guide.md), follow the Style Guide.

---

## Naming

### Underscores

- Do **not** use underscores in Go names, except:
  1. Package names imported only by generated code
  2. Test, benchmark, or example function names in `*_test.go`
  3. Low-level OS or cgo interop (rare)
- Filenames may contain underscores. Filenames are not Go identifiers.

### Package names

- Use **lowercase only**. Do not use underscores (`tabwriter`, not `tab_writer`).
- Avoid names that shadow common locals (`usercount` is better than `count`).
- **Rename at import** if a dependency uses underscores. The local name must
  follow Go naming.
- Avoid vague names: `util`, `utility`, `common`, `helper`, `model`,
  `testhelper`. Name the package by what it provides.

### Receiver names

- Keep receivers **short** (one or two letters).
- Use an **abbreviation of the type**.
- Keep the name **consistent** for that type.
- Prefer `(t Tray)`, `(ri *ResearchInfo)`, `(w *ReportWriter)`, `(s *Scanner)`.
- Do not use `(tray Tray)`, `(this ...)`, or `(self ...)`.

### Constant names

- Use **MixedCaps**. Exported names start with uppercase. Unexported names start
  with lowercase.
- Do not use `MAX_PACKET_SIZE`. Do not use a `k` prefix.
- Name by **role**, not by value. Bad: `const Twelve = 12`.

### Initialisms

- Keep the same case inside an initialism: `URL` or `url`, never `Url`.
- Exported: `XMLAPI`, `ID`, `DB`, `Txn`.
- Unexported: `xmlAPI`, `id`, `db`.
- Keep odd standard casing: `iOS` → exported `IOS`; `gRPC` → exported `GRPC`;
  `DDoS` → exported `DDoS`, unexported `ddos`.

### Getters

- Do **not** use a `Get` or `get` prefix unless the concept is “get” (for
  example HTTP GET).
- Use a noun: `Counts`, not `GetCounts`.
- For heavy work, use `Compute`, `Fetch`, or a similar verb.

### Variable names

- Name length grows with **scope** and shrinks with frequent **use**.
- Prefer **what it holds / how it is used**, not where it came from.
- Omit type words when obvious: `userCount` not `numUsers`; `users` not
  `userSlice`.
- Add a type qualifier only when two forms exist (for example `ageString` vs
  `age`).
- Single-letter names are OK for receivers, `r`/`w` for Reader/Writer types,
  loop indices, and short scope.
- Do not abbreviate by dropping letters in exported names (`Sandbox`, not
  `Sbx`).

### Repetition

- **Package vs symbol**: Do not repeat the package in exported names. Prefer
  `widget.New`, `db.Load`. For one main type, use `New`.
- **Variable vs type**: Omit the type from the name unless two values of the
  same type are in scope.
- **Context**: Omit names that repeat package, type, method, or file context.

---

## Commentary

- Prefer comments readable near 80 columns. Wrap into multiple single-line
  comments. No fixed limit. Break on punctuation when it helps.
- All **top-level exported** names need doc comments. Use full sentences that
  start with the name (an article is OK).
- Document unexported names when behavior is not obvious.
- Complete sentences use capital letters and punctuation. Field fragments need
  not.
- One **package comment** per package, directly above `package` with no blank
  line. For `main`, use the binary name.
- Prefer runnable examples in test files. They appear in godoc.

### Named result parameters

- Use named results when they **clarify** (same type returned more than once, or
  the caller must act, such as calling a cancel func).
- Do not use named results only to avoid a variable or to enable naked returns
  in large functions.
- Prefer clarity over brevity.

---

## Imports

- **Rename** only when needed (collision, or proto names with underscores).
- For proto imports, remove underscores and add a `pb` suffix (for example
  `foopb`).
- Keep the same local name for the same import across files.
- On collision, rename the most local or project-specific import.
- **Group** imports in this order:
  1. Standard library
  2. Other project or vendored packages
  3. Protobuf
  4. Side-effect imports (`_ "pkg"`)
- Use blank imports only in `main` or tests that need them. Avoid them in
  library packages. Exceptions: nogo bypass, `//go:embed` with `embed`.
- Do **not** use dot imports (`. "foo"`). Keep qualification so origin is clear.

---

## Errors

- Return `error` as the last result. Success means a `nil` error.
- Callers must not rely on other return values when error is non-nil, unless
  docs say otherwise.
- Exported APIs return `error`, not concrete error types (avoids nil interface
  bugs).
- Error strings: lowercase, no trailing punctuation. Capitalize only for
  exported names, proper nouns, or acronyms. Log or UI text may use capitals.
- Handle errors. Do not discard with `_` unless docs allow it. Handle, return,
  or in rare cases use `log.Fatal` / `panic`. Comment when you ignore an error.
- Prefer `(value T, ok bool)` or `(value T, err error)` over in-band sentinels
  such as `-1`, `nil`, or `""`.
- Handle the error path first. Prefer early return. Avoid `else` for the normal
  path.

---

## Language

### Literal formatting

- For struct literals from other packages, always use **field names**.
- For package-local types, field names are optional. Use them for clarity or
  many fields.
- Keep the closing brace at the same indent as the opening brace.
- End multi-line literals with a comma, then the closing brace on the next line.
- Use cuddled braces `}}{{` only when indentation matches and inner values are
  literals or proto builders, not variables.
- Omit repeated types in slice or map literals when obvious. Use `gofmt -s` for
  struct literals.
- Omit zero-value fields when clarity stays intact. In table-driven tests, omit
  zero fields that are not relevant to the case.

### Nil slices

- Prefer `var s []T` over `s := []T{}` for locals that may be returned.
- Do not design APIs that treat nil and empty slices as different.
- Use `len(s) == 0` for “empty.”

### Formatting and line breaks

- Avoid line breaks that align the next line with an indented block. If you
  cannot avoid it, add a space.
- Keep function signatures on one line when possible. Shorten with locals or
  option structs.
- Do not break calls only for line length.
- Do not break `if` / `for` / `switch` so the condition aligns with the body.
  Extract booleans or locals.
- Put the variable on the left: `if result == "foo"`.
- Keep `switch` / `case` on one line when possible. If long, indent cases and
  separate with a blank line.

### Copying

- Do not copy types that must not be copied (`sync.Mutex`, `bytes.Buffer`, or
  structs that contain them).
- Prefer pointer receivers and pointer params for those types.

### Control flow and APIs

- Do **not** panic for normal errors. Return `error`.
- In `main` or init paths, consider `log.Exit` for config errors.
- Use `panic` only for impossible bugs.
- Name Must helpers `MustXYZ` or `mustXYZ`. Use them for init or test helpers
  that call `t.Fatal`. Do not use them for user input.
- Make goroutine exit clear. Avoid leaks. Use `context.Context`,
  `sync.WaitGroup`, or explicit signals. Document when goroutines stop.
- Put **interfaces** in the **consumer** package. Producers return concrete
  types. Do not export test doubles. Do not define interfaces before use.
- Use **generics** when they meet requirements. Avoid premature use. Prefer
  interfaces when they suffice.
- Do not pass pointers only to save a few bytes. Pass pointers for large
  structs, protos, or mutation.
- Choose value or pointer receivers by correctness. Use pointer if you mutate,
  hold sync types, or hold a large value. When unsure, use pointer. Prefer one
  style of receiver methods per type.
- Do not use a redundant `break` at the end of a case. Use a labeled `break` to
  leave an enclosing `for` from inside a `switch`.
- Prefer synchronous APIs. Let the caller add concurrency.
- Use `type T1 = T2` only to refer to an existing type (for example migration).
  Prefer `type T1 T2` for new types.
- Use `%q` for strings when empty or control characters matter.
- Prefer `any` over `interface{}` in new code.

---

## Common libraries

- Define **flags** only in `main`. Flag names use snake_case. Variables use
  MixedCaps. Libraries expose Go APIs, not flags.
- Use the project log package. Prefer `log.Exit` without a stack when that fits.
  Use `log.Fatal` when a stack is needed.
- Put `context.Context` first: `func F(ctx context.Context, ...)`.
- Do not store context in structs. Pass it as an argument.
- Use `context.Background()` (or `tb.Context()` in tests) only in entrypoints.
- Do not invent custom context types.
- Use `crypto/rand` for keys and secrets. Do not use `math/rand` for secrets.

---

## Useful test failures

- Do **not** use assertion libraries. Use Go `testing`. Use `cmp` for equality
  and diffs.
- Failure messages must allow diagnosis without reading the test source.
- Include the function name and inputs. Put **got before want**.
- For large output, use diffs and state direction (for example `(-want +got)`).
- Prefer a full expected value with `cmp.Equal` / `cmp.Diff` over hand checks.
- For protos, use `protocmp.Transform()`.
- Do not compare unstable output such as raw JSON bytes. Compare semantic data.
- Prefer `t.Error` over `t.Fatal` for multiple checks. Use `t.Fatal` when later
  checks are meaningless.
- Do not use error string text for control flow. Use `errors.Is` or `cmp` with
  `cmpopts.EquateErrors`.

---

## Test structure

- Use **subtests** for tables, setup or teardown, and parallelism.
- Keep subtest names readable. Do not use slashes.
- Do not depend on other subtests’ state.
- Use one table when logic is the same. Identify the row by name or inputs, not
  by index.
- Pass `*testing.T` into helpers (after ctx if present). Call `t.Helper()`.
- Use `package foo` for white-box tests. Use `package foo_test` for black-box
  tests or to break import cycles.
- Use only the standard `testing` package. Do not add third-party test
  frameworks.

---

## Non-decisions

No prescribed choice for:

- `var i int` vs `i := 0`
- `&File{}` vs `new(File)`
- `got` / `want` order in `cmp.Diff`
- `errors.New("foo")` vs `fmt.Errorf("foo")`

Be consistent locally.
