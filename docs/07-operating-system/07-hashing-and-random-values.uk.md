# Хешування та випадкові значення

Збірна солянка з невеликих пакетів стандартної бібліотеки, які
трапляються постійно: хешування, підписування, генерація токенів,
кодування байтів у текст і gzip. Жоден не складний, і кожен має один
спосіб зробити неправильно.

```go
sum := sha256.Sum256([]byte("hello"))
fmt.Println(hex.EncodeToString(sum[:]))
// output: 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
```

## Хешування

`sha256.Sum256` — одноразова форма. Вона повертає **масив фіксованого
розміру**, `[32]byte`, тож передача його будь-куди, де очікується зріз,
потребує `sum[:]`:

```go
sum := sha256.Sum256([]byte("hello"))
fmt.Println(len(sum))       // output: 32
fmt.Printf("%x\n", sum)     // output: 2cf24dba...938b9824
```

`%x` на масиві працює напряму — найкоротший спосіб надрукувати дайджест.

Для даних, які ви не хочете тримати в пам'яті, хеш — це `io.Writer`:

```go
h := sha256.New()
io.Copy(h, r)
fmt.Println(hex.EncodeToString(h.Sum(nil)))
```

`h.Sum(nil)` додає дайджест до переданого зрізу — `nil` означає "дай
мені новий". Це не скидає хеш.

Ось як обчислити контрольну суму файлу, не читаючи його весь одразу, і
як `io.Copy` зі статті [читачі та письменники](02-readers-and-writers.md)
знову себе виправдовує.

**SHA-256 не для паролів.** Він швидкий, а це саме те, що неправильно
для пароля. Використовуйте навмисно повільну функцію — bcrypt чи
Argon2 — обидві живуть поза стандартною бібліотекою.

## HMAC: доведення, що повідомлення від вас

Простий хеш доводить цілісність. HMAC доводить *автентичність*, бо для
його створення потрібен ключ:

```go
key := []byte("secret")

m := hmac.New(sha256.New, key)
m.Write([]byte("message"))
sig := m.Sum(nil)
```

Перевіряйте, обчислюючи знову й порівнюючи через `hmac.Equal`:

```go
m2 := hmac.New(sha256.New, key)
m2.Write([]byte("message"))
fmt.Println(hmac.Equal(sig, m2.Sum(nil)))   // output: true
```

**Використовуйте `hmac.Equal`, ніколи `==` чи `bytes.Equal`.** Звичайне
порівняння повертається, щойно два байти відрізняються, тож те, скільки
часу воно зайняло, розкриває, скільки підпису було правильним —
достатньо, за багато спроб, щоб відновити дійсний. `hmac.Equal` завжди
займає однаковий час.

Це механізм за підписаними cookie й перевіркою вхідних вебхуків:
відправник підписує спільним секретом, ви обчислюєте знову й
порівнюєте.

Для секретів, що не є HMAC, — токена API, спільного ключа — загальна
форма така:

```go
fmt.Println(subtle.ConstantTimeCompare([]byte("abc"), []byte("abc")))   // output: 1
fmt.Println(subtle.ConstantTimeCompare([]byte("abc"), []byte("abd")))   // output: 0
```

Вона повертає `int`, а не `bool` — `1` означає рівність. Зауважте, вона
повертає `0` для входів різної довжини взагалі без порівняння, тож
довжина не захищена.

## Випадкові значення: `crypto/rand`, а не `math/rand`

Два пакети, і вибір неправильного — це вразливість безпеки, а не
питання стилю.

| Пакет | Для |
|---|---|
| `math/rand/v2` | симуляції, джитер, перемішування, тестові дані |
| `crypto/rand` | токени, ідентифікатори сесій, ключі, nonce, солі |

`math/rand` передбачуваний за жменькою виводів. Усе, що зловмисник не
повинен вгадати, береться з `crypto/rand`:

```go
buf := make([]byte, 32)
n, err := rand.Read(buf)
fmt.Println(n, err)   // output: 32 <nil>
```

Тридцять два байти — звичний розмір для токена сесії — 256 бітів
ентропії.

## Кодування байтів у текст

Випадкові байти не друковані, тож їх кодують. Варіант має значення:

```go
raw := []byte{0xfb, 0xff, 0x01}

fmt.Println(base64.StdEncoding.EncodeToString(raw))       // output: +/8B
fmt.Println(base64.URLEncoding.EncodeToString(raw))       // output: -_8B
fmt.Println(base64.RawURLEncoding.EncodeToString(raw))    // output: -_8B
```

`StdEncoding` дає `+`, `/` і `=`, усі з яких потребують екранування в
URL чи cookie. `URLEncoding` заміняє на `-` і `_`; варіанти `Raw`
прибирають доповнення `=`. Для всього, що подорожує в URL, заголовку чи
cookie, використовуйте `RawURLEncoding`:

```go
tok := base64.RawURLEncoding.EncodeToString(buf)
fmt.Println(len(tok))   // output: 43
```

32 байти стають 43 символами без доповнення й без нічого для
екранування.

`encoding/hex` — інший варіант — удвічі довший, але однозначний і
прийнятий для дайджестів.

## `hash/fnv` для некриптографічного хешування

Коли потрібне число, похідне від рядка — індекс шарду, ключ блокування,
кошик кешу — криптографічний хеш надлишковий. FNV швидкий і
детермінований у різних запусках і машинах:

```go
f := fnv.New64a()
f.Write([]byte("job-name"))
fmt.Println(f.Sum64())   // output: 17796606368501603464
```

Ця детермінованість — уся суть: вбудоване хешування мап у Go
рандомізоване для кожного процесу, тож його не можна використати, щоб
домовитись про щось між процесами. FNV — можна.

Ніколи не використовуйте його там, де зловмисник обирає ввід — навмисно
створити колізії тривіально.

## gzip

`compress/gzip` обгортає письменник і читач, тож він складається з
усім іншим:

```go
var buf bytes.Buffer
zw := gzip.NewWriter(&buf)
zw.Write(data)
zw.Close()          // обов'язково: записує футер
```

`Close` не опційний — він скидає останній блок і записує футер gzip.
Потік без нього обрізаний і не розпакується. Читання назад:

```go
zr, err := gzip.NewReader(&buf)
if err != nil {
    return err
}
out, err := io.ReadAll(zr)
if err != nil {
    return err
}
```

> **З досвіду Python:** `hashlib.sha256().hexdigest()` стає
> `hex.EncodeToString(sum[:])`, `hmac.compare_digest` стає
> `hmac.Equal`, а `secrets.token_urlsafe(32)` стає `crypto/rand` плюс
> `base64.RawURLEncoding`. Те саме правило діє в обох мовах: `random`
> для симуляцій, `secrets` для безпеки.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| хешувати зріз | `sha256.Sum256(b)` → `[32]byte`, використовуйте `sum[:]` |
| хешувати потік | `h := sha256.New()`, `io.Copy(h, r)`, `h.Sum(nil)` |
| дайджест як текст | `hex.EncodeToString(...)` або `%x` |
| підписати | `hmac.New(sha256.New, key)` |
| перевірити підпис | `hmac.Equal(a, b)` — ніколи `==` |
| порівняти секрет | `subtle.ConstantTimeCompare(a, b) == 1` |
| невгадувані байти | `crypto/rand.Read(buf)` |
| текст токена | `base64.RawURLEncoding.EncodeToString(buf)` |
| симуляції, джитер | `math/rand/v2` |
| детермінований ключ кошика | `fnv.New64a()` |
| стиснути | `gzip.NewWriter(w)` — **обов'язково `Close()`** |
| паролі | не тут; bcrypt чи Argon2 |

## Джерела

- [`crypto/sha256` — pkg.go.dev/crypto/sha256](https://pkg.go.dev/crypto/sha256)
- [`crypto/hmac` — pkg.go.dev/crypto/hmac](https://pkg.go.dev/crypto/hmac)
- [`crypto/subtle` — pkg.go.dev/crypto/subtle](https://pkg.go.dev/crypto/subtle)
- [`crypto/rand` — pkg.go.dev/crypto/rand](https://pkg.go.dev/crypto/rand)
- [`encoding/base64` — pkg.go.dev/encoding/base64](https://pkg.go.dev/encoding/base64)
- [`hash/fnv` — pkg.go.dev/hash/fnv](https://pkg.go.dev/hash/fnv)
- [`compress/gzip` — pkg.go.dev/compress/gzip](https://pkg.go.dev/compress/gzip)
