# Патерни ООП у Go

У Go немає `class`, немає успадкування й немає конструкторів. Та попри це
він покриває все, по що тягнеться об'єктноорієнтований код — просто складає
результат із чотирьох менших частин, які ви вже зустрічали: **структур**,
**методів**, **інтерфейсів** та **вбудовування**. Ця стаття зіставляє
звичні ідеї ООП із тим, як їх роблять у Go.

| Ідея ООП | Механізм Go |
|---|---|
| клас | структура + методи |
| конструктор | функція `NewT(...) T` (проста домовленість) |
| метод екземпляра | метод з отримувачем |
| інкапсуляція | експортовані проти неекспортованих імен (на рівні **пакета**) |
| успадкування | — немає; для повторного використання — **вбудовування** |
| поліморфізм | **інтерфейси** |
| абстрактний базовий клас | інтерфейс |

## «Об'єкти» — це структури з методами

Структура збирає дані; методи дають їй поведінку. Ідіоматичний
«конструктор» — це просто функція з ім'ям `New...`, що повертає готове
значення:

```go
type Counter struct{ n int }

func NewCounter(start int) *Counter { return &Counter{n: start} }

func (c *Counter) Inc()       { c.n++ }
func (c *Counter) Value() int { return c.n }

c := NewCounter(10)
c.Inc()
fmt.Println(c.Value())   // output: 11
```

## Інкапсуляція — на рівні пакета, не класу

Керування доступом у Go — це **регістр літер**: ідентифікатор, що
починається з великої літери, є *експортованим* (видимим іншим пакетам);
з малої — *неекспортованим* (видимим лише всередині свого пакета). Межа
приватності — це **пакет**, а не тип.

```go
type Account struct {
    owner   string   // неекспортоване: інші пакети не можуть це чіпати
    balance int      // неекспортоване
}

func NewAccount(owner string) *Account { return &Account{owner: owner} }

func (a *Account) Deposit(amount int) { a.balance += amount }
func (a *Account) Balance() int       { return a.balance }

a := NewAccount("Ada")
a.Deposit(100)
fmt.Println(a.Balance())   // output: 100
```

Код в іншому пакеті може викликати `Deposit` та `Balance`, але не може
прямо читати чи писати `balance` — це і є інкапсуляція. (Усередині *того
самого* пакета видно все, тож бар'єр стосується меж пакетів.)

## Композиція замість успадкування: вбудовування

Замість підкласів ви **вбудовуєте** одну структуру в іншу. Поля *й методи*
внутрішнього типу підвищуються, тож зовнішній тип ніби «має» їх — повторне
використання без ієрархії успадкування.

```go
type Logger struct{ prefix string }

func (l Logger) Log(msg string) string { return l.prefix + ": " + msg }

type Server struct {
    Logger          // вбудоване — Server отримує Log()
    addr string
}

s := Server{Logger: Logger{prefix: "srv"}, addr: ":8080"}
fmt.Println(s.Log("up"))   // output: srv: up  — підвищено з Logger
```

Підвищений метод можна **перевизначити**, оголосивши метод з тим самим
ім'ям на зовнішньому типі; зовнішній заступає внутрішній, до якого все ще
можна дістатися через ім'я поля:

```go
func (s Server) Log(msg string) string {
    return "[" + s.addr + "] " + s.Logger.Log(msg)   // явний виклик вбудованого
}

s := Server{Logger: Logger{prefix: "srv"}, addr: ":8080"}
fmt.Println(s.Log("up"))   // output: [:8080] srv: up
```

> **З погляду Python:** вбудовування виглядає як успадкування, але ним не
> є — немає ні базового класу, ні `super()`. Це *композиція* з
> автоматичним переадресуванням; до внутрішнього значення ви дістаєтеся
> явно як `s.Logger`.

## Поліморфізм через інтерфейси

Різні «підкласи» стають різними типами, що задовольняють один інтерфейс —
спільний базовий тип не потрібен:

```go
type Speaker interface{ Speak() string }

type Dog struct{}
func (Dog) Speak() string { return "woof" }

type Cat struct{}
func (Cat) Speak() string { return "meow" }

func chorus(speakers []Speaker) string {
    out := ""
    for _, s := range speakers {
        out += s.Speak() + " "
    }
    return out
}

fmt.Print(chorus([]Speaker{Dog{}, Cat{}}))   // output: woof meow 
```

## Вбудовування інтерфейсів: більше з меншого

Інтерфейси вбудовують інтерфейси, складаючи можливості. Саме так
стандартна бібліотека будує `io.ReadWriter` із `io.Reader` + `io.Writer`:

```go
type Reader interface{ Read() string }
type Writer interface{ Write(s string) }

type ReadWriter interface {
    Reader          // вбудовані інтерфейси
    Writer
}
```

Тип задовольняє `ReadWriter` автоматично, щойно має і `Read`, і `Write` —
оголошення не потрібне.

## Чому немає успадкування?

Go свідомо обходиться без успадкування. Глибокі ієрархії класів
прив'язують підклас до предків і стають крихкими; Go підштовхує вас до
маленьких інтерфейсів (для поліморфізму) плюс вбудовування (для повторного
використання), які компонуються гнучкіше. Практична настанова:
**моделюйте «is-a» через інтерфейси, а «has-a» через вбудовування** — і ви
рідко сумуватимете за класами.

## Швидка довідка

| Хочете… | Робіть так |
|---|---|
| «об'єкт» | структура + методи |
| конструктор | `func NewT(...) *T` |
| приватний стан | поля з малої літери (неекспортовані) + експортовані методи |
| повторно використати поведінку іншого типу | вбудуйте його |
| перевизначити підвищений метод | оголосіть однойменний метод на зовнішньому типі |
| поліморфізм | оголосіть інтерфейс; багато типів його задовольняють |
| поєднати можливості | вбудуйте інтерфейси |

## Джерела

- [Effective Go: embedding — go.dev/doc/effective_go#embedding](https://go.dev/doc/effective_go#embedding)
- [Struct types (embedded fields) — go.dev/ref/spec#Struct_types](https://go.dev/ref/spec#Struct_types)
- [Exported identifiers — go.dev/ref/spec#Exported_identifiers](https://go.dev/ref/spec#Exported_identifiers)
- [Interface types (embedding) — go.dev/ref/spec#Interface_types](https://go.dev/ref/spec#Interface_types)
- [Go FAQ: is Go object-oriented? — go.dev/doc/faq#Is_Go_an_object-oriented_language](https://go.dev/doc/faq#Is_Go_an_object-oriented_language)
