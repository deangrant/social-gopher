# SOLID (Go) — Reference

Quick lookup: tables, checklists, and Go idioms. Concepts align with
[SKILL.md](SKILL.md). Code patterns are in [examples.md](examples.md).

---

## Principle summary

| Principle | Go idiom | Key question | Anti-pattern indicator |
| --------- | -------- | ------------ | ---------------------- |
| **SRP** | Focused packages, cohesive types, clear constructors | What single purpose does this package or type serve? | One type validates, stores, sends mail, and formats HTTP |
| **OCP** | Interfaces, composition, function values, new impls | Can I add behavior without editing stable core code? | Every new feature edits one large `switch` or service |
| **LSP** | Implicit interfaces with documented semantics | Can any implementation substitute without breaking callers? | Impl panics, ignores context, or changes error meaning |
| **ISP** | Small consumer-owned interfaces | Does this client need every method on this interface? | Fat `Repository` forces unused methods or empty stubs |
| **DIP** | Interfaces at use sites; wire in `main` | Does high-level code import concrete DB or SDK types? | Domain package imports `database/sql` drivers deeply |

---

## SRP — expanded checklist

**Too-many-responsibilities signs**

- [ ] One package mixes HTTP, persistence, and domain rules with no boundary.
- [ ] Changes for feature A force retests of unrelated feature B.
- [ ] One struct owns SMTP, SQL, and payment clients together.
- [ ] Tests need a database, broker, and mail server for one unit.

**Cohesion indicators (good)**

- [ ] Edits cluster in one package when one requirement changes.
- [ ] The package name states the purpose clearly.
- [ ] You can describe the responsibility in one sentence without “and also.”

**Over-fragmentation warnings**

- [ ] Every type has its own package with awkward exports.
- [ ] Call chains hide simple data flow.
- [ ] Splits create duplicate types and mapping code with no benefit.

**“Reasons to change” prompts**

- If validation rules change, which packages move?
- If the database schema changes, which packages move?
- If the mail provider changes, which packages move?
- Overlap in answers suggests mixed responsibilities.

### SRP — when to apply vs overkill

| Apply | Overkill |
| ----- | -------- |
| A second axis of change is clear. | Split for purity with no second use or test pain. |
| Tests are hard because concerns mix. | Extract a one-use helper into a new package. |
| Package name cannot state one purpose. | One logical service cut into many tiny packages. |

---

## OCP — expanded checklist

**Open-for-extension signs**

- [ ] New behavior arrives as a new type that implements an interface.
- [ ] Stable algorithms depend on interfaces or small function values.
- [ ] Reviewers see most additions in new files, not edits to fragile cores.

**Closed-for-modification signs**

- [ ] Core loops rarely change when you add a new format or provider.
- [ ] Callers of the stable API do not need signature changes for each variant.

**Premature abstraction signs**

- [ ] Interface has one implementation and no test fake.
- [ ] Names end in `Interface` or `Impl` with no real variation.
- [ ] Layers of factories hide a single constructor call.

### OCP — when to apply vs overkill

| Apply | Overkill |
| ----- | -------- |
| Two or more real behaviors share a call site. | Interface “for future plugins” with no second case. |
| Stable core must not grow with each provider. | Function values or a small `switch` would stay clearer. |

---

## LSP — expanded checklist

**Contract quality**

- [ ] Interface comments state success and error behavior.
- [ ] Implementations return errors in the same cases.
- [ ] Nil and zero-value rules match across implementations.
- [ ] `context.Context` cancel stops work where promised.
- [ ] Concurrency safety matches what callers assume.

**Substitution failures**

- [ ] One impl panics where others return `error`.
- [ ] One impl writes partial data and reports success.
- [ ] One impl requires extra setup not stated by the interface.
- [ ] Callers use type switches to special-case “broken” impls.

### LSP — when to apply vs overkill

| Apply | Overkill |
| ----- | -------- |
| Multiple impls share one call path. | Deep type trees that mimic class inheritance. |
| Fakes in tests must match production rules. | Formal proofs for trivial local helpers. |

---

## ISP — expanded checklist

**Segregation signs (good)**

- [ ] Each interface matches one client role.
- [ ] Function parameters list only needed methods.
- [ ] Consumer package owns the interface definition.
- [ ] Implementors do not stub unused methods.

**Fat interface signs (bad)**

- [ ] One interface mixes read, write, admin, and lifecycle methods.
- [ ] Many implementors leave methods empty or panic.
- [ ] Changing one unused method forces wide recompiles.

### ISP — when to apply vs overkill

| Apply | Overkill |
| ----- | -------- |
| Different clients need different method sets. | One method per interface with no separate clients. |
| Concrete `*os.File` exposes unused methods. | Renaming every one-method interface without need. |

**Rule of thumb:** Accept the smallest interface the function needs. Return
concrete structs from constructors.

---

## DIP — expanded checklist

**Inversion signs (good)**

- [ ] High-level package imports few or no infrastructure packages.
- [ ] Dependencies appear as interface fields or parameters.
- [ ] `main` or a setup package constructs concrete types.
- [ ] Unit tests inject fakes without network or disk.

**Concrete coupling signs (bad)**

- [ ] Domain logic calls `sql.Open` or a vendor SDK directly.
- [ ] Package-level mutable globals hold required services.
- [ ] Import cycles appear between policy and detail packages.

### DIP — when to apply vs overkill

| Apply | Overkill |
| ----- | -------- |
| You need tests without I/O. | Interface over a single stable stdlib helper. |
| You may swap providers. | DI framework for a small program. |
| Import direction is wrong today. | Abstract every struct field “by policy.” |

---

## Package design notes

SOLID works best at **package boundaries**.

Prefer capability packages:

```plain
cmd/api/           # wire dependencies
internal/signup/   # use case + small interfaces
internal/postgres/ # implements store interfaces
internal/email/    # implements notifier interfaces
internal/httpapi/  # transport adapters
```

Avoid dump layouts that collect unrelated code:

```plain
internal/models/
internal/repositories/
internal/services/
internal/utils/
```

**Import graph goals**

- Keep imports acyclic.
- Prefer a wide, flat graph.
- Push concrete knowledge up to `main`.
- Leave lower packages with abstractions they define or need.

---

## When SOLID goes wrong in Go

1. **Premature interfaces** — one implementation, no test boundary, names like
   `FooInterface` / `FooImpl`.
2. **Class-style design** — deep embedding trees used as inheritance.
3. **Over-split packages** — one type per package, hard navigation, awkward API.
4. **Hidden injection** — package globals or service locators hide dependencies.
5. **Provider-owned fat interfaces** — large interfaces forced on all clients.
6. **Abstraction without variation** — interfaces added before a second real
   behavior exists.

---

## Practical review checklist

Use this list in code review.

### SRP

- [ ] Does this package have a clear purpose?
- [ ] Does this function do one coherent job?
- [ ] Are validation, persistence, transport, and orchestration mixed without need?
- [ ] Would a small change force edits in unrelated code?

### OCP

- [ ] Can new behavior arrive without editing stable code?
- [ ] Is the extension point based on real variation?
- [ ] Would a function value be simpler than an interface?
- [ ] Is a small `switch` clearer because the case set is stable?

### LSP

- [ ] Are interface contracts documented?
- [ ] Do implementations handle errors consistently?
- [ ] Do implementations respect `context.Context`?
- [ ] Are concurrency guarantees consistent?

### ISP

- [ ] Does the client depend only on methods it uses?
- [ ] Is the interface defined by the consumer?
- [ ] Could a large interface split into smaller role interfaces?
- [ ] Are concrete types better here?

### DIP

- [ ] Are dependencies passed explicitly?
- [ ] Does high-level policy avoid importing low-level infrastructure packages?
- [ ] Is `main` responsible for wiring concrete dependencies?
- [ ] Can the unit run in tests without network, disk, or a database?

---

## Design takeaways

Use SOLID as a review lens, not as a rigid template.

Strong Go designs usually have:

- narrow interfaces at consumer boundaries
- explicit dependencies through constructors or parameters
- cohesive packages with clear reasons to change
- concrete types until abstraction is justified
- documented interface behavior
- errors wrapped with useful context
- `context.Context` honored on I/O paths
- small fakes for substitutable implementations

Good Go design stays simple, explicit, and easy to change for the reasons your
system actually changes.
