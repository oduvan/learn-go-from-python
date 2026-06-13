# Встановлення Go

Ці матеріали розраховані на **Go 1.26.4** (поточний стабільний). Перевіряйте [go.dev/dl](https://go.dev/dl/) для отримання найактуальнішої версії у будь-який момент.

## macOS

Три розумних варіанти. Оберіть один — не змішуйте їх.

### Варіант 1 — Homebrew (рекомендовано для щоденної роботи на macOS)

Якщо ви вже використовуєте Homebrew:

```bash
brew install go
```

- Легке оновлення: `brew upgrade go`.
- Чисте видалення: `brew uninstall go`.
- Шлях встановлення — префікс Homebrew (`/opt/homebrew/Cellar/go/...` на Apple Silicon, `/usr/local/Cellar/go/...` на Intel), з символічним посиланням у `$(brew --prefix)/bin/go`.
- Відстає від офіційних релізів на кілька днів — зазвичай не критично.

### Варіант 2 — Офіційний інсталятор `.pkg`

1. Завантажте з [go.dev/dl](https://go.dev/dl/) — оберіть `darwin-arm64.pkg` (Apple Silicon) або `darwin-amd64.pkg` (Intel).
2. Двічі клацніть і дотримуйтесь підказок (потрібен пароль адміністратора).
3. Інсталятор розміщує toolchain у `/usr/local/go` і додає `/usr/local/go/bin` до вашого `PATH` через `/etc/paths.d/go`.
4. Перезапустіть термінал.

### Варіант 3 — Ручне встановлення з архіву

Для повного контролю над місцем встановлення:

```bash
# download go1.26.4.darwin-arm64.tar.gz from https://go.dev/dl/

sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ~/Downloads/go1.26.4.darwin-arm64.tar.gz
```

Додайте `/usr/local/go/bin` до вашого `PATH` у `~/.zshrc` (або `~/.bash_profile`):

```bash
export PATH=$PATH:/usr/local/go/bin
```

Перезавантажте оболонку і перевірте (дивіться "Перевірка встановлення" нижче).

## Linux

Офіційний спосіб — архів tar. Пакети дистрибутивів (`apt install golang-go`, `dnf install golang` тощо) часто відстають від офіційного релізу на кілька місяців — підходить для нескладного використання, але не рекомендується, якщо вам потрібні актуальні функції.

1. Завантажте відповідний `.tar.gz` з [go.dev/dl](https://go.dev/dl/) для вашої архітектури.
2. Видаліть попереднє встановлення і розпакуйте:

```bash
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
```

3. Додайте `/usr/local/go/bin` до вашого `PATH`, дописавши цей рядок у `~/.profile` (або `~/.bashrc`):

```bash
export PATH=$PATH:/usr/local/go/bin
```

4. Застосуйте і перевірте:

```bash
source ~/.profile
go version
```

## Windows

1. Завантажте `.msi` з [go.dev/dl](https://go.dev/dl/).
2. Двічі клацніть; інсталятор розміщує Go у `Program Files` і додає до `PATH`.
3. Закрийте і знову відкрийте всі відкриті командні рядки, щоб вони отримали оновлений `PATH`.
4. Перевірте у новому командному рядку:

```cmd
go version
```

## Перевірка встановлення

Після встановлення виконайте три швидкі перевірки:

```bash
go version
# example: go version go1.26.4 darwin/arm64

go env GOROOT
# wherever the toolchain landed (e.g. /usr/local/go)

go env GOPATH
# defaults to $HOME/go — holds module cache and installed tools
```

Потім запустіть hello-world, щоб підтвердити, що весь конвеєр працює наскрізь:

```bash
mkdir -p /tmp/hello && cd /tmp/hello
go mod init example.com/hello
cat > hello.go <<'EOF'
package main

import "fmt"

func main() {
    fmt.Println("hello, go")
}
EOF
go run .
# hello, go
```

Якщо `go version` працює, а `go run .` — ні, toolchain встановлено правильно, а збій пов'язаний з вашим кодом або з `PATH` для кешу модулів.

## Встановлення додаткових версій паралельно

Якщо Go вже встановлено, ви можете отримати інші версії через офіційний механізм `dl/` — менеджер версій не потрібен:

```bash
go install golang.org/dl/go1.22.3@latest
go1.22.3 download
go1.22.3 version
go1.22.3 build ./...
```

Кожна версія стає своєю власною командою (`go1.22.3`, `go1.23.0` тощо). Вони встановлюються у `~/sdk/<version>/` і не заважають вашому основному бінарному файлу `go`.

Знайдіть `GOROOT` кожного:

```bash
go1.22.3 env GOROOT
```

Щоб видалити паралельну версію, видаліть її директорію `GOROOT` і бінарний файл `goX.Y.Z` з `$GOBIN`.

Дивіться [06-multiple-go-versions.md](06-multiple-go-versions.md) для ширшого контексту — `GOTOOLCHAIN=auto` автоматично обробляє вибір версії для кожного проєкту, щойно у вас встановлено будь-який Go.

## Видалення Go

### macOS (встановлення через `.pkg`)

```bash
sudo rm -rf /usr/local/go
sudo rm /etc/paths.d/go
```

За бажанням також видаліть `~/go/`, щоб очистити кеш модулів і встановлені інструменти.

### macOS (встановлення через Homebrew)

```bash
brew uninstall go
```

### Linux

```bash
sudo rm -rf /usr/local/go
# then remove the PATH line from ~/.profile or ~/.bashrc
```

### Windows

Панель керування → **Установка та видалення програм** → **Go Programming Language** → **Видалити**. Змінні середовища очищаються автоматично.

## З досвіду Python

Кілька речей, які варто переосмислити після років з Python:

- **Один toolchain постачає все.** Компілятор, компонувальник, засіб запуску тестів, форматер, менеджер залежностей — усе це бінарний файл `go`. Немає аналога необхідності окремо встановлювати `pip`, `venv`, `pytest`, `black` і `flake8`.
- **Жодних віртуальних середовищ для кожного проєкту.** Ізоляція модулів забезпечується файлами `go.mod` у корені кожного проєкту; залежності кешуються глобально у `$GOPATH/pkg/mod/`, але розрізняються за модулем. Нічого «активувати» не потрібно.
- **`$GOBIN` ≈ місце встановлення `pipx`.** Після `go install <tool>@latest` бінарний файл опиняється у `~/go/bin/` — єдиному глобальному місці для CLI-інструментів, без віртуальних середовищ.
- **Жодної пастки «системного Python».** macOS не постачає Go, тож немає ризику випадково оновити системний Go. Що б ви не встановили — це єдиний Go.

## Джерела

- [Завантаження та встановлення — go.dev/doc/install](https://go.dev/doc/install)
- [Усі релізи Go — go.dev/dl](https://go.dev/dl/)
- [Керування встановленнями Go — go.dev/doc/manage-install](https://go.dev/doc/manage-install)
- [Посібник: Початок роботи з Go — go.dev/doc/tutorial/getting-started](https://go.dev/doc/tutorial/getting-started)
