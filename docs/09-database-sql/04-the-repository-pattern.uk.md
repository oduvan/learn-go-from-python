# Патерн репозиторію

Розкидання SQL по всіх ваших обробниках прив'язує всю програму до
однієї бази даних. Виправлення — це межа: інтерфейс, що описує те, що
вам потрібно, і один пакет, який його реалізує.

```go
type UserStore interface {
    ByID(ctx context.Context, id int64) (User, error)
    Create(ctx context.Context, name string) (User, error)
}
```

Усе, що вище цієї межі, працює в термінах `User` і `ErrUserNotFound`.
Ніщо вище неї не імпортує `database/sql`.

## Два пакети, один контракт

Інтерфейс, доменні типи й сигнальні помилки (sentinel errors) лежать у
пакеті-**контракті**. Він взагалі не має імпортів бази даних:

```go
type User struct {
    ID   int64
    Name string
}

var ErrUserNotFound = errors.New("user not found")

type UserStore interface {
    ByID(ctx context.Context, id int64) (User, error)
    Create(ctx context.Context, name string) (User, error)
}
```

SQL живе в окремому пакеті-**реалізації**, у неекспортованій структурі,
що тримає пул:

```go
type userStore struct{ db *sql.DB }

func (s userStore) ByID(ctx context.Context, id int64) (User, error) {
    var u User
    err := s.db.QueryRowContext(ctx,
        `SELECT id, name FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Name)

    if errors.Is(err, sql.ErrNoRows) {
        return User{}, fmt.Errorf("id %d: %w", id, ErrUserNotFound)
    }
    if err != nil {
        return User{}, fmt.Errorf("selecting user %d: %w", id, err)
    }
    return u, nil
}
```

Неекспортована, бо виклики (callers) мають тримати саме інтерфейс.
Конструктор повертає його:

```go
func NewUserStore(db *sql.DB) UserStore { return userStore{db: db} }
```

Доведіть, що реалізація задовольняє контракт, на етапі компіляції:

```go
var _ UserStore = userStore{}
```

## Перетворюйте помилки сховища на межі

Це та частина, що несе на собі весь патерн. `sql.ErrNoRows` — це
концепція `database/sql`; якщо дати їй просочитись нагору, кожен
виклик буде імпортувати пакет SQL, і зміна сховища зламає їх усі.

Сховище перетворює її:

```go
_, err := store.ByID(ctx, 99999)

fmt.Println(errors.Is(err, ErrUserNotFound))   // output: true
fmt.Println(errors.Is(err, sql.ErrNoRows))     // output: false
fmt.Println(err)                               // output: id 99999: user not found
```

Обгортання через `%w` лишає `errors.Is` робочим, водночас додаючи,
якого саме id бракувало. Неочікувані помилки теж обгортаються, але
проходять як самі собою — виклик не може сплутати обірване з'єднання з
відсутнім рядком.

Тепер бізнес-логіка читається в доменних термінах:

```go
func (g Greeter) Greet(ctx context.Context, id int64) (string, error) {
    u, err := g.users.ByID(ctx, id)
    if errors.Is(err, ErrUserNotFound) {
        return "hello, stranger", nil
    }
    if err != nil {
        return "", fmt.Errorf("greeting %d: %w", id, err)
    }
    return "hello, " + u.Name, nil
}
```

## Залежте від найвужчого інтерфейсу

Єдиний інтерфейс `Store` із сорока методами змушує кожного споживача
знати про всі вони, а кожен тестовий дублікат — реалізовувати їх усі.
Оголошуйте той невеликий набір, який справді використовує кожен
споживач:

```go
type Greeter struct{ users UserStore }
```

`Greeter` потребує два методи. Дайте йому два. Це принцип «приймайте
інтерфейси, повертайте конкретні типи» зі статті про
[інтерфейси](../03-object-oriented-go/02-interfaces.md), застосований
до сховищ.

Структура-контейнер — розумний спосіб зібрати їх усіх при старті —
просто не передавайте весь контейнер у кожен компонент:

```go
type Stores struct {
    Users UserStore
    Posts PostStore
}

g := Greeter{users: stores.Users}   // не так: Greeter{stores}
```

## Тестування без бази даних

Оскільки споживач тримає інтерфейс, достатньо написаного вручну
дубліката. Поля-функції дозволяють кожному тесту сказати точно те, що
йому потрібно:

```go
type stubUsers struct {
    ByIDFn func(ctx context.Context, id int64) (User, error)
}

func (s stubUsers) ByID(ctx context.Context, id int64) (User, error) {
    return s.ByIDFn(ctx, id)
}
func (s stubUsers) Create(context.Context, string) (User, error) { return User{}, nil }
```

Та сама логіка тепер працює взагалі без бази даних:

```go
g := Greeter{users: stubUsers{ByIDFn: func(_ context.Context, id int64) (User, error) {
    return User{ID: id, Name: "Stub"}, nil
}}}
fmt.Println(g.Greet(ctx, 1))   // output: hello, Stub <nil>
```

А гілки, які незручно відтворити на справжній базі даних, тепер
стають одним рядком кожна:

```go
// не знайдено
return User{}, ErrUserNotFound
// → hello, stranger <nil>

// збій з'єднання
return User{}, errors.New("connection reset")
// → greeting 7: connection reset
```

Другий випадок — це аргумент на користь усього патерну. Симулювати
обірване з'єднання проти живої бази даних складно; проти інтерфейсу —
тривіально.

## Де це перестає бути виправданим

- **Не додавайте метод на кожен запит.** Сховище з методами `ByName`,
  `ByNameAndStatus`, `ByNameAndStatusAndCreatedAfter` — це погано
  написаний конструктор запитів. Замість цього візьміть
  структуру-фільтр.
- **Не мапте геть усе.** Звітний запит, що живить один ендпоінт, може
  бути єдиним методом, що повертає спеціально створений тип
  результату. Проштовхування його крізь доменну модель нікому не
  допомагає.
- **Не абстрагуйте заради заміни, якої не станеться.** Справжня
  користь — це тестованість і те, що SQL зібраний в одному місці.
  «Можливо, ми змінимо базу даних» майже ніколи не збувається, а
  проєктування під це лише погіршує інтерфейс.
- **Транзакції перетинають сховища.** Двом сховищам в одній транзакції
  потрібно її ділити — і це саме транзакція, перенесена через
  контекст, з [попередньої статті](03-transactions.md), саме тому, що
  сигнатури методів сховища не згадують `*sql.Tx`.

> **З досвіду Python:** це той самий патерн репозиторію, який ви б
> побудували над SQLAlchemy, але тут інтерфейс визначає *споживач*, і
> задовольняється він неявно, тож немає ні базового класу, ні
> реєстрації. Тестовий дублікат — це будь-яка структура з потрібними
> методами.

## Швидка довідка

| Питання | Що робити |
|---|---|
| де живе інтерфейс | у пакеті-контракті, разом із доменними типами |
| де живе SQL | в окремому пакеті, у неекспортованій структурі |
| конструктор | `func NewUserStore(db *sql.DB) UserStore` |
| довести, що задовольняє | `var _ UserStore = userStore{}` |
| відсутність рядків | перетворити на власний `ErrNotFound`, обгорнутий через `%w` |
| інші помилки | обгорнути з контекстом, пропустити далі |
| що приймають споживачі | найвужчий інтерфейс, який вони використовують |
| тестові дублікати | структура з полями-функціями |
| транзакції між сховищами | нести їх у контексті |
| гранулярність | структури-фільтри, а не метод на кожен запит |

## Джерела

- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
- [`errors.Is` — pkg.go.dev/errors#Is](https://pkg.go.dev/errors#Is)
- [`sql.ErrNoRows` — pkg.go.dev/database/sql#pkg-variables](https://pkg.go.dev/database/sql#pkg-variables)
- [Go Code Review Comments: interfaces — go.dev/wiki/CodeReviewComments#interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- [Accessing databases — go.dev/doc/database/](https://go.dev/doc/database/)
