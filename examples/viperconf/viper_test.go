// Verifies docs/13-third-party-libraries/05-viper.md
package viperconf

import (
	"reflect"
	"testing"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment string        `mapstructure:"ENVIRONMENT"`
	Port        int           `mapstructure:"PORT"`
	Timeout     time.Duration `mapstructure:"TIMEOUT"`
	DBHost      string        `mapstructure:"DB_HOST"`
}

// bindEnvs is the article's recommendation: derive the key list from the
// struct tags so adding a field needs no other change.
func bindEnvs(v *viper.Viper, cfg any) {
	t := reflect.TypeOf(cfg)
	for i := range t.NumField() {
		if tag := t.Field(i).Tag.Get("mapstructure"); tag != "" && tag != "-" {
			_ = v.BindEnv(tag)
		}
	}
}

// The article's headline gotcha: AutomaticEnv makes Get work but leaves
// Unmarshal empty, with no error from either.
func TestAutomaticEnvDoesNotFeedUnmarshal(t *testing.T) {
	t.Setenv("ONLY_ENV", "zzz")

	v := viper.New()
	v.AutomaticEnv()

	if got := v.GetString("ONLY_ENV"); got != "zzz" {
		t.Errorf("Get = %q, want \"zzz\"", got)
	}
	if !v.IsSet("ONLY_ENV") {
		t.Error("IsSet = false, want true")
	}

	var c struct {
		OnlyEnv string `mapstructure:"ONLY_ENV"`
	}
	if err := v.Unmarshal(&c); err != nil {
		t.Fatalf("Unmarshal returned an error: %v", err)
	}
	if c.OnlyEnv != "" {
		t.Errorf("Unmarshal filled the field with %q — the article says it stays empty; "+
			"if viper fixed this, the article needs updating", c.OnlyEnv)
	}
}

func TestBindEnvMakesUnmarshalWork(t *testing.T) {
	t.Setenv("ENVIRONMENT", "prod")
	t.Setenv("PORT", "7070")
	t.Setenv("TIMEOUT", "45s")
	t.Setenv("DB_HOST", "db.host")

	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("ENVIRONMENT", "local")
	v.SetDefault("PORT", 8080)
	v.SetDefault("TIMEOUT", "5s")
	bindEnvs(v, Config{})

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		t.Fatal(err)
	}
	want := Config{Environment: "prod", Port: 7070, Timeout: 45 * time.Second, DBHost: "db.host"}
	if c != want {
		t.Errorf("got %+v, want %+v", c, want)
	}
}

func TestDefaultsApplyWhenUnset(t *testing.T) {
	v := viper.New()
	v.SetDefault("ENVIRONMENT", "local")
	v.SetDefault("PORT", 8080)
	v.SetDefault("TIMEOUT", "5s")

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		t.Fatal(err)
	}
	if c.Environment != "local" || c.Port != 8080 || c.Timeout != 5*time.Second {
		t.Errorf("got %+v", c)
	}
}

// SetDefault registers the key, so it alone is enough for Unmarshal.
func TestSetDefaultAlsoRegistersTheKey(t *testing.T) {
	t.Setenv("PORT", "1234")

	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("PORT", 8080)

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		t.Fatal(err)
	}
	if c.Port != 1234 {
		t.Errorf("Port = %d, want 1234 (SetDefault registers the key so the env wins)", c.Port)
	}
}
