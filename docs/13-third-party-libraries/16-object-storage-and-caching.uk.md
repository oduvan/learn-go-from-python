# Об'єктне сховище та кешування

Дві інфраструктурні залежності з однією спільною ідеєю: поставити
інтерфейс перед ними, тож локальна розробка не потребує жодної з них, і
тести теж.

> **Модулі:** `github.com/aws/aws-sdk-go-v2` (з `config` та
> `service/s3`) та `github.com/valkey-io/valkey-go`.

```go
type ObjectStore interface {
    Put(ctx context.Context, key string, r io.Reader) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

## Визначте інтерфейс, не клієнт

Обробники не мають знати, чи файл у S3, чи на диску. Інтерфейс вище —
весь контракт, і він навмисно крихітний — та сама аргументація зі
статті про [патерн репозиторію](../09-database-sql/04-the-repository-pattern.md).

Зауважте, він виражений через `io.Reader` та `io.ReadCloser`, тож
викликач може стрімити завантаження на 2 ГБ без того, щоб воно
проходило через пам'ять.

Дайте йому доменну помилку:

```go
var ErrObjectNotFound = errors.New("object not found")
```

## Реалізація на файловій системі

Локальна реалізація коротка, і саме вона змушує розробку й тести
працювати взагалі без хмарних облікових даних:

```go
type localStore struct{ root string }

func (s localStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
    f, err := os.Open(s.path(key))
    if errors.Is(err, os.ErrNotExist) {
        return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
    }
    return f, err
}
```

```go
st.Put(ctx, "a/b/file.txt", strings.NewReader("payload"))
rc, err := st.Get(ctx, "a/b/file.txt")   // "payload"

_, err = st.Get(ctx, "missing")
errors.Is(err, ErrObjectNotFound)        // true
st.Delete(ctx, "missing")                // nil — deleting what is absent is fine
```

Дві поведінки, які мають збігатися між реалізаціями: відсутній об'єкт
повертає ваш сентинел, а видалення відсутнього об'єкта — **не**
помилка. Якщо це розходиться, заміна реалізації ламає викликачів.

Застосуйте `filepath.FromSlash` до ключа й відхиляйте `..`, або ключ
від користувача втече за межі вашого кореневого каталогу.

## Реалізація на S3

```go
cfg, err := config.LoadDefaultConfig(ctx)
client := s3.NewFromConfig(cfg)
```

`LoadDefaultConfig` проходить стандартний ланцюжок облікових даних —
середовище, спільний файл конфігурації, роль інстансу — тож продакшн
не потребує змін коду. Для локального ендпоінта MinIO чи Garage
перевизначте базовий URL в опціях.

Перекладіть типізовані помилки SDK у власні:

```go
out, err := c.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &key})
if err != nil {
    var nsk *types.NoSuchKey
    var nf *types.NotFound
    if errors.As(err, &nsk) || errors.As(err, &nf) {
        return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
    }
    return nil, fmt.Errorf("getting %s: %w", key, err)
}
return out.Body, nil
```

Обидва типи важливі — який ви отримаєте, залежить від операції, а
перевірка лише `NoSuchKey` пропускає `HeadObject`.

Ще дві речі варто знати. `manager.NewUploader` обробляє multipart-
завантаження для великих об'єктів, чого не робить простий `PutObject`.
А **presigned URL** дозволяє клієнту завантажувати чи скачувати
напряму, тож великий файл узагалі не проходить через ваш сервіс:

```go
ps := s3.NewPresignClient(client)
req, err := ps.PresignGetObject(ctx, in, s3.WithPresignExpires(15*time.Minute))
```

## Вибір реалізації при старті

```go
func NewObjectStore(cfg Config) ObjectStore {
    if cfg.S3Bucket == "" {
        return localStore{root: cfg.LocalStoragePath}
    }
    return s3Store{client: s3.NewFromConfig(awsCfg), bucket: cfg.S3Bucket}
}
```

Одне рішення, в одному місці. Усе нижче за течією тримає інтерфейс.

## Кешування з valkey

Valkey — форк Redis; `valkey-go` використовує білдер команд замість
стрінгово-типізованих аргументів:

```go
c, err := valkey.NewClient(valkey.ClientOption{
    InitAddress: []string{"127.0.0.1:6379"},
})
defer c.Close()

err = c.Do(ctx, c.B().Set().Key("k").Value("v").Ex(30*time.Second).Build()).Error()

got, err := c.Do(ctx, c.B().Get().Key("k").Build()).ToString()
// "v"
```

`c.B()` будує команду з аргументами, перевіреними на етапі компіляції,
тож одруківка в `SET` — помилка збірки, а не помилка часу виконання.
Клієнт безпечний для конкурентного використання й автоматично
конвеєризує запити.

### Промах — це значення помилки

```go
_, err := c.Do(ctx, c.B().Get().Key("absent").Build()).ToString()
fmt.Println(valkey.IsValkeyNil(err))   // output: true
// valkey nil message
```

Промах кешу повертається як помилка, і це не провал. Перевіряйте це
явно:

```go
val, err := c.Do(ctx, c.B().Get().Key(k).Build()).ToString()
switch {
case valkey.IsValkeyNil(err):
    return compute()          // miss
case err != nil:
    return compute()          // cache broken — still serve the request
default:
    return val, nil
}
```

**Завжди встановлюйте TTL.** `Ex(30*time.Second)` при записі; кеш без
терміну придатності — це витік пам'яті з зайвими кроками.

### Pub/sub

```go
sub, cancel := c.Dedicate()
defer cancel()

go sub.Receive(ctx, sub.B().Subscribe().Channel("events").Build(),
    func(m valkey.PubSubMessage) {
        // handle m.Message
    })

c.Do(ctx, c.B().Publish().Channel("events").Message("hello").Build())
// received: hello
```

Підписка потребує **виділеного з'єднання** — `Dedicate()` — бо
підписане з'єднання не може обслуговувати інші команди.

Саме так [події, надіслані сервером](../08-http-with-net-http/05-server-sent-events.md)
працюють між репліками: клієнт підключений до одного інстансу, тож
подія, викликана на іншому, дістанеться його лише якщо інстанси
поділяють шину. Публікуйте в канал, підписник кожної репліки отримує
це, і кожна пересилає своїм власним підключеним клієнтам.

Зауважте, pub/sub — це **надіслав і забув**. Репліка, що вимкнена,
повністю пропускає повідомлення. Для всього, що не можна втратити,
використовуйте справжню чергу.

## Провалюйтесь м'яко

Жодна з цих двох залежностей не має класти ваш сервіс:

```go
cache, err := valkey.NewClient(opt)
if err != nil {
    slog.Warn("cache unavailable, continuing without it", "error", err)
    cache = nil        // callers check, or use a no-op implementation
}
```

Недосяжність кешу має означати повільніші відповіді, а не помилки.
No-op реалізація інтерфейсу чистіша за перевірки на nil всюди —
кожен промах, кожен запис — успіх, а решта коду про це навіть не
знає.

Об'єктне сховище зазвичай навпаки: якщо завантаження — це продукт,
провал голосно при старті — правильний вибір. Вирішуйте свідомо, до
якого з двох типів належить кожна залежність.

> **З досвіду Python:** клієнт S3 — це boto3 з явними типами помилок
> замість `ClientError` плюс рядковий код, а `valkey-go` — це `redis-py`
> з API білдера. Звичка інтерфейс-плюс-локальна-реалізація — та сама,
> яку ви б отримали від moto чи fakeredis, тільки це ваш власний код, а
> не шар мокання.

## Швидка довідка

| Задача | Форма |
|---|---|
| абстракція | невеликий інтерфейс `ObjectStore` над `io.Reader` |
| локальна розробка | реалізація на файловій системі, без облікових даних |
| клієнт S3 | `config.LoadDefaultConfig` + `s3.NewFromConfig` |
| відсутній об'єкт | `errors.As` для `*types.NoSuchKey` **і** `*types.NotFound` |
| великі завантаження | `manager.NewUploader` |
| пряма передача клієнту | `s3.NewPresignClient` |
| клієнт кешу | `valkey.NewClient`, команди через `c.B()` |
| промах | `valkey.IsValkeyNil(err)` — не провал |
| термін придатності | завжди `.Ex(d)` |
| підписка | `c.Dedicate()` — підписане з'єднання виключне |
| події між репліками | публікуйте в канал; кожна репліка розсилає далі |
| коли недоступно | деградуйте для кешу, провалюйтесь голосно для сховища |

## Джерела

- [aws-sdk-go-v2 S3 — pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3)
- [AWS SDK for Go v2 — aws.github.io/aws-sdk-go-v2/docs/](https://aws.github.io/aws-sdk-go-v2/docs/)
- [`valkey-go` — pkg.go.dev/github.com/valkey-io/valkey-go](https://pkg.go.dev/github.com/valkey-io/valkey-go)
- [Valkey commands — valkey.io/commands/](https://valkey.io/commands/)
- [Redis pub/sub — redis.io/docs/latest/develop/interact/pubsub/](https://redis.io/docs/latest/develop/interact/pubsub/)
