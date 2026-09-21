// Verifies docs/13-third-party-libraries/04-yaml-and-toml.md
package yamltoml

import (
	"strings"
	"testing"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type Server struct {
	Host    string        `yaml:"host" json:"host" toml:"host"`
	Port    int           `yaml:"port" json:"port" toml:"port"`
	Timeout time.Duration `yaml:"timeout" json:"timeout" toml:"timeout"`
	Tags    []string      `yaml:"tags,omitempty" json:"tags,omitempty"`
	Secret  string        `yaml:"-" json:"-"`
}

type Doc struct {
	Server Server            `yaml:"server" toml:"server"`
	Extra  map[string]string `yaml:"extra" toml:"extra"`
}

const yamlSrc = `
server:
  host: example.com
  port: 8080
  timeout: 30s
  tags: [a, b]
extra:
  k: v
`

func TestYAMLParsesDurationNatively(t *testing.T) {
	var d Doc
	if err := yaml.Unmarshal([]byte(yamlSrc), &d); err != nil {
		t.Fatal(err)
	}
	if d.Server.Host != "example.com" || d.Server.Port != 8080 {
		t.Errorf("got %+v", d.Server)
	}
	if d.Server.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s — the article claims yaml.v3 parses this natively", d.Server.Timeout)
	}
	if len(d.Server.Tags) != 2 || d.Extra["k"] != "v" {
		t.Errorf("got %+v", d)
	}
}

func TestYAMLMarshalShape(t *testing.T) {
	out, err := yaml.Marshal(Doc{Server: Server{Host: "h", Port: 1, Timeout: time.Second}})
	if err != nil {
		t.Fatal(err)
	}
	// The article shows four-space indent and a bare `extra: {}`.
	want := "server:\n    host: h\n    port: 1\n    timeout: 1s\nextra: {}\n"
	if string(out) != want {
		t.Errorf("got:\n%q\nwant:\n%q", string(out), want)
	}
}

func TestYAMLKnownFieldsRejectsTypos(t *testing.T) {
	dec := yaml.NewDecoder(strings.NewReader("server:\n  nope: 1\n"))
	dec.KnownFields(true)
	var d Doc
	err := dec.Decode(&d)
	if err == nil {
		t.Fatal("expected an error for an unknown field")
	}
	if !strings.Contains(err.Error(), "field nope not found in type") {
		t.Errorf("error = %q, want it to name the unknown field", err)
	}
}

func TestYAMLTypeErrorNamesTheLine(t *testing.T) {
	var d Doc
	err := yaml.Unmarshal([]byte("server:\n  port: notanumber\n"), &d)
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("error = %v, want it to mention line 2", err)
	}
	if !strings.Contains(err.Error(), "cannot unmarshal !!str `notanumber` into int") {
		t.Errorf("error = %q, does not match the article", err)
	}
}

func TestTOMLBasics(t *testing.T) {
	var d Doc
	src := "[server]\nhost = \"t.example\"\nport = 9090\n"
	if err := toml.Unmarshal([]byte(src), &d); err != nil {
		t.Fatal(err)
	}
	if d.Server.Host != "t.example" || d.Server.Port != 9090 {
		t.Errorf("got %+v", d.Server)
	}
}

// The article's headline TOML warning: unlike YAML, a duration string is
// rejected. This test fails if go-toml ever gains that support, which is
// exactly when the article would need updating.
func TestTOMLRejectsDurationString(t *testing.T) {
	var d Doc
	err := toml.Unmarshal([]byte("[server]\ntimeout = \"5s\"\n"), &d)
	if err == nil {
		t.Fatal("expected an error: the article says TOML cannot parse a duration")
	}
	if !strings.Contains(err.Error(), "cannot decode TOML string into struct field") ||
		!strings.Contains(err.Error(), "time.Duration") {
		t.Errorf("error = %q, does not match the article", err)
	}
}

func TestTOMLTypeErrorNamesTheField(t *testing.T) {
	var d Doc
	err := toml.Unmarshal([]byte("[server]\nport = \"nope\"\n"), &d)
	if err == nil || !strings.Contains(err.Error(), "Server.Port") {
		t.Fatalf("error = %v, want it to name the field", err)
	}
}
