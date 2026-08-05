# Google Go Style — Official References

This file lists official Go Style materials and defines shared terms.

## Source

**Go Style (Overview)**  
https://google.github.io/styleguide/go/index

## Document structure

| Document | Link | Primary audience | Normative | Canonical |
| -------- | ---- | ---------------- | --------- | --------- |
| **Style Guide** | [guide](https://google.github.io/styleguide/go/guide) | Everyone | Yes | Yes |
| **Style Decisions** | [decisions](https://google.github.io/styleguide/go/decisions) | Readability mentors | Yes | No |
| **Best Practices** | [best-practices](https://google.github.io/styleguide/go/best-practices) | Anyone interested | No | No |

- **Style Guide**: Foundation of Go style. Definitive. Basis for the other two.
- **Style Decisions**: Detailed decisions and reasons. May change with language
  or patterns.
- **Best Practices**: Common patterns that read well. Not canonical. Use them to
  keep a codebase uniform.

Skill files:

- [guide.md](guide.md)
- [style-decisions.md](style-decisions.md)
- [best-practices.md](best-practices.md)
- [SKILL.md](SKILL.md)

## Definitions

- **Canonical**: Prescriptive and enduring. All code should follow. Change is
  rare. Canonical docs stay short.
- **Normative**: Agreed style for reviewers. Terms stay consistent. Rules may
  change. Authors need not memorize every detail.
- **Idiomatic**: Common and familiar in Go. Prefer idiomatic patterns when both
  forms serve the same purpose.

## Additional references

The guide assumes familiarity with
[Effective Go](https://go.dev/doc/effective_go).

### External

- [Go Language Specification](https://go.dev/ref/spec)
- [Go FAQ](https://go.dev/doc/faq)
- [Go Memory Model](https://go.dev/ref/mem)
- [Go Data Structures](https://research.swtch.com/godata)
- [Go Interfaces](https://research.swtch.com/interfaces)
- [Go Proverbs](https://go-proverbs.github.io/)

### Testing

- Unit testing practices in the style guide and best practices
- TotT: Identifier Naming
- TotT: Testing State vs. Testing Interactions
- TotT: Effective Testing
- TotT: Risk-driven Testing
- TotT: Change-detector Tests Considered Harmful

### Other writings

- [Go and Dogma](https://go.dev/blog/go-and-dogma)
- [Less is exponentially more](https://commandcenter.blogspot.com/2012/06/less-is-exponentially-more.html)
- [Gofmt's style is no one's favorite, yet Gofmt is everyone's favorite](https://www.youtube.com/watch?v=PAAkCSZYW1c)
