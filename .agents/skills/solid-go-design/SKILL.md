---
name: solid-go-design
description: >-
  Apply SOLID design principles in idiomatic Go for maintainable packages and
  interfaces. Use when writing or reviewing Go code for modularity, designing
  small interfaces, managing package boundaries and imports, refactoring for
  testability, or evaluating whether abstractions are justified. Covers
  consumer-owned interfaces, composition and embedding, constructor injection,
  and dependency direction.
trigger: >-
  SOLID, SRP, OCP, LSP, ISP, DIP, Go interfaces, package design, accept
  interfaces return structs, consumer-owned interfaces, dependency injection,
  import graph, embedding, Go design review, refactoring Go for testability
---

# SOLID Design Principles (Go)

Use this skill when you shape Go packages for clear structure and safe change.
Go does not use class hierarchies. Use **packages**, **structs**,
**composition**, and **small interfaces** instead.

The five principles still apply. They guide where to split packages, how narrow
to make interfaces, and when to keep a concrete type.

---

## 1. SOLID in Go (brief overview)

| Principle | One-line focus in Go |
| --------- | -------------------- |
| **S**RP | Keep one clear reason to change for each package, type, or function. |
| **O**CP | Add behavior with new types or implementations. Do not edit stable core code when you can avoid it. |
| **L**SP | Each implementation must keep the behavioral contract that callers expect. |
| **I**SP | Clients depend only on the methods they need. Prefer small interfaces. |
| **D**IP | High-level code depends on abstractions. Concrete types live at the edges. |

**Interfaces as contracts:** An interface defines required behavior. Types
satisfy interfaces by method set. They do not declare the interface by name.

**Composition over inheritance:** Embed structs to reuse fields and methods.
Do not model deep type trees.

**Default rule:** Keep concrete types until a second real implementation or a
test boundary needs an interface.

---

## 2. Single Responsibility Principle (SRP)

**Idea:** A package, type, or function has **one reason to change**. Related
code that changes together stays together.

**Cohesion:** Put related functions and types in one package. Keep unrelated
concerns apart.

**Misconceptions and balance:**

- SRP does **not** mean one function per file or one method per type.
- Do **not** split code only to make a diagram look clean.
- Split when a second axis of change is real and clear.
- Keep code together when a split only adds noise.

**Go mechanisms:**

- **Package names:** Name a package by its purpose (`http`, `json`, `checkout`).
  Avoid dump names such as `utils`, `common`, `helpers`, or `misc`.
- **Focused types:** Keep domain rules near the data they govern.
- **Narrow dependencies:** Pass storage, mail, and payment behind small
  interfaces when those concerns change for different reasons.
- **Constructor injection:** Pass required dependencies into `New...` so the
  type responsibility stays clear.

**When not to apply:** Do not extract a helper that one call site uses once and
never reuses.

---

## 3. Open/Closed Principle (OCP)

**Idea:** Code is **open for extension** and **closed for modification** of
stable, well-tested core logic.

**Framing:** Prefer to **add** a new type that implements an interface, or a new
strategy struct, instead of growing a central `if` or `switch` that knows every
detail—*when that core is truly stable*.

**Misconceptions:**

- OCP does **not** mean “never change any file.” Early code changes freely.
- OCP does **not** require an interface for every feature.
- Simple code that changes in one place is often correct.

**Over-engineering warning:** Add an extension point only when multiple real
behaviors exist or will exist soon. One implementation does not need an
interface “for later.”

**Go mechanisms:**

- **Interfaces:** Add a new implementation without editing callers that depend
  on the interface.
- **Composition and embedding:** Build richer types from smaller types.
- **Function values:** A small `func` parameter can extend behavior without a
  named interface type.
- **Middleware and hooks:** Extend request handling without editing core
  handlers when the hook boundary is stable.

**When not to apply:** Prefer a clear `switch` when the set of cases is small
and stable.

---

## 4. Liskov Substitution Principle (LSP)

**Idea:** Any type that satisfies an interface must be a **safe substitute**.
Callers must not need to know the concrete type to stay correct.

**Behavioral contracts:**

- **Preconditions:** What must be true before a method runs.
- **Postconditions:** What the method guarantees after it returns.
- **Invariants:** What stays true across operations.

**Contract rules:** An implementation must **not** require more from the caller
than the interface states. It must **not** deliver less than the interface
promises.

**Misconceptions:** LSP is not about identical structs. It is about preserved
assumptions, including errors, empty results, and side effects.

**Go mechanisms:**

- **Implicit interfaces:** Any type with the matching methods satisfies the
  interface. Document the **meaning**, not only the signature.
- **Small interfaces:** Small contracts are easier to satisfy correctly.
- **Standard library models:** Types that act like `io.Reader` must follow the
  same read and error rules callers expect.
- **Context:** Implementations that accept `context.Context` must honor cancel
  and deadline rules consistently.

**Pitfalls:** Do not return surprising nil values where callers expect a value
or a clear error. Do not ignore context cancel. Do not change concurrency
guarantees across implementations.

---

## 5. Interface Segregation Principle (ISP)

**Idea:** Clients must not depend on methods they do not use. Prefer **small,
focused** interfaces over one large interface.

**Misconception:** ISP does **not** mean exactly one method per interface. Group
methods by the **role the client needs**.

**Go mechanisms:**

- **Consumer-owned interfaces:** Define the interface in the package that uses
  it, not only in the package that implements it.
- **Narrow parameters:** If a function only writes bytes, accept `io.Writer`,
  not `*os.File` or a large custom interface.
- **Role interfaces:** Name interfaces by capability (`Store`, `Notifier`,
  `Clock`), not by a large system noun.
- **Accept interfaces, return structs:** Functions accept the smallest
  interface they need. Constructors and factories return concrete types.

**Balance:** Do not create many tiny interfaces with no separate clients. Split
when different callers need different method sets.

---

## 6. Dependency Inversion Principle (DIP)

**Idea:** High-level policy must not depend on low-level details. Both depend on
**abstractions**. Details implement those abstractions.

**Misconception:** DIP is **not** the same as dependency injection. DIP sets
**dependency direction**. Constructor injection is one technique that supports
it.

**Go mechanisms:**

- **Interfaces at the boundary:** High-level packages declare the behavior they
  need.
- **Concrete packages at the edge:** Database, mail, and HTTP clients implement
  those interfaces.
- **Composition root:** Wire concrete types in `main` or a small setup package.
- **Acyclic imports:** Keep the import graph free of cycles. Prefer a wide,
  flat graph over a tall, narrow one.
- **Explicit fields:** Store dependencies in struct fields. Avoid hidden package
  globals for required services.

**When not to apply:** Direct use of a concrete type is fine when there is one
stable implementation and no test or swap pressure.

**Testing:** Small fakes that satisfy consumer interfaces let you test use cases
without disk, network, or a database.

---

## 7. Cross-references

- **Code examples:** [examples.md](examples.md) — numbered Go snippets for SRP
  through DIP.
- **Checklists and tables:** [reference.md](reference.md) — principle summary,
  apply vs overkill, package design, and review checks.

---

## 8. Quick checklist (rapid validation)

**SRP**

- [ ] Can I name **one primary responsibility** for this package or type?
- [ ] Do unrelated requirement changes force edits in the **same** place?
- [ ] Did I split enough for cohesion without useless fragments?

**OCP**

- [ ] Can I **add** a new behavior with **minimal** edits to stable core code?
- [ ] Is the extension point justified by real variation, not speculation?

**LSP**

- [ ] Would **any** implementation keep callers **correct**?
- [ ] Are error, context, and concurrency rules consistent and documented?

**ISP**

- [ ] Does each client need **all** methods on this interface?
- [ ] Is the interface defined by the **consumer** and as small as the call
  site requires?

**DIP**

- [ ] Does high-level code depend on an **interface**, not a concrete
  infrastructure type?
- [ ] Is wiring in `main` or tests, not deep inside domain packages?
- [ ] Am I avoiding a needless interface over a single stable type?

---

## 9. Support files

- **Reference sheets:** [reference.md](reference.md)
- **Code examples:** [examples.md](examples.md)
