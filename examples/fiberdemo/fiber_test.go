// Verifies docs/13-third-party-libraries/10-fiber.md
package fiberdemo

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/valyala/fasthttp"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Store interface{ ByID(string) (User, error) }

type stubStore struct{}

func (stubStore) ByID(id string) (User, error) {
	if id == "404" {
		return User{}, fmt.Errorf("not found")
	}
	return User{Name: "Ada:" + id, Age: 36}, nil
}

type Handler struct{ users Store }

func (h Handler) get(c fiber.Ctx) error {
	u, err := h.users.ByID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(u)
}

func app() *fiber.App {
	a := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})
	a.Use(requestid.New())
	a.Use(recover.New())

	h := Handler{users: stubStore{}}
	a.Get("/users/:id", h.get)
	a.Get("/search", func(c fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("q=%q page=%q", c.Query("q"), c.Query("page")))
	})
	a.Post("/users", func(c fiber.Ctx) error {
		var u User
		if err := c.Bind().Body(&u); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
		}
		return c.Status(fiber.StatusCreated).JSON(u)
	})
	a.Get("/boom", func(c fiber.Ctx) error { panic("kaboom") })
	a.Get("/fail", func(c fiber.Ctx) error { return fiber.NewError(fiber.StatusTeapot, "teapot") })
	a.Get("/stream", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.RequestCtx().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			for i := 1; i <= 3; i++ {
				fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
				if err := w.Flush(); err != nil {
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}))
		return nil
	})
	return a
}

func do(t *testing.T, a *fiber.App, method, target, body string) (int, string, http.Header) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, "http://x"+target, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(b)), resp.Header
}

func TestRoutingAndJSON(t *testing.T) {
	a := app()
	code, body, hdr := do(t, a, "GET", "/users/7", "")
	if code != 200 || body != `{"name":"Ada:7","age":36}` {
		t.Errorf("%d %s", code, body)
	}
	if hdr.Get("X-Request-Id") == "" {
		t.Error("requestid middleware did not set X-Request-Id")
	}
}

func TestQueryParams(t *testing.T) {
	code, body, _ := do(t, app(), "GET", "/search?q=go&page=2", "")
	if code != 200 || body != `q="go" page="2"` {
		t.Errorf("%d %s", code, body)
	}
}

func TestHandlerErrorPath(t *testing.T) {
	code, body, _ := do(t, app(), "GET", "/users/404", "")
	if code != 404 || body != `{"error":"not found"}` {
		t.Errorf("%d %s", code, body)
	}
}

func TestBindBody(t *testing.T) {
	a := app()
	code, body, _ := do(t, a, "POST", "/users", `{"name":"Bo","age":7}`)
	if code != 201 || body != `{"name":"Bo","age":7}` {
		t.Errorf("%d %s", code, body)
	}
	code, body, _ = do(t, a, "POST", "/users", `{`)
	if code != 400 || body != `{"error":"bad body"}` {
		t.Errorf("malformed body: %d %s", code, body)
	}
}

func TestRecoverTurnsPanicInto500(t *testing.T) {
	code, body, _ := do(t, app(), "GET", "/boom", "")
	if code != 500 || !strings.Contains(body, "kaboom") {
		t.Errorf("%d %s", code, body)
	}
}

func TestErrorHandlerReceivesFiberError(t *testing.T) {
	code, body, _ := do(t, app(), "GET", "/fail", "")
	if code != 418 || body != `{"error":"teapot"}` {
		t.Errorf("%d %s", code, body)
	}
}

func TestNotFoundAndMethodMismatch(t *testing.T) {
	a := app()
	if code, body, _ := do(t, a, "GET", "/nope", ""); code != 404 || body != `{"error":"Not Found"}` {
		t.Errorf("404 case: %d %s", code, body)
	}
	if code, body, _ := do(t, a, "DELETE", "/users/7", ""); code != 405 || body != `{"error":"Method Not Allowed"}` {
		t.Errorf("405 case: %d %s", code, body)
	}
}

// SSE under Fiber goes through fasthttp, not http.Flusher.
func TestSSEStreaming(t *testing.T) {
	code, body, hdr := do(t, app(), "GET", "/stream", "")
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if ct := hdr.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q", ct)
	}
	for _, want := range []string{"event: tick", "data: 1", "data: 2", "data: 3"} {
		if !strings.Contains(body, want) {
			t.Errorf("stream missing %q; got:\n%s", want, body)
		}
	}
}
