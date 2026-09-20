# YAML та TOML

Стандартна бібліотека має JSON і XML, але не YAML чи TOML. Обидва
поширені в конфігурації, і обидва працюють так, як навчала стаття про
[кодування JSON](../06-text-time-and-data/07-encoding-json.md) —
теги структур та `Marshal`/`Unmarshal`.

> **Модулі:** `gopkg.in/yaml.v3` та `github.com/pelletier/go-toml/v2`.

```go
var doc Doc
err := yaml.Unmarshal(data, &doc)
```

## Кілька просторів тегів на одній структурі

Тип, що читається з файлу конфігурації й віддається як JSON, несе
обидва теги — механіка зі статті про
[структури](../02-language-basics/10-structs.md), де кожна бібліотека
читає лише свій власний ключ:

```go
type Server struct {
    Host    string        `yaml:"host" json:"host" toml:"host"`
    Port    int           `yaml:"port" json:"port" toml:"port"`
    Timeout time.Duration `yaml:"timeout" json:"timeout" toml:"timeout"`
    Tags    []string      `yaml:"tags,omitempty" json:"tags,omitempty"`
    Secret  string        `yaml:"-" json:"-"`
}
```

`omitempty` і `-` означають те саме, що й у `encoding/json`. Ключ за
замовчуванням, однак, відрізняється: `encoding/json` використовує ім'я
поля Go, тоді як `yaml.v3` переводить його в нижній регістр. Пишіть тег
явно — і питання не виникає.

## YAML

```go
src := `
server:
  host: example.com
  port: 8080
  timeout: 30s
  tags: [a, b]
extra:
  k: v
`

var d Doc
err := yaml.Unmarshal([]byte(src), &d)
// {Server:{Host:example.com Port:8080 Timeout:30s Tags:[a b]} Extra:map[k:v]}
```

**`yaml.v3` парсить `time.Duration` з рядка.** `30s` стає справжньою
тривалістю без жодного власного анмаршалера — вартий знання факт, бо
саме через нього структури конфігурації можуть використовувати
правильні типи.

Маршалінг відступає чотирма пробілами й бере в лапки лише за потреби:

```go
out, _ := yaml.Marshal(Doc{Server: Server{Host: "h", Port: 1, Timeout: time.Second}})
```

```yaml
server:
    host: h
    port: 1
    timeout: 1s
extra: {}
```

### Відхиляйте невідомі поля

За замовчуванням нерозпізнаний ключ мовчки ігнорується, тож одруківка
у файлі конфігурації нічого не робить, а значення за замовчуванням
залишається. `KnownFields` перетворює це на помилку:

```go
dec := yaml.NewDecoder(r)
dec.KnownFields(true)
err := dec.Decode(&d)
// yaml: unmarshal errors:
//   line 2: field nope not found in type main.Server
```

Вмикайте це для всього, що редагує людина. Одруківка `tiemout`, після
якої залишається значення за замовчуванням — це поганий вечір.

Помилки типів уже повідомляють рядок:

```go
yaml.Unmarshal([]byte("server:\n  port: notanumber\n"), &d)
// yaml: unmarshal errors:
//   line 2: cannot unmarshal !!str `notanumber` into int
```

### У YAML є гострі кути

Вони належать формату, не бібліотеці:

- **Відступ значущий**, а таб — синтаксична помилка.
- **Значення без лапок вгадуються.** `yes`, `no`, `on`, `off` стають
  булевими; провідний нуль може стати вісімковим числом. Беріть у лапки
  все, що має залишитись рядком — особливо номери версій і коди країн.
- **Якорі та псевдоніми** (`&name`, `*name`) існують і розгортаються
  при читанні, що іноді корисно, а частіше дивує.
- **Ненадійний YAML небезпечно парсити в `any`.** Надавайте перевагу
  структурі й встановлюйте ліміт розміру вхідних даних.

## TOML

Плоскіший, без значущих пробілів, однозначні скаляри:

```toml
[server]
host = "t.example"
port = 9090
```

```go
var d Doc
err := toml.Unmarshal([]byte(src), &d)
// {Host:t.example Port:9090}
```

Помилки називають поле й типи, що беруть участь:

```go
toml.Unmarshal([]byte("[server]\nport = \"nope\"\n"), &d)
// toml: cannot decode TOML string into struct field main.Server.Port of type int
```

### TOML не парсить тривалість

Це відмінність, яка спіймає вас під час перенесення структури між
двома форматами:

```go
// timeout = "5s"
// toml: cannot decode TOML string into struct field
//       main.Server.Timeout of type time.Duration
```

А маршалінг тривалості видає сире ціле число наносекунд замість `5s`.
Якщо структура спільна для YAML і TOML, або оголосіть поле як `string`
і парсіть самостійно через `time.ParseDuration`, або дайте типу власні
`UnmarshalText`/`MarshalText`.

У TOML є власні дати й час, тоді як YAML лише наближено це підтримує.

## Що вибрати

| Використання | Формат |
|---|---|
| Kubernetes, CI, усе в цій екосистемі | YAML — реального вибору немає |
| власний файл конфігурації інструменту | TOML — менше способів помилитися |
| API | JSON — він у стандартній бібліотеці |

Перевага TOML у тому, що в ньому немає неоднозначних скалярів і правил
відступу, тож файл, відредагований вручну, важче зламати. Перевага
YAML у тому, що все інше вже говорить цією мовою.

Що б ви не обрали: декодуйте в **структуру**, а не в
`map[string]any`, і валідуйте після декодування. Обидва формати
охоче віддадуть вам синтаксично правильний документ, що нічого не
означає.

> **З досвіду Python:** `yaml.Unmarshal` — це `yaml.safe_load` у
> dataclass, і тут немає небезпечного завантажувача, до якого можна
> випадково потягнутися. `KnownFields(true)` — та сама строгість, яку
> дає `extra="forbid"` у pydantic, і тут вона теж вимкнена за
> замовчуванням.

## Швидка довідка

| Задача | Форма |
|---|---|
| декодувати YAML | `yaml.Unmarshal(b, &v)` |
| закодувати YAML | `yaml.Marshal(v)` — відступ у чотири пробіли |
| відхилити одруківки | `dec := yaml.NewDecoder(r); dec.KnownFields(true)` |
| тривалості в YAML | працюють нативно з `30s` |
| декодувати TOML | `toml.Unmarshal(b, &v)` |
| тривалості в TOML | **не підтримуються** — використовуйте рядок або `UnmarshalText` |
| теги | `yaml:"name,omitempty"`, `toml:"name"`, `-` щоб пропустити |
| кілька форматів | накладіть теги на одне поле |
| завжди | декодуйте в структуру, потім валідуйте |

## Джерела

- [`gopkg.in/yaml.v3` — pkg.go.dev/gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)
- [`go-toml/v2` — pkg.go.dev/github.com/pelletier/go-toml/v2](https://pkg.go.dev/github.com/pelletier/go-toml/v2)
- [YAML 1.2 specification — yaml.org/spec/1.2.2/](https://yaml.org/spec/1.2.2/)
- [TOML specification — toml.io/en/v1.0.0](https://toml.io/en/v1.0.0)
- [`time.ParseDuration` — pkg.go.dev/time#ParseDuration](https://pkg.go.dev/time#ParseDuration)
