# Транзакції

Транзакція змушує кілька інструкцій виконатись або провалитись разом.
API складається із трьох методів; головне тут — переконатись, що
відкат (rollback) завжди відбувається, зокрема і при паніці.

```go
tx, err := db.BeginTx(ctx, nil)
defer tx.Rollback()   // нічого не робить після коміту

// ... інструкції на tx ...

return tx.Commit()
```

## `*sql.Tx` — це одне з'єднання

`BeginTx` бере з'єднання з пулу і утримує його, доки ви не зробите
коміт або відкат. Усе в транзакції мусить проходити через `*sql.Tx` —
інструкція, видана на `db`, поки транзакція відкрита, виконується на
*іншому* з'єднанні і не є частиною цієї транзакції.

`*sql.Tx`, яку ніколи не завершують, назавжди «зливає» це з'єднання.
Достатня кількість таких — і пул вичерпується, що проявляється як
зависання запитів, а не як помилка.

## Патерн замикання (closure)

Замість того, щоб лишати коміт і відкат на розсуд кожного місця
виклику, обгорніть це один раз, щоб про життєвий цикл неможливо було
забути:

```go
func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
            return errors.Join(err, rbErr)
        }
        return err
    }
    return tx.Commit()
}
```

У шляху обробки помилок є дві деталі. `errors.Join` зберігає **обидва**
збої, коли сам відкат не вдається — оригінальна помилка пояснює, що
пішло не так, а помилка відкату пояснює, чому дані тепер можуть бути
неузгодженими. А `sql.ErrTxDone` відфільтровується, бо транзакція, яку
база даних уже перервала сама, повідомляє про це при відкаті, і це не
нова проблема:

```go
tx, _ := db.BeginTx(ctx, nil)
tx.Commit()
fmt.Println(errors.Is(tx.Rollback(), sql.ErrTxDone))   // output: true
```

Саме тому голий `defer tx.Rollback()` у прикладі на початку статті
безпечний: після успішного коміту він нічого не робить.

Використання:

```go
err := WithTx(ctx, db, func(tx *sql.Tx) error {
    if _, err := tx.ExecContext(ctx, `UPDATE acct SET bal = bal - 50 WHERE id='a'`); err != nil {
        return err
    }
    _, err := tx.ExecContext(ctx, `UPDATE acct SET bal = bal + 50 WHERE id='b'`)
    return err
})
// a,b = 50 50
```

Поверніть помилку — і нічого не записується:

```go
err := WithTx(ctx, db, func(tx *sql.Tx) error {
    tx.ExecContext(ctx, `UPDATE acct SET bal = bal - 50 WHERE id='a'`)
    return errors.New("business rule failed")
})
// err: business rule failed
// баланс не змінився
```

## Паніка мусить відкотити транзакцію і продовжити панікувати

Без цього паніка всередині замикання розкручує стек повз ваш коміт і
відкат, і транзакція лишається відкритою, доки з'єднання не «помре»:

```go
defer func() {
    if p := recover(); p != nil {
        _ = tx.Rollback()
        panic(p)          // підняти знову: не ковтайте її
    }
}()
```

Повторна паніка важлива. Проковтнута паніка перетворює помилку
програміста на непомітну відсутність дії; повторне підняття дозволяє
middleware [recover](../08-http-with-net-http/03-middleware.md)
перетворити її на `500`, поки дані лишаються узгодженими.

```go
// замикання панікує після вставки (insert)
// recovered: boom
// d вставлено? false
```

## Перенесення транзакції через контекст

Явна передача `tx` означає, що кожна функція, яка може виконуватись
усередині транзакції, приймає `*sql.Tx` — це поширюється по всьому
дереву викликів і змушує тримати дві версії функцій, які могли б бути
будь-якою з них.

Альтернатива — покласти це в контекст і дозволити кожному виклику
самому визначати, що використовувати:

```go
type txKey struct{}

func dbOrTx(ctx context.Context, db *sql.DB) execer {
    if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
        return tx
    }
    return db
}
```

`execer` — маленький інтерфейс із методами, які вже мають і `*sql.DB`,
і `*sql.Tx`, тож той самий код працює в обох випадках:

```go
type execer interface {
    ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
    QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
}
```

Тепер функція не залежить від того, є транзакція чи ні:

```go
func count(ctx context.Context, db *sql.DB) int {
    var n int
    dbOrTx(ctx, db).QueryRowContext(ctx, `SELECT count(*) FROM acct`).Scan(&n)
    return n
}
```

**Будьте чесні щодо компромісу.** Сигнатура більше не каже, чи
виконується функція в транзакції, а це саме та інформація, яку
хочеться мати, читаючи незнайомий код. Це також відкриває можливість
для реальної помилки: передати контекст, що пережив час життя
транзакції, і записи підуть не туди. Явна передача `tx` зрозуміліша в
невеликій кодовій базі; підхід із контекстом окупається, коли дерево
викликів стає глибоким.

## Вкладеність має приєднуватись, а не вкладатись

У SQL немає вкладених транзакцій. Якщо внутрішній виклик починає ще
одну транзакцію на пулі, він забирає *друге* з'єднання, яке потім
чекає на блокування, утримувані першим — самовзаємне блокування
(self-deadlock), що виглядає як зависання.

Перевірка контексту спершу змушує внутрішній виклик приєднатись до
зовнішнього:

```go
func WithTx(ctx context.Context, db *sql.DB, fn func(ctx context.Context) error) error {
    if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
        return fn(ctx)            // вже всередині: просто виконати
    }
    // ... begin, defer, commit — як вище
}
```

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    return WithTx(ctx, db, func(ctx context.Context) error {
        _, err := dbOrTx(ctx, db).ExecContext(ctx, `INSERT INTO acct VALUES ('c', 5)`)
        return err
    })
})
// err: <nil> — одна транзакція, закомічена один раз
```

Зверніть увагу, що це означає: внутрішній блок не може відкотитись сам
по собі. Помилка будь-де перериває всю зовнішню транзакцію. Якщо вам
справді потрібен частковий відкат, це `SAVEPOINT`, який ви видаєте як
SQL.

## Ізоляція та лише читання

`BeginTx` приймає опції. `nil` використовує типове значення драйвера,
яке для PostgreSQL — read-committed:

```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
    ReadOnly:  true,
})
```

```go
_, err := tx.ExecContext(ctx, `INSERT INTO acct VALUES ('e',1)`)
fmt.Println(err != nil)   // output: true
```

`ReadOnly` — дешева страховка для звітних запитів. Сильніші рівні
ізоляції можуть провалюватись під час коміту з помилкою серіалізації,
яку *застосунок* повинен повторити самостійно — тож, якщо ви піднімаєте
рівень ізоляції, додайте цикл повторних спроб.

## Тримайте транзакції короткими

З'єднання утримується на весь блок, тож ніколи не робіть усередині
нічого повільного:

- Жодних HTTP-викликів. Стороння система, що не відповідає, тепер
  тримає з'єднання з базою даних відкритим.
- Жодного очікування на канал чи блокування.
- Жодної роботи, яку можна було б зробити до `BeginTx`.

Прочитайте те, що потрібно, обчисліть, а тоді відкрийте транзакцію й
записуйте.

## Скасування

Контекст, переданий у `BeginTx`, керує всією транзакцією. Якщо його
скасовано, транзакція автоматично відкочується, і подальше
використання повертає помилку — тож відключення клієнта не може
лишити напівзастосований запис.

Фонова робота, яка мусить завершитись у будь-якому разі, потребує
контексту, який *не* успадковує скасування запиту.

> **З досвіду Python:** тут немає `with conn.begin():` і немає режиму
> autocommit-off — транзакція є явним об'єктом, і саме `defer` разом із
> функцією-обгорткою дають гарантію `with`. Повторна паніка в блоці
> recover — це те, що `__exit__` робить за вас, коли виняток
> поширюється далі.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| почати | `db.BeginTx(ctx, nil)` |
| завершити | `tx.Commit()` / `tx.Rollback()` |
| страховка | `defer tx.Rollback()` — нічого не робить після коміту |
| вже завершено | `errors.Is(err, sql.ErrTxDone)` — ігноруйте це |
| відкат теж не вдався | `errors.Join(err, rbErr)` |
| паніка | `recover`, відкат, **знову `panic(p)`** |
| уникнути передачі `tx` всюди | тримайте його в контексті, визначайте на кожен виклик |
| вкладеність | виявляйте й приєднуйтесь; у SQL немає вкладених транзакцій |
| частковий відкат | `SAVEPOINT`, виданий як SQL |
| лише читання / сильніша ізоляція | `&sql.TxOptions{...}` |
| правило на майбутнє | без мережевих викликів, без очікування, коротко |

## Джерела

- [`sql.Tx` — pkg.go.dev/database/sql#Tx](https://pkg.go.dev/database/sql#Tx)
- [`sql.DB.BeginTx` — pkg.go.dev/database/sql#DB.BeginTx](https://pkg.go.dev/database/sql#DB.BeginTx)
- [`sql.TxOptions` — pkg.go.dev/database/sql#TxOptions](https://pkg.go.dev/database/sql#TxOptions)
- [`sql.ErrTxDone` — pkg.go.dev/database/sql#pkg-variables](https://pkg.go.dev/database/sql#pkg-variables)
- [Executing transactions — go.dev/doc/database/execute-transactions](https://go.dev/doc/database/execute-transactions)
