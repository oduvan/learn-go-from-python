# OIDC та OAuth

Делегування логіну постачальнику ідентичності, і обробка сесії та
API-токенів навколо цього. Криптографія — зі стандартної бібліотеки —
[хешування та випадкові значення](../07-operating-system/07-hashing-and-random-values.md)
покриває примітиви; ось як вони складаються разом.

> **Модулі:** `golang.org/x/oauth2` та
> `github.com/coreos/go-oidc/v3`.

```go
conf := &oauth2.Config{
    ClientID:    clientID,
    RedirectURL: "https://app.example/callback",
    Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
    Endpoint:    provider.Endpoint(),
}
```

## Ніколи не будуйте це самі

Два правила перед усім іншим. **Не реалізуйте постачальника
ідентичності**, і **не зберігайте паролі**, якщо це не ваш продукт.
Делегування OIDC-постачальнику означає, що ви ніколи не тримаєте
обліковий дані, які можна вкрасти у вас.

Те, що ви реалізуєте — це клієнтська сторона, і саме там, нижче,
трапляються помилки на клієнтській стороні.

## Потік authorization code

1. Користувач заходить на захищену сторінку; ви перенаправляєте його
   до постачальника з `state` і PKCE-викликом.
2. Він автентифікується там.
3. Постачальник перенаправляє назад із `code`.
4. Ви обмінюєте код на токени, **сервер-до-сервера**.
5. Ви перевіряєте ID-токен і створюєте власну сесію.

```go
provider, err := oidc.NewProvider(ctx, "https://idp.example")
```

`NewProvider` завантажує документ discovery, тож ендпоінти й ключі
підпису йдуть від постачальника, а не з вашої конфігурації. Робіть це
один раз при старті — це мережевий виклик.

## `state` і PKCE не опціональні

```go
url := conf.AuthCodeURL("state-xyz",
    oauth2.SetAuthURLParam("code_challenge", challenge),
    oauth2.SetAuthURLParam("code_challenge_method", "S256"))
```

```
https://idp.example/authorize
  has client_id              true
  has state                  true
  has code_challenge_method  true
  has scope                  true
  has response_type          true
```

**`state`** — це випадкове значення, яке ви зберігаєте (у
короткоживучій кукі) і порівнюєте при зворотному виклику. Без нього
атакувальник може підкинути браузеру жертви власний код авторизації —
логін-CSRF, що заводить жертву в акаунт атакувальника.

**PKCE** доводить, що клієнт, який обмінює код, — той самий, що почав
потік. Згенеруйте випадковий verifier, надішліть його SHA-256-хеш як
виклик, а сам verifier — на момент обміну:

```go
func pkce() (verifier, challenge string) {
    b := make([]byte, 32)
    rand.Read(b)
    verifier = base64.RawURLEncoding.EncodeToString(b)
    s := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(s[:])
    return
}
```

```go
// verifier len: 43   challenge len: 43
// challenge verifies: true
// no padding chars:   true
```

`RawURLEncoding` важливий: специфікація вимагає base64url без
доповнення, а `StdEncoding` видає `+`, `/` і `=` — усі вони ламаються
в URL. Це та сама думка про кодування зі статті про хешування, з
конкретним наслідком.

PKCE спочатку призначався для мобільних застосунків. Тепер його
рекомендують для всіх клієнтів, включно з конфіденційними.

## Обмін і перевірка

```go
tok, err := conf.Exchange(ctx, code,
    oauth2.SetAuthURLParam("code_verifier", verifier))

rawID, ok := tok.Extra("id_token").(string)
if !ok {
    return errors.New("no id_token in response")
}

verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
idToken, err := verifier.Verify(ctx, rawID)

var claims struct {
    Email    string `json:"email"`
    Name     string `json:"name"`
    Verified bool   `json:"email_verified"`
}
if err := idToken.Claims(&claims); err != nil {
    return err
}
```

**Завжди `Verify`.** Він перевіряє підпис за опублікованими ключами
постачальника, видавця, аудиторію й термін дії. JWT читабельний
будь-ким; декодування його без перевірки означає довіру всьому, що
надіслав браузер.

Перевіряйте `email_verified`, перш ніж довіряти email як
ідентичності. Деякі постачальники охоче засвідчать неперевірену
адресу, а за цим слідує захоплення акаунта.

## Ваша власна сесія

Токени постачальника — для постачальника. Видайте власну сесію й
тримайте її непрозорою:

```go
b := make([]byte, 32)
rand.Read(b)
sessionID := base64.RawURLEncoding.EncodeToString(b)
```

Зберігайте сесію на боці сервера, з ключем за цим id. Тоді логаут,
відкликання й зміна ролей — це одне видалення.

```go
http.SetCookie(w, &http.Cookie{
    Name:     "session",
    Value:    sessionID,
    Path:     "/",
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteLaxMode,
    MaxAge:   int((24 * time.Hour).Seconds()),
})
```

`HttpOnly` не пускає JavaScript, `Secure` тримає її подалі від
звичайного HTTP, `SameSite=Lax` притупляє CSRF.

### Підписані кукі, якщо треба залишатись без стану

```go
func sign(payload string) string {
    m := hmac.New(sha256.New, signingKey)
    m.Write([]byte(payload))
    return payload + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func verify(tok string) (string, bool) {
    payload, sig, ok := strings.Cut(tok, ".")
    if !ok {
        return "", false
    }
    m := hmac.New(sha256.New, signingKey)
    m.Write([]byte(payload))
    want := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
    if subtle.ConstantTimeCompare([]byte(sig), []byte(want)) != 1 {
        return "", false
    }
    return payload, true
}
```

```go
tok := sign("user:ada")
got, ok := verify(tok)          // "user:ada" true

tampered := strings.Replace(tok, "ada", "bob", 1)
_, ok = verify(tampered)        // false
```

Порівняння за постійний час обов'язкове, а не декоративне — звичайне
`==` розкриває, скільки саме символів підпису було правильним.

Підпис — не шифрування: корисне навантаження читабельне. А ціна
відсутності стану в тому, що ви не можете відкликати сесію, тож
тримайте термін дії коротким.

## API-токени

Для машинних викликачів видавайте випадковий токен і **зберігайте
лише його хеш**:

```go
func newToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return "snx_" + base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(t string) string {
    s := sha256.Sum256([]byte(t))
    return base64.RawURLEncoding.EncodeToString(s[:])
}
```

```go
// prefix: true   len: 47
// hash stable: true   hash != token: true
```

Покажіть токен один раз, зберігайте хеш. Витеклий дамп бази даних тоді
не містить нічого придатного для використання.

Звичайний SHA-256 тут правильний і водночас неправильний для паролів:
256-бітний випадковий токен не піддається перебору, тож повільне
хешування, що захищає слабкий пароль, тут нічого не дає й лише додає
затримку до кожного запиту.

Префікс `snx_` допомагає сканерам секретів помітити витеклий токен у
репозиторії — варто робити.

Порівнюйте при пошуку через `subtle.ConstantTimeCompare`, і давайте
токенам термін дії та прапорець відкликання.

## Refresh-токени

`oauth2.Config.TokenSource` оновлює автоматично, коли токен спливає:

```go
client := conf.Client(ctx, tok)   // refreshes as needed
```

Refresh-токени — довгоживучі облікові дані — зберігайте їх зашифрованими
й підтримуйте ротацію, де кожне оновлення видає новий і скасовує
старий.

## Якщо ви — сервер авторизації

Усе вище — це *клієнтська* сторона. Іноді ви — інший кінець —
найчастіше, коли щось має автентифікуватись до *вашого* API від імені
користувача, що якраз потрібно MCP-серверу, виставленому через HTTP.

Тоді елементи міняються місцями:

- **`/.well-known/oauth-authorization-server`** і
  `/.well-known/oauth-protected-resource` — документи discovery, тож
  клієнт може знайти ваші ендпоінти без конфігурації.
- **`GET /authorize`** — автентифікуйте користувача, потім
  перенаправте назад із короткоживучим кодом. Зберігайте код разом із
  id клієнта, redirect URI й PKCE-викликом.
- **`POST /token`** — обмін коду. Перевірте, що redirect URI
  збігається **точно** з тим, що було зареєстровано, і перевірте
  PKCE-verifier: перерахуйте `base64url(sha256(verifier))` і
  порівняйте за постійний час зі збереженим викликом.
- **`POST /register`** — динамічна реєстрація клієнта, якщо клієнти не
  налаштовані заздалегідь.

Три правила несуть на собі більшу частину безпеки. Коди авторизації —
**одноразові** й короткоживучі, тож повторне використання одного має
скасувати сесію, а не видати другий токен. Redirect URI звіряються з
**allowlist**, ніколи за префіксом — збіг за префіксом дозволяє
атакувальнику дописати шлях і отримати код. А refresh-токени мають
**ротуватись**: кожне оновлення видає новий і скасовує попередника,
тож викрадений токен можна виявити, коли легітимний клієнт пред'явить
старий.

Відповіді про помилки тут дотримуються RFC 6749, а не вашої звичної
форми помилки — тіло JSON з `error` і `error_description`, та
конкретні коди на кшталт `invalid_grant`. Клієнти це парсять, тож це
одне з місць, де власний хелпер помилок — неправильний вибір.

## Чек-лист

- `state` на кожному потоці, порівнюється при зворотному виклику
- PKCE з `S256`
- `Verify` ID-токен — ніколи не декодувати й довіряти
- перевіряти `email_verified`
- кукі `HttpOnly`, `Secure`, `SameSite`
- непрозорі id сесії, збережені на боці сервера
- API-токени хешовані в спокої, порівнювані за постійний час
- обмінюйте коди лише на боці сервера; клієнтський секрет ніколи не
  доходить до браузера

> **З досвіду Python:** `oauth2.Config` — це клієнт Authlib, а
> `go-oidc` робить discovery й перевірку JWT, які робить OIDC-міксин
> Authlib. Тут нічого не автоматичне — `state` і PKCE — параметри, які
> ви передаєте, що робить їх легкими для пропуску й важливими не
> пропустити.

## Швидка довідка

| Задача | Форма |
|---|---|
| discovery | `oidc.NewProvider(ctx, issuer)` один раз при старті |
| URL перенаправлення | `conf.AuthCodeURL(state, pkceParams...)` |
| CSRF | випадковий `state`, збережений і порівняний |
| PKCE | SHA-256 випадкового verifier, `RawURLEncoding`, `S256` |
| обмін | `conf.Exchange(ctx, code, code_verifier)` — лише на боці сервера |
| **перевірка** | `provider.Verifier(...).Verify(ctx, rawIDToken)` |
| claims | `idToken.Claims(&struct)`, перевірити `email_verified` |
| сесія | 32 випадкові байти, на боці сервера, `HttpOnly`+`Secure`+`SameSite` |
| сесія без стану | HMAC-SHA256 + `subtle.ConstantTimeCompare` |
| API-токени | випадкові, з префіксом, **хешовані в спокої** |
| оновлення | `conf.Client(ctx, tok)` |

## Джерела

- [`golang.org/x/oauth2` — pkg.go.dev/golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
- [`go-oidc` — pkg.go.dev/github.com/coreos/go-oidc/v3/oidc](https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc)
- [OAuth 2.0 for browser-based apps — datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps)
- [PKCE, RFC 7636 — datatracker.ietf.org/doc/html/rfc7636](https://datatracker.ietf.org/doc/html/rfc7636)
- [OpenID Connect Core — openid.net/specs/openid-connect-core-1_0.html](https://openid.net/specs/openid-connect-core-1_0.html)
- [`http.Cookie` — pkg.go.dev/net/http#Cookie](https://pkg.go.dev/net/http#Cookie)
