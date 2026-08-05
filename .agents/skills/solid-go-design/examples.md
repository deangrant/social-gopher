# SOLID (Go) — Examples

Teaching snippets for packages, interfaces, and dependency direction.
See [SKILL.md](SKILL.md) for guidance and [reference.md](reference.md) for
checklists.

---

## Example 1: SRP — split mixed responsibilities

**Weak:** One method saves a user, sends mail, and logs. It has many reasons to
change.

```go
func (s *UserService) Register(username, password string) error {
	// save user to database
	// send confirmation email
	// log registration event
	return nil
}
```

**Better:** The service coordinates. Each dependency owns one concern.

```go
type UserStore interface {
	SaveUser(username, password string) error
}

type Mailer interface {
	SendConfirmation(username string) error
}

type Logger interface {
	Log(message string)
}

type UserService struct {
	store  UserStore
	mailer Mailer
	logger Logger
}

func NewUserService(store UserStore, mailer Mailer, logger Logger) *UserService {
	return &UserService{store: store, mailer: mailer, logger: logger}
}

func (s *UserService) Register(username, password string) error {
	if err := s.store.SaveUser(username, password); err != nil {
		return err
	}
	if err := s.mailer.SendConfirmation(username); err != nil {
		return err
	}
	s.logger.Log("user registered: " + username)
	return nil
}
```

**Tie-back:** **SRP** — registration policy stays in `UserService`. Storage,
mail, and logging change behind separate interfaces.

---

## Example 2: OCP — extend without a growing switch

**Weak:** Each new payment method edits `ProcessPayment`.

```go
func (p *PaymentProcessor) ProcessPayment(method string) {
	if method == "credit_card" {
		// process card
	} else if method == "paypal" {
		// process paypal
	}
}
```

**Better:** Add a new type that implements `PaymentMethod`. Leave the processor
stable.

```go
type PaymentMethod interface {
	Process() error
}

type CreditCard struct{}

func (CreditCard) Process() error {
	// process card
	return nil
}

type PayPal struct{}

func (PayPal) Process() error {
	// process paypal
	return nil
}

type PaymentProcessor struct{}

func (PaymentProcessor) ProcessPayment(method PaymentMethod) error {
	return method.Process()
}
```

**Tie-back:** **OCP** — new methods add new types. Stable code depends on
`PaymentMethod` only.

---

## Example 3: LSP — keep substitutable contracts

**Weak:** Code that mutates a rectangle breaks square constraints if you force
one type into the other.

**Better:** Share behavior through a small interface. Each type keeps its own
rules.

```go
type Shape interface {
	Area() float64
}

type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

type Square struct {
	Side float64
}

func (s Square) Area() float64 {
	return s.Side * s.Side
}

func PrintArea(shape Shape) {
	fmt.Printf("area: %.2f\n", shape.Area())
}
```

**Tie-back:** **LSP** — `Rectangle` and `Square` both satisfy `Shape`. Callers
can substitute either type without broken invariants.

---

## Example 4: ISP — accept only the methods you need

**Weak:** `Save` takes `*os.File` and couples to disk-only APIs.

```go
func Save(f *os.File, doc *Document) error {
	// write doc to f
	return nil
}
```

**Still too wide:** `io.ReadWriteCloser` forces read and close duties that
`Save` does not need.

```go
func Save(rwc io.ReadWriteCloser, doc *Document) error {
	// write doc to rwc
	return nil
}
```

**Better:** Accept `io.Writer`. The function states only the needed capability.

```go
func Save(w io.Writer, doc *Document) error {
	// write doc to w
	return nil
}
```

**Role interfaces for domain clients:**

```go
type Worker interface {
	Work()
}

type Eater interface {
	Eat()
}

type Sleeper interface {
	Sleep()
}
```

A robot implements `Worker` only. It does not implement `Eat` or `Sleep`.

**Tie-back:** **ISP** — clients depend on small method sets. Narrow parameters
make functions easier to test and reuse.

---

## Example 5: DIP — invert dependencies and wire in main

**High-level package owns the interfaces it needs:**

```go
package signup

import "context"

type User struct {
	ID    string
	Email string
}

type UserStore interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Save(ctx context.Context, user User) error
}

type IDGenerator interface {
	NewID() string
}

type WelcomeSender interface {
	SendWelcome(ctx context.Context, user User) error
}

type Service struct {
	users    UserStore
	ids      IDGenerator
	welcome  WelcomeSender
}

func NewService(users UserStore, ids IDGenerator, welcome WelcomeSender) *Service {
	return &Service{users: users, ids: ids, welcome: welcome}
}

func (s *Service) Signup(ctx context.Context, email string) (User, error) {
	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	if exists {
		return User{}, errEmailTaken
	}
	user := User{ID: s.ids.NewID(), Email: email}
	if err := s.users.Save(ctx, user); err != nil {
		return User{}, err
	}
	if err := s.welcome.SendWelcome(ctx, user); err != nil {
		return User{}, err
	}
	return user, nil
}
```

**Concrete package implements details:**

```go
package postgres

import (
	"context"
	"database/sql"

	"example.com/app/internal/signup"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const q = `select exists(select 1 from users where email = $1)`
	var exists bool
	err := s.db.QueryRowContext(ctx, q, email).Scan(&exists)
	return exists, err
}

func (s *UserStore) Save(ctx context.Context, user signup.User) error {
	const q = `insert into users (id, email) values ($1, $2)`
	_, err := s.db.ExecContext(ctx, q, user.ID, user.Email)
	return err
}
```

**Composition root wires concrete types:**

```go
func main() {
	db := mustOpenDB()
	defer db.Close()

	users := postgres.NewUserStore(db)
	ids := uuidGenerator{}
	welcome := email.NewWelcomeSender(smtpClient)

	service := signup.NewService(users, ids, welcome)
	handler := httpapi.NewSignupHandler(service)

	log.Fatal(http.ListenAndServe(":8080", handler))
}
```

**Test with fakes:**

```go
type fakeUserStore struct {
	byEmail map[string]signup.User
}

func (s *fakeUserStore) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, ok := s.byEmail[email]
	return ok, nil
}

func (s *fakeUserStore) Save(ctx context.Context, user signup.User) error {
	s.byEmail[user.Email] = user
	return nil
}

func TestSignupCreatesUser(t *testing.T) {
	users := &fakeUserStore{byEmail: map[string]signup.User{}}
	service := signup.NewService(users, fixedIDs{id: "user-123"}, noopWelcome{})

	got, err := service.Signup(context.Background(), "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "user-123" {
		t.Fatalf("id = %q, want %q", got.ID, "user-123")
	}
}
```

**Tie-back:** **DIP** — `signup` depends on abstractions it defines. `postgres`
and `email` supply details. `main` and tests wire the graph.

---

## Example 6: combined review pattern — matcher filter

Use one stable filter. Extend match rules with new types.

```go
type Mobile struct {
	Brand  string
	Memory int
}

type Matcher interface {
	Matched(mobile Mobile) bool
}

type Brand struct {
	Name string
}

func (b Brand) Matched(mobile Mobile) bool {
	return mobile.Brand == b.Name
}

type Memory struct {
	Size int
}

func (m Memory) Matched(mobile Mobile) bool {
	return mobile.Memory == m.Size
}

type Filter struct{}

func (Filter) Filter(mobiles []Mobile, matcher Matcher) []Mobile {
	var matches []Mobile
	for _, mobile := range mobiles {
		if matcher.Matched(mobile) {
			matches = append(matches, mobile)
		}
	}
	return matches
}
```

**Tie-back:** **OCP** and **ISP** — `Filter` stays closed. New match rules add
small types. Clients pass only the matcher they need.
