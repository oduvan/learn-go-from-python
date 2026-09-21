// Verifies docs/13-third-party-libraries/17-oidc-and-oauth.md
package authdemo

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

var signingKey = []byte("server-secret-key")

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

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "snx_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(t string) string {
	s := sha256.Sum256([]byte(t))
	return base64.RawURLEncoding.EncodeToString(s[:])
}

func pkce() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	s := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(s[:])
	return verifier, challenge, nil
}

func TestSignedCookieRoundTripAndTamper(t *testing.T) {
	tok := sign("user:ada")
	got, ok := verify(tok)
	if !ok || got != "user:ada" {
		t.Fatalf("verify = %q %v", got, ok)
	}
	if _, ok := verify(strings.Replace(tok, "ada", "bob", 1)); ok {
		t.Error("a tampered cookie was accepted")
	}
	if _, ok := verify("no-dot-here"); ok {
		t.Error("a malformed cookie was accepted")
	}
}

func TestAPITokenShapeAndHashing(t *testing.T) {
	tok, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tok, "snx_") {
		t.Errorf("token = %q, want an snx_ prefix for secret scanners", tok)
	}
	if len(tok) != 47 {
		t.Errorf("len = %d, want 47 (4 prefix + 43 base64url of 32 bytes)", len(tok))
	}
	h := hashToken(tok)
	if h == tok {
		t.Error("the stored hash equals the token")
	}
	if hashToken(tok) != h {
		t.Error("hashing is not deterministic")
	}

	other, _ := newToken()
	if other == tok {
		t.Error("two generated tokens collided")
	}
}

func TestPKCEChallengeVerifies(t *testing.T) {
	v, ch, err := pkce()
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 43 || len(ch) != 43 {
		t.Errorf("verifier=%d challenge=%d, want 43 each", len(v), len(ch))
	}
	sum := sha256.Sum256([]byte(v))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != ch {
		t.Error("challenge is not base64url(sha256(verifier))")
	}
	// RawURLEncoding is required: +, / and = all break in a URL.
	if strings.ContainsAny(ch, "+/=") {
		t.Errorf("challenge %q contains characters that need URL escaping", ch)
	}
}

func TestConstantTimeCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"abc", "abc", 1},
		{"abc", "abd", 0},
		{"abc", "ab", 0}, // different lengths are not equal, and not protected
	}
	for _, c := range cases {
		if got := subtle.ConstantTimeCompare([]byte(c.a), []byte(c.b)); got != c.want {
			t.Errorf("ConstantTimeCompare(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestAuthCodeURLCarriesStateAndPKCE(t *testing.T) {
	_, ch, err := pkce()
	if err != nil {
		t.Fatal(err)
	}
	conf := &oauth2.Config{
		ClientID:     "client-123",
		ClientSecret: "secret",
		RedirectURL:  "https://app.example/callback",
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://idp.example/authorize",
			TokenURL: "https://idp.example/token",
		},
	}
	u := conf.AuthCodeURL("state-xyz",
		oauth2.SetAuthURLParam("code_challenge", ch),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))

	if !strings.HasPrefix(u, "https://idp.example/authorize?") {
		t.Fatalf("url = %s", u)
	}
	for _, want := range []string{
		"client_id=client-123", "state=state-xyz",
		"code_challenge_method=S256", "response_type=code", "scope=",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("url missing %q\ngot %s", want, u)
		}
	}
}
