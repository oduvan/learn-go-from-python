# MCP-сервери через mcp-go

Model Context Protocol дозволяє AI-асистенту викликати ваш код. Ви
виставляєте інструменти; клієнт їх виявляє, вирішує, коли їх викликати,
і подає результати назад моделі.

> **Модуль:** `github.com/mark3labs/mcp-go`.

```go
srv := mcpserver.NewMCPServer("demo", "1.0.0",
    mcpserver.WithToolCapabilities(true),
)
srv.AddTool(searchTool(), s.handleSearch)
```

## Що таке MCP

Протокол JSON-RPC між **клієнтом** (Claude Desktop, IDE, агент) і
**сервером** (ваш процес). Сервер пропонує три типи речей:

| Примітив | Значення |
|---|---|
| **Tools** | функції, які модель може викликати, з JSON Schema |
| **Resources** | вміст для читання, який клієнт може отримати |
| **Prompts** | шаблони промптів, які може викликати користувач |

Tools — те, що реалізує більшість серверів, і те, чому присвячена ця
стаття. Модель сама обирає, коли викликати кожен, тож опис і схема —
не документація, а інтерфейс, про який модель міркує.

## Інструмент — це схема

```go
func searchTool() mcpgo.Tool {
    return mcpgo.NewTool("search",
        mcpgo.WithDescription("Search the knowledge base."),
        mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("What to search for")),
        mcpgo.WithNumber("limit", mcpgo.Description("Max results, default 10")),
    )
}
```

Це видає JSON Schema:

```json
{"properties":{"limit":{"description":"Max results, default 10","type":"number"},
 "query":{"description":"What to search for","type":"string"}},
 "required":["query"],"type":"object"}
```

**Пишіть описи ретельно.** Це єдина річ, що каже моделі, що робить
інструмент і коли ним користуватись. "Search" марне; "Search the
knowledge base by keyword; returns up to `limit` matching documents
with their ids" дає набагато краще обрання інструменту. Скажіть, що
він повертає й коли *не* варто його використовувати.

## Розділяйте визначення й поведінку

Домовленість, яку варто прийняти рано: схеми в одному файлі, обробники
в іншому.

```
internal/mcp/
  tools_search.go      → func searchTool() mcpgo.Tool
  handle_search.go     → func (s *Server) handleSearch(...)
  server.go            → registration
```

Щойно інструментів стає більше за жменю, файл, що змішує обидва —
нечитабельний, а схема — та частина, яку перечитують найчастіше, бо
саме її бачить модель.

## Обробники й нетипізована мапа аргументів

```go
func (s *Server) handleSearch(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
    q := argString(req, "query")
    if q == "" {
        return toolError("query is required"), nil
    }
    return toolJSON(map[string]any{"query": q, "hits": []string{"a", "b"}}), nil
}
```

Аргументи приходять як `map[string]any`, тож кожному обробнику потрібні
аксесори. Напишіть їх один раз:

```go
func argString(req mcpgo.CallToolRequest, key string) string {
    args, ok := req.Params.Arguments.(map[string]any)
    if !ok {
        return ""
    }
    v, _ := args[key].(string)
    return v
}
```

Твердження comma-ok повертає `""` і для "відсутнього", і для
"неправильного типу" — тож якщо вам потрібно їх розрізняти, напишіть
другий аксесор, що повертає `(string, bool)`. Числа приходять як
`float64`, точнісінько як у статті про
[кодування JSON](../06-text-time-and-data/07-encoding-json.md).

## Ключова ідіома: помилки — це дані

Ось те, що варто засвоїти назавжди.

```go
res, err := s.handleSearch(ctx, req)
// Go err: <nil>   isError: true   content: query is required
```

Провал валідації повертає `(toolError(msg), nil)` — результат,
позначений як помилка, з **nil** помилкою Go. Повідомлення йде назад
моделі, яка може прочитати його й спробувати ще раз із кращим
аргументом.

Повертайте ненульову помилку Go лише для справжнього збою транспорту
чи протоколу. Робити це для поганого аргументу — перетворити те, що
модель могла б виправити, на зламаний виклик.

Наслідок: пишіть повідомлення про помилки інструменту **для моделі**.
"query is required" — це дієво. "invalid input" — ні.

```go
func toolError(msg string) *mcpgo.CallToolResult {
    return mcpgo.NewToolResultError(msg)
}

func toolJSON(v any) *mcpgo.CallToolResult {
    b, err := json.Marshal(v)
    if err != nil {
        return toolError("encoding result: " + err.Error())
    }
    return mcpgo.NewToolResultText(string(b))
}
```

Пропускайте кожен результат через два-три хелпери на кшталт цих — і
форма виводу залишається послідовною серед сотні інструментів.

## Обгортання кожного інструмента

Реєстрація — те місце, де додають наскрізну поведінку, за формою
декоратора зі статті про
[middleware](../08-http-with-net-http/03-middleware.md):

```go
func (s *Server) tracked(name string, h mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
    return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
        ctx, span := tracer.Start(ctx, "mcp."+name)
        defer span.End()

        start := time.Now()
        res, err := h(ctx, req)
        s.hist.Record(ctx, time.Since(start).Seconds())
        return res, err
    }
}

srv.AddTool(searchTool(), s.tracked("search", s.handleSearch))
```

```
[tracked] tool=search dur<1s=true err=<nil>
```

Одна обгортка дає кожному інструменту трейсинг, метрики, логування й
аудиторський слід. Додайте туди й `recover` — паніка в обробнику
інакше кладе весь сервер.

## Транспорти

```go
// stdio: the client launches your binary
mcpserver.NewStdioServer(srv).ServeStdio(ctx)

// HTTP: a long-running service
mcpserver.NewStreamableHTTPServer(srv)
```

**stdio** — так десктопний клієнт запускає локальний сервер: він
породжує ваш процес і спілкується через stdin і stdout. А це означає,
що **нічого іншого не може писати в stdout** — випадковий
`fmt.Println` псує потік протоколу. Надсилайте логи в stderr і
налаштуйте `slog` відповідно.

**Streamable HTTP** підходить для спільного, задеплоєного сервера. Він
монтується як обробник, тож стоїть поруч із вашими наявними
маршрутами.

Прапорець `--mode`, що перемикає між ними, дозволяє одному бінарнику
робити обидва.

## Автентифікація

stdio успадковує довіру того, хто запустив процес. HTTP — ні, а
ендпоінт MCP — це API, що виконує код за запитом.

Щонайменше вимагайте bearer-токен. Для розгортання з кількома
користувачами специфікований підхід — OAuth 2.1 з ендпоінтами
discovery, побудований на елементах зі статті про
[OIDC та OAuth](17-oidc-and-oauth.md).

І проєктуйте інструменти захисно. Модель можна вмовити викликати
що завгодно — тож інструмент, що виконує наданий викликачем SQL, має
парсити й обмежувати його, той, що пише, має бути обмежений за
областю дії, а руйнівні операції взагалі не варто виставляти.

## Інший напрямок: викликати модель самостійно

MCP-сервер виставляє інструменти *для* клієнта. Іноді клієнт — ви:
ваш сервіс викликає модель і дозволяє їй використовувати ваші
інструменти. Усі примітиви вже покриті: [HTTP-клієнт](../08-http-with-net-http/02-http-client.md)
з тайм-аутами й повторними спробами, [обмежувач швидкості](03-golang-x-sync-and-time.md)
і [власний маршалінг JSON](../06-text-time-and-data/07-encoding-json.md)
для блоків вмісту з позначеним об'єднанням, які використовують ці API.
Більшості постачальників не потрібен SDK; `net/http` достатньо.

Новою є форма розмови. Цикл використання інструментів:

```go
for range maxTurns {
    resp, err := client.Complete(ctx, messages, tools)
    if err != nil {
        return err
    }

    calls := resp.ToolCalls()
    if len(calls) == 0 {
        return resp.Text(), nil        // the model is done
    }

    messages = append(messages, resp.AsMessage())
    for _, call := range calls {
        result := dispatch(ctx, call)  // run the tool
        messages = append(messages, toolResultMessage(call.ID, result))
    }
}
```

Модель сама вирішує, коли зупинитись. Чотири речі, які варто зробити
правильно:

- **Обмежте кількість ітерацій.** Без `maxTurns` модель, що продовжує
  викликати інструменти, крутиться в циклі, доки не скінчиться ваш
  бюджет.
- **Помилки повертаються як результати**, точнісінько як на боці
  сервера. Провалений виклик інструмента — це повідомлення, на яке
  модель може відреагувати, а не помилка Go, що перериває цикл.
- **Стежте за вікном контексту.** Кожен результат додається, тож
  інструмент, що повертає великий вивід, заповнює вікно за три
  ітерації. Обрізайте, пагінуйте або повертайте id.
- **Дедлайн контексту покриває весь цикл**, а не один виклик.
  Закладайте бюджет на кілька раундтрипів.

Промпти варто тримати поза кодом — у файлах чи таблиці бази даних —
тож вони можуть змінюватись без деплою, а рендеринг їх через
[`text/template`](../08-http-with-net-http/04-templates.md) кращий за
конкатенацію рядків.

## Проєктування інструментів, якими модель може користуватись

- **Кілька широких інструментів кращі за багато вузьких.** Модель, що
  обирає з двадцяти інструментів, справляється краще, ніж із двохсот.
- **Повертайте структурований JSON**, не прозу. Модель це парсить.
- **Тримайте відповіді малими.** Усе повернене потрапляє у вікно
  контексту; пагінуйте й повертайте id, за якими модель може перейти
  далі.
- **Робіть їх ідемпотентними**, де можливо. Повторений виклик має бути
  безпечним.
- **Кажіть, чого інструмент не робить.** Запобігання неправильному
  виклику так само цінне, як і вмикання правильного.

> **З досвіду Python:** протокол ідентичний, а офіційний Python SDK
> магічніший — декоратори виводять схему з type hints. Тут ви
> оголошуєте схему явно, що означає більше набору тексту, але не
> лишає сумнівів у тому, що бачить модель.

## Швидка довідка

| Задача | Форма |
|---|---|
| сервер | `mcpserver.NewMCPServer(name, version, opts...)` |
| схема інструмента | `mcpgo.NewTool(name, WithDescription, WithString(...))` |
| обов'язковий аргумент | `mcpgo.Required()` |
| зареєструвати | `srv.AddTool(def, handler)` |
| обробник | `func(ctx, mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error)` |
| прочитати аргументи | твердження `req.Params.Arguments.(map[string]any)` |
| **поганий аргумент** | `return toolError(msg), nil` — nil помилка Go |
| справжній збій | ненульова помилка Go |
| результати | один хелпер `toolJSON`/`toolText`, використовуваний всюди |
| наскрізні турботи | обгортайте обробник при реєстрації |
| локальний клієнт | `ServeStdio` — **нічого в stdout, крім протоколу** |
| задеплоєний | `NewStreamableHTTPServer`, за автентифікацією |

## Джерела

- [Model Context Protocol — modelcontextprotocol.io](https://modelcontextprotocol.io/)
- [MCP specification — spec.modelcontextprotocol.io](https://spec.modelcontextprotocol.io/)
- [`mcp-go` — pkg.go.dev/github.com/mark3labs/mcp-go](https://pkg.go.dev/github.com/mark3labs/mcp-go)
- [mcp-go repository — github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [MCP transports — modelcontextprotocol.io/docs/concepts/transports](https://modelcontextprotocol.io/docs/concepts/transports)
