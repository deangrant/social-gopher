# Google Go Style Guide

This file summarizes the Style Guide. It is normative and canonical.
See [references.md](references.md) for the official link.

Apply with [style-decisions.md](style-decisions.md) and
[best-practices.md](best-practices.md).

---

## Style principles (order of importance)

1. **Clarity** — Purpose and reason are clear to the reader.
2. **Simplicity** — Use the simplest way that meets the goal.
3. **Concision** — Keep a high signal-to-noise ratio.
4. **Maintainability** — Make correct change easy over time.
5. **Consistency** — Match the rest of the codebase when other principles do not decide.

Consistency does **not** override the other principles. Use it to break ties.

---

## Clarity

- Optimize for the **reader**, not the author.
- Make code easy to **read** more than easy to write.
- Build clarity with **naming**, **commentary**, and **organization**.
- Answer two questions:
  - **What** does the code do?
  - **Why** does it do that?
- Prefer **why** in comments. Avoid comments that only restate the code.
- Unusual code should signal “needs attention” (for example performance). Keep
  ordinary code uniform so special code stands out.

---

## Simplicity

- Write the **simplest** code that meets behavior and performance goals.
- Simple code:
  - Reads top to bottom with ease
  - Does not require memorized context
  - Avoids needless abstraction and noisy names
  - Makes control flow and data flow obvious
  - Uses comments for **why**, not what
  - Has clear docs and useful errors or test failures
  - Is usually **not** clever
- Complexity is allowed when justified (performance, many clients). Then
  **document** it and add tests or examples.
- If the goal is simple but the code is complex, find a simpler form.

### Least mechanism

Prefer the smallest standard mechanism:

1. **Core language** (channels, slices, maps, loops, structs) when enough
2. **Standard library** when needed
3. **Project libraries** before new dependencies or custom machinery

- In tests, override flag-bound variables directly when possible. Prefer that
  over `flag.Set`, unless you test the CLI with `os/exec`.
- For set membership, `map[K]bool` is usually enough. Use a set library only
  when you need more behavior.

---

## Concision

- Keep high signal-to-noise. Naming and structure should show what matters.
- Reduce repetitive code, extra syntax, opaque names, needless abstraction, and
  noisy whitespace.
- Use **table-driven tests** to factor repetition and keep differences visible.
- Use **common idioms** so readers can scan quickly. If you deviate (for
  example `err == nil` instead of `err != nil`), add a short comment:

```go
// Good: reader sees this is "if NO error"
if err := doSomething(); err == nil { // if NO error
    // ...
}
```

---

## Maintainability

- Code is **edited** more than it is written. Clarity enables safe change.
- Maintainable code:
  - Is easy to modify correctly
  - Has APIs that can grow without breaks
  - Makes assumptions explicit
  - Maps abstractions to the **problem**, not to code shape
  - Avoids needless coupling and unused features
  - Has tests with clear, actionable failures
- **Interfaces** remove context. Use them when the benefit is clear. Document
  contracts and non-obvious usage.
- Do **not** hide critical details in easy-to-miss places (for example `=` vs
  `:=`, or a single `!` in a long condition). Prefer explicit steps or a comment.

```go
// Bad: = vs := is easy to miss
if user, err = db.UserByID(userID); err != nil { ... }

// Good: explicit and clear
u, err := db.UserByID(userID)
if err != nil {
    return fmt.Errorf("invalid origin user: %s", err)
}
user = u
```

```go
// Bad: the ! is easy to miss
leap := (year%4 == 0) && (!(year%100 == 0) || (year%400 == 0))

// Good: named parts and a comment
// Gregorian leap years are not only year%4 == 0.
var (
    leap4   = year%4 == 0
    leap100 = year%100 == 0
    leap400 = year%400 == 0
)
leap := leap4 && (!leap100 || leap400)
```

- Use **predictable names**: same concept → same name in params, receivers, and
  docs. Minimize **dependencies**. Do not rely on internal or undocumented
  behavior.

---

## Consistency

- Code should look and behave like nearby and project code.
- Same problem → same approach in a package.
- Same concept → same naming.
- Do not override the guide for “local consistency.” If the guide states a rule,
  follow the guide unless a documented exception applies.

---

## Core guidelines

### Formatting

- All Go source must match **`gofmt`** output.
- Format generated code too (for example with `format.Source`).

### MixedCaps

- Use **MixedCaps** for exported multi-word names.
- Use **mixedCaps** for unexported multi-word names.
- Do not use snake_case.
- Constants: `MaxLength` (exported), `maxLength` (unexported).
- Local variables follow unexported capitalization.

### Line length

- No fixed line length.
- If a line is too long, **refactor** first. Then consider wrapping.
- Do **not** split a line:
  - Before an indentation change (function declaration, conditional)
  - Only to wrap a long string (for example a URL)

### Naming

- Keep names short and clear.
- Use context.
- Avoid repetition and restating the obvious.
- See [style-decisions.md](style-decisions.md) for detailed naming rules.

### Local consistency

- When the guide is silent, match **nearby** code (same file or package).
- **Valid** local choices include `%s` vs `%v` for errors, and buffered channels
  vs mutexes when both fit.
- **Invalid** local choices include fixed line-length limits and assertion-based
  test libraries.
- New code must not **worsen** existing deviations. Fix or refactor in the same
  change, or choose an approach that does not spread the problem.
