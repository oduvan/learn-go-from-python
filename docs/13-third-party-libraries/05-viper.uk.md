# viper

Конфігурація зі змінних середовища, файлів, значень за замовчуванням і
прапорців, зібрана в одному місці. Стаття про
[патерни конфігурації](../11-architecture-and-conventions/03-configuration-patterns.md)
будувала це вручну через `reflect`; viper — те саме, з форматами
файлів і пріоритетністю включно.

> **Модуль:** `github.com/spf13/viper`.

```go
v := viper.New()
v.AutomaticEnv()
v.SetDefault("PORT", 8080)

var cfg Config
err := v.Unmarshal(&cfg)
```

## Використовуйте екземпляр, а не пакет

`viper.Get` та подібні працюють із синглтоном на рівні пакета —
глобальним змінюваним станом, спільним із кожною бібліотекою, що також
використовує viper. Створіть власний:

```go
v := viper.New()
```

Потім передавайте отриману структуру, а не сам екземпляр viper.
Конфігурацію варто прочитати один раз і перетворити на звичайне
значення.

## Теги `mapstructure`, не `json`

viper декодує через `mapstructure`, тож саме цей тег він читає:

```go
type Config struct {
    Environment string        `mapstructure:"ENVIRONMENT"`
    Port        int           `mapstructure:"PORT"`
    Timeout     time.Duration `mapstructure:"TIMEOUT"`
    DBHost      string        `mapstructure:"DB_HOST"`
}
```

Тег `json:` тут нічого не робить. Імена полів зіставляються без
урахування регістру навіть без тегу, але будьте явними — `DBHost` і
`DB_HOST` самі по собі не зіставляться.

Поля тривалості працюють: типові хуки декодування viper викликають
`time.ParseDuration`, тож `TIMEOUT=45s` потрапляє як справжній
`time.Duration`.

## Пастка: `AutomaticEnv` не наповнює `Unmarshal`

Ця коштує людям цілого вечора. `AutomaticEnv()` змушує `Get` звертатися
до середовища за потреби:

```go
v.AutomaticEnv()
fmt.Printf("Get=%q IsSet=%v\n", v.GetString("ONLY_ENV"), v.IsSet("ONLY_ENV"))
// output: Get="zzz" IsSet=true
```

Але `Unmarshal` перебирає ключі, про які viper *знає*, а `AutomaticEnv`
не реєструє жодного — він лише перехоплює звернення. Тому:

```go
var c struct {
    OnlyEnv string `mapstructure:"ONLY_ENV"`
}
v.Unmarshal(&c)
// c.OnlyEnv == ""
```

`Get` повертає значення, а `Unmarshal` залишає поле порожнім — і без
жодної помилки в обох випадках.

Виправлення — зробити кожен ключ відомим, через `BindEnv` або
`SetDefault`:

```go
for _, k := range []string{"ENVIRONMENT", "PORT", "TIMEOUT", "DB_HOST"} {
    v.BindEnv(k)
}

var cfg Config
err := v.Unmarshal(&cfg)
// {Environment:prod Port:7070 Timeout:45s DBHost:db.host} err=<nil>
```

Замість того, щоб підтримувати цей список вручну, згенеруйте його з
тегів структури — той самий обхід через reflect, що й у базовій статті:

```go
func bindEnvs(v *viper.Viper, cfg any) {
    t := reflect.TypeOf(cfg)
    for i := range t.NumField() {
        if tag := t.Field(i).Tag.Get("mapstructure"); tag != "" && tag != "-" {
            _ = v.BindEnv(tag)
        }
    }
}
```

Тепер додати поле до структури — це все, що потрібно.

## Значення за замовчуванням і файли

```go
v.SetDefault("ENVIRONMENT", "local")
v.SetDefault("PORT", 8080)
v.SetDefault("TIMEOUT", "5s")
```

`SetDefault` також реєструє ключ, тож ці поля не потребують `BindEnv`.

```go
v.SetConfigFile(".env")
v.SetConfigType("env")
if err := v.ReadInConfig(); err != nil {
    var notFound viper.ConfigFileNotFoundError
    if !errors.As(err, &notFound) {
        return err          // a real parse error
    }
    // absent is fine
}
```

Розрізняйте "файлу немає" і "файл зламаний". Ставлення до кожної
помилки як до "відсутнього" означає, що зіпсована конфігурація мовчки
працюватиме на значеннях за замовчуванням.

`MergeInConfig` накладає другий файл поверх першого — саме так
`.env.local` перевизначає `.env`.

## Пріоритетність

Найвищий пріоритет перемагає:

1. `v.Set()` — явне перевизначення
2. прив'язаний прапорець
3. середовище
4. файл конфігурації
5. `SetDefault`

Передбачувано, коли ви це знаєте, і непомітно, якщо ні — тож запишіть
це у своєму README.

## Чи варта вона цієї залежності

Будьте чесними щодо цього обміну.

**За неї:** кілька форматів файлів, живе перезавантаження, віддалені
бекенди, інтеграція з прапорцями. Якщо вам потрібно два з них, viper
заслуговує на своє місце.

**Проти неї:** велике дерево залежностей заради того, що часто
вкладається у сорок рядків, `Unmarshal`, що мовчки провалюється на
незареєстрованих ключах, і слабо типізований API, де одруківка в ключі
дає нульове значення, а не помилку компіляції.

Для сервісу, що читає двадцять змінних середовища, завантажувач,
написаний вручну, з базової статті менший, повністю типізований і
провалюється голосно. Тягніться до viper, коли вам справді потрібні
шаровані файли й формати.

Що б ви не використовували, правила з базової статті все ще діють:
завантажуйте один раз при старті, валідуйте через `errors.Join`,
провалюйтесь до початку обслуговування запитів, і ніколи не логуйте
структуру.

> **З досвіду Python:** viper — це `dynaconf` — шаровані джерела з
> пріоритетністю — тоді як версія, написана вручну, ближча до
> `pydantic-settings`. Той самий обмін в обох мовах: гнучкість проти
> схеми, яку може перевірити компілятор.

## Швидка довідка

| Задача | Форма |
|---|---|
| екземпляр | `viper.New()`, ніколи глобальні змінні пакета |
| теги структури | `mapstructure:"KEY"` |
| читати середовище через `Get` | `v.AutomaticEnv()` |
| **зробити видимим для `Unmarshal`** | `v.BindEnv(key)` або `v.SetDefault(key, …)` |
| прив'язати кожне поле | reflect над тегами `mapstructure` |
| файл | `SetConfigFile` + `SetConfigType` + `ReadInConfig` |
| відсутній чи зламаний | `errors.As(err, &viper.ConfigFileNotFoundError{})` |
| накладати файли | `MergeInConfig` |
| у структуру | `v.Unmarshal(&cfg)` — тривалості парсяться нативно |
| потім | валідуйте, потім передавайте структуру, не viper |

## Джерела

- [viper — pkg.go.dev/github.com/spf13/viper](https://pkg.go.dev/github.com/spf13/viper)
- [viper README — github.com/spf13/viper](https://github.com/spf13/viper)
- [`mapstructure` — pkg.go.dev/github.com/go-viper/mapstructure/v2](https://pkg.go.dev/github.com/go-viper/mapstructure/v2)
- [The twelve-factor app: config — 12factor.net/config](https://12factor.net/config)
