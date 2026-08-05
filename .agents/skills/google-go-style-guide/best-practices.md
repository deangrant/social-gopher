# Go Style Best Practices

Guidance for common Go situations. These patterns are not canonical.
See [references.md](references.md) for the official source.

Use judgment when more than one approach fits. Prefer clarity and explicit
dependencies.

---

## Naming

### Avoid repetition at the call site

- **Package**: Do not repeat the package name in function names. Prefer
  `yamlconfig.Parse(input)` over `yamlconfig.ParseYAMLConfig(input)`.
- **Receiver**: Do not repeat the receiver type. Prefer `(c *Config).WriteTo(w)`
  over `WriteConfigTo(w)`.
- **Parameters**: Do not repeat parameter names. Prefer
  `Override(dest, source *Config)` over
  `OverrideFirstWithSecond(dest, source *Config)`.
- **Return types**: Do not repeat the return type in the name. Prefer
  `Transform(input *Config)` over `TransformToJSON(input *Config)`.
- Add disambiguating words only when needed (for example `WriteTextTo` vs
  `WriteBinaryTo`).

### Function and method naming

- **Returning a value**: Use a noun-like name. Avoid a `Get` prefix.
  - Good: `(c *Config).JobName(key string) (value string, ok bool)`
  - Bad: `(c *Config).GetJobName(key string) (value string, ok bool)`
- **Doing work**: Use a verb-like name, for example
  `(c *Config).WriteDetail(w io.Writer)`.
- **Type-specific variants**: Include the type when more than one version
  exists: `ParseInt`, `ParseInt64`. Omit the type for the primary version:
  `Marshal()` vs `MarshalText()`.

### Test double and helper packages

- Name test helper packages by appending `test` (for example `creditcardtest`
  for `creditcard`).
- For one type double, use a short name: `Stub` in `creditcardtest`.
- Name behavior-specific doubles by behavior: `AlwaysCharges`,
  `AlwaysDeclines`.
- For multiple types, use explicit names: `StubService`, `StubStoredValue`.
- In tests, prefix variables when doubles sit next to production types (for
  example `spyCC`).

### Shadowing vs stomping

- **Stomping** (same variable reused with `:=`) is OK when the old value is no
  longer needed (for example a new timed `ctx`).
- **Shadowing** (new variable in an inner scope) can hide bugs. For a
  conditional deadline, assign with `=` in the outer scope so the capped `ctx`
  remains in use after the block.
- Do not shadow standard package names (for example `url := "..."` hides
  `net/url`).

### Util packages

- Do not use vague names such as `util`, `helper`, or `common`.
- Name packages by what they provide so call sites stay clear.

### Package size and layout

- Group related types that interact in the same package.
- If users usually need two types together, keep them in one package.
- Prefer focused packages. Split when concepts are distinct.
- Avoid one huge file and many tiny files. Group by theme (for example
  `reader.go` / `writer.go`).
- Optional: use `doc.go` for package docs only.

---

## Imports

- For proto or gRPC, use descriptive renamed imports: `foopb`, `foogrpc`. Avoid
  very short names such as `xpb`.
- Follow the import grouping in [style-decisions.md](style-decisions.md).

---

## Error handling

### Structure

- Use **structured errors** when callers must tell cases apart: sentinels
  (`var ErrDuplicate = errors.New("duplicate")`) or types with fields.
- Use `errors.Is` / `errors.As` for wrapped errors.
- Never match errors by string text.

### Adding context

- Add **non-redundant** context.
- Good: `fmt.Errorf("launch codes unavailable: %v", err)`
- Bad: `fmt.Errorf("could not open settings.txt: %v", err)` when the path is
  already in the error.
- Do not wrap only to say “it failed.” Return the error as-is when you add no
  meaning.

### %v vs %w

- Use **`%v`** for human messages or when callers do not inspect the chain.
- Use **`%w`** when callers should use `errors.Is` / `errors.As`.
- At RPC or storage boundaries, often translate to status codes instead of
  wrapping with `%w`.
- Put `%w` at the **end** of the format string:
  `fmt.Errorf("context: %w", err)`.

### Logging errors

- Prefer not to log and also return the same error. Let the caller log.
- Avoid duplicate log lines.
- Be careful with personal data in logs.
- Use expensive error-level logging sparingly. Prefer actionable messages.

### Program initialization and panics

- Propagate init and config errors to `main`.
- In `main`, call `log.Exit` with an actionable message.
- Prefer returned errors over panics.
- Use `log.Fatal` for invariant violations that make state unrecoverable.
- Do not recover panics to hide crashes. Fix the bug.
- Panic is acceptable for:
  1. API misuse (as in `reflect`)
  2. A package-internal panic turned into an error at the API boundary
  3. Unreachable code after `log.Fatalf`
- Never expose panic as part of a public API.

---

## Documentation

- Document error-prone or non-obvious parameters and fields. Explain why they
  matter. Do not document every parameter by habit.
- For context, you need not restate that cancel yields `ctx.Err()`. Document
  special context behavior.
- Assume read-only operations are safe for concurrent use. Assume mutating
  operations are not. Document unclear cases.
- Document cleanup duties (for example “Call Stop” or “Close resp.Body”).
- Document important sentinels and error types, and whether they are pointer or
  value types.
- Use runnable examples. Preview godoc. Separate paragraphs with blank lines.
  Indent verbatim code by two spaces.

---

## Variable declarations

- Prefer `:=` for new variables with non-zero values.
- Use `var` when the zero value is intentional.
- Use composite literals when you have initial elements.
- Avoid noisy empty literals when the zero value is enough.
- Use size hints with `make` when size is known and it matters. Do not
  over-preallocate.
- Specify channel direction when possible:
  `func sum(values <-chan int)`.

---

## Function argument lists

- Keep signatures short.
- For many arguments, use an **option struct** or **variadic options**.
- Pass an option struct as the last argument. Do not put `context.Context` in
  the struct.
- Prefer an option struct when many callers set several options.
- Prefer variadic options when most callers use defaults.
- Process variadic options in order. The last option wins on conflict.

---

## Tests

- Fail in the `Test` function. Avoid assertion helpers that take `*testing.T`
  and fail inside.
- Validation helpers may **return** an error or a diff. The test decides how to
  fail.
- Setup helpers use `t.Helper()`. On setup failure, `t.Fatal` is appropriate.
- Use `t.Fatal` for setup that applies to the whole test. For one table row,
  use `t.Error` and `continue`, or `t.Fatal` inside `t.Run`.
- Only the test goroutine may call `t.Fatal` / `t.FailNow`. Other goroutines use
  `t.Error` and return.
- Use field names in table struct literals.
- Prefer setup only in tests that need it. Avoid package-level init for every
  test.
- Use `sync.Once` for expensive shared setup without teardown.
- Use `TestMain` only when shared setup **and** teardown are required.
- For acceptance tests, provide a validation function that returns an error.
- Prefer real transports for integration tests over hand-rolled client mocks.

---

## String concatenation

- Simple cases: use `+`.
- Formatted cases: use `fmt.Sprintf`.
- If the target is an `io.Writer`, use `fmt.Fprintf`.
- Piecemeal building: use `strings.Builder`.
- Constant multi-line text: use backticks.

---

## Global state

- Avoid package-level state that controls behavior for all clients.
- Prefer **instance-based APIs**. Construct a type and pass it as a dependency.
- Globals cause test-order issues, hard isolation, and unclear lifecycle.
- If you offer a global convenience API:
  1. Provide an instance-based API as the primary API
  2. Make the global API a thin wrapper
  3. Document use and test reset rules
- Libraries must not rely on package-level state of dependencies.
- Safe globals are logically constant, effectively stateless to callers, or have
  no observable effect outside the process.

---

## CLI and options

- Prefer a subcommand library when the binary has subcommands. `cobra` is common
  outside Google.
- With cobra, use `cmd.Context()`, not `context.Background()`.
- Keep CLI code apart from library code. The binary is one consumer of the
  library.

---

## Closing rule

When unsure, prefer clarity and explicit dependencies over convenience and
global state.
