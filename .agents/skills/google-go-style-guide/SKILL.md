---
name: google-go-style-guide
description: >-
  Apply Google Go style principles, naming, formatting, errors, tests, and API
  patterns. Use when writing or reviewing Go code, formatting with gofmt,
  choosing names, structuring imports, handling errors, writing godoc, designing
  function options, or shaping table-driven tests.
trigger: >-
  gofmt, MixedCaps, Go style, Google Go style, naming, imports, errors, %w,
  table-driven tests, option structs, context.Context, godoc, Go review
---

# Google Go Style Guide

Use this skill when you write or review Go code. Follow Google Go style so code
stays clear, simple, and easy to change.

Official overview: https://google.github.io/styleguide/go/index  
Full link list: [references.md](references.md)

Assume knowledge of [Effective Go](https://go.dev/doc/effective_go).

---

## 1. Document roles

| Document | Role | Use when |
| -------- | ---- | -------- |
| **Style Guide** | Normative and canonical. Foundation. | All Go code. See [guide.md](guide.md). |
| **Style Decisions** | Normative, not canonical. Detailed decisions. | Naming, commentary, imports, errors, language, tests. See [style-decisions.md](style-decisions.md). |
| **Best Practices** | Not normative or canonical. Common patterns. | Call-site naming, errors, docs, tests, APIs, global state. See [best-practices.md](best-practices.md). |

If Style Decisions conflict with the Style Guide, follow [guide.md](guide.md).

---

## 2. Definitions

- **Canonical**: Enduring rules. All code should follow them. They change rarely.
- **Normative**: Agreed style for reviews. Terms stay consistent. Rules may change.
- **Idiomatic**: Common Go patterns. Prefer idiomatic forms when they do the same job.

---

## 3. Style principles (order of importance)

Use this order from the Style Guide. Consistency does **not** override the other
principles. Use Consistency to break ties.

1. **Clarity** — The reader sees purpose and reason.
2. **Simplicity** — Use the simplest code that meets goals.
3. **Concision** — Keep a high signal-to-noise ratio.
4. **Maintainability** — Make correct change easy over time.
5. **Consistency** — Match nearby and project code when principles do not decide.

---

## 4. Core guidelines (canonical)

- **Formatting**: Match **`gofmt`** output. Format generated code too.
- **MixedCaps**: Use **MixedCaps** for exported names. Use **mixedCaps** for
  unexported names. Do not use snake_case. Locals follow unexported rules.
- **Line length**: No fixed limit. Refactor before you wrap. Do **not** split a
  line before an indentation change. Do **not** wrap a long string only to
  shorten a line.
- **Naming**: Keep names short and clear. Use context. Do not repeat the obvious.
  See [style-decisions.md](style-decisions.md) and
  [best-practices.md](best-practices.md).
- **Local consistency**: If the guide is silent, match nearby code in the same
  file or package. Do not spread existing deviations. Fix them or avoid them.

---

## 5. What these documents do

These documents:

- Set principles for style choices
- Record settled Go style
- Show canonical examples and idioms
- Explain style decisions
- Reduce surprise in reviews
- Keep reviewer terms consistent

These documents do **not**:

- Replace judgment
- List every review comment
- Justify large style-only rewrites

Write **new** code to current guidance. Fix nearby issues over time.

---

## 6. Application workflow

When you write or edit `.go` files:

1. Apply this overview (principles and core guidelines).
2. Apply [guide.md](guide.md) for principles and core guidelines in full.
3. Apply [style-decisions.md](style-decisions.md) for naming, commentary,
   imports, errors, language, libraries, and tests.
4. Apply [best-practices.md](best-practices.md) for call-site naming, error
   context, docs, variables, function args, tests, strings, and global state.
5. Open [references.md](references.md) for official links.

---

## 7. Quick checklist

- [ ] Does `gofmt` accept the file?
- [ ] Do names use MixedCaps and avoid needless repetition?
- [ ] Is purpose clear to the reader (what and why)?
- [ ] Is the simplest mechanism enough (language, then stdlib, then project libs)?
- [ ] Are errors handled, returned, or clearly ignored?
- [ ] Is `context.Context` the first parameter where needed, and not stored in structs?
- [ ] Are interfaces owned by the consumer when used for substitution?
- [ ] Do tests use `testing` (and `cmp` if needed) without assertion libraries?
- [ ] Do failure messages identify inputs and show got before want?
- [ ] Are package-level globals avoided for client behavior?

---

## 8. Support files

- **Style Guide:** [guide.md](guide.md)
- **Style Decisions:** [style-decisions.md](style-decisions.md)
- **Best Practices:** [best-practices.md](best-practices.md)
- **Official links:** [references.md](references.md)
