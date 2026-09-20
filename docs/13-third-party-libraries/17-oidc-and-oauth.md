# OIDC and OAuth

Delegating login to an identity provider, and the session and API-token
handling around it. The cryptography is standard-library —
[hashing and random values](../07-operating-system/07-hashing-and-random-values.md)
covers the primitives; this is how they fit together.

> **Modules:** `golang.org/x/oauth2` and
> `github.com/coreos/go-oidc/v3`.

```go
conf := &oauth2.Config{
    ClientID:    clientID,
    RedirectURL: "https://app.example/callback",
    Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
    Endpoint:    provider.Endpoint(),
}
```

## Never build this yourself

Two rules before anything else. **Do not implement an identity
provider**, and **do not store passwords** unless that is your
product. Delegating to an OIDC provider means you never hold a
credential that can be stolen from you.

What you do implement is the client side — and the parts below are
where client-side mistakes happen.

## The authorization code flow

1. User hits a protected page; you redirect to the provider with a
   `state` and a PKCE challenge.
2. They authenticate there.
3. The provider redirects back with a `code`.
4. You exchange the code for tokens, **server to server**.
5. You verify the ID token and create your own session.

```go
provider, err := oidc.NewProvider(ctx, "https://idp.example")
```

`NewProvider` fetches the discovery document, so endpoints and signing
keys come from the provider rather than your config. Do it once at
startup — it makes a network call.

## `state` and PKCE are not optional

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

**`state`** is a random value you store (in a short-lived cookie) and
compare on the callback. Without it, an attacker can feed a victim's
browser their own authorization code — a login CSRF that logs the
victim into the attacker's account.

**PKCE** proves the client redeeming the code is the one that started
the flow. Generate a random verifier, send its SHA-256 hash as the
challenge, and send the verifier at exchange time:

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

`RawURLEncoding` matters: the spec requires base64url with no padding,
and `StdEncoding` produces `+`, `/` and `=` — all of which break in a
URL. That is the encoding point from the hashing article, with a
concrete consequence.

PKCE was originally for mobile apps. It is now recommended for all
clients, including confidential ones.

## Exchange and verify

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

**Always `Verify`.** It checks the signature against the provider's
published keys, the issuer, the audience and the expiry. A JWT is
readable by anyone — decoding one without verifying it means trusting
whatever the browser sent.

Check `email_verified` before trusting an email as an identity. Some
providers will happily assert an unverified address, and account
takeover follows.

## Your own session

The provider's tokens are for the provider. Issue your own session,
and keep it opaque:

```go
b := make([]byte, 32)
rand.Read(b)
sessionID := base64.RawURLEncoding.EncodeToString(b)
```

Store the session server-side, keyed by that id. Then logout,
revocation and role changes are one delete.

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

`HttpOnly` keeps JavaScript out, `Secure` keeps it off plain HTTP,
`SameSite=Lax` blunts CSRF.

### Signed cookies, if you must be stateless

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

The constant-time comparison is required, not decorative — a plain
`==` leaks how much of the signature was right.

Signing is not encryption: the payload is readable. And the cost of
statelessness is that you cannot revoke, so keep the lifetime short.

## API tokens

For machine callers, issue a random token and **store only its hash**:

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

Show the token once, keep the hash. A leaked database then contains
nothing usable.

Plain SHA-256 is right here and wrong for passwords: a 256-bit random
token is not brute-forceable, so the slow hashing that protects a weak
password buys nothing and costs latency on every request.

The `snx_` prefix helps secret scanners spot a leaked token in a
repository — worth doing.

Compare on lookup with `subtle.ConstantTimeCompare`, and give tokens an
expiry and a revocation flag.

## Refresh tokens

`oauth2.Config.TokenSource` refreshes automatically when a token
expires:

```go
client := conf.Client(ctx, tok)   // refreshes as needed
```

Refresh tokens are long-lived credentials — store them encrypted, and
support rotation, where each refresh issues a new one and invalidates
the old.

## If you are the authorization server

Everything above is the *client* side. Occasionally you are the other
end — most commonly when something must authenticate to *your* API on
a user's behalf, which is what an MCP server exposed over HTTP needs.

The pieces then invert:

- **`/.well-known/oauth-authorization-server`** and
  `/.well-known/oauth-protected-resource` — discovery documents, so a
  client can find your endpoints without configuration.
- **`GET /authorize`** — authenticate the user, then redirect back with
  a short-lived code. Store the code against the client id, the
  redirect URI and the PKCE challenge.
- **`POST /token`** — exchange the code. Verify the redirect URI
  matches **exactly** what was registered, and verify the PKCE
  verifier: recompute `base64url(sha256(verifier))` and compare in
  constant time against the stored challenge.
- **`POST /register`** — dynamic client registration, if clients are
  not pre-configured.

Three rules carry most of the security. Authorization codes are
**single use** and short-lived, so redeeming one twice must invalidate
the session rather than issue a second token. Redirect URIs are matched
against an **allowlist**, never by prefix — a prefix match lets an
attacker append a path and receive the code. And refresh tokens should
**rotate**: each refresh issues a new one and invalidates its
predecessor, so a stolen token is detectable when the legitimate client
presents the old one.

Error responses here follow RFC 6749 rather than your usual error
shape — a JSON body with `error` and `error_description`, and specific
codes like `invalid_grant`. Clients parse it, so this is one place a
house error helper is the wrong thing to use.

## The checklist

- `state` on every flow, compared on callback
- PKCE with `S256`
- `Verify` the ID token — never decode and trust
- check `email_verified`
- cookies `HttpOnly`, `Secure`, `SameSite`
- opaque session ids, stored server-side
- API tokens hashed at rest, compared in constant time
- exchange codes server-side only; the client secret never reaches a
  browser

> **From Python:** `oauth2.Config` is Authlib's client, and `go-oidc`
> does the discovery and JWT verification Authlib's OIDC mixin does.
> Nothing here is automatic — `state` and PKCE are parameters you pass,
> which makes it easy to omit them and important not to.

## Quick reference

| Task | Form |
|---|---|
| discovery | `oidc.NewProvider(ctx, issuer)` once at startup |
| redirect URL | `conf.AuthCodeURL(state, pkceParams...)` |
| CSRF | a random `state`, stored and compared |
| PKCE | SHA-256 of a random verifier, `RawURLEncoding`, `S256` |
| exchange | `conf.Exchange(ctx, code, code_verifier)` — server-side |
| **verify** | `provider.Verifier(...).Verify(ctx, rawIDToken)` |
| claims | `idToken.Claims(&struct)`, check `email_verified` |
| session | 32 random bytes, server-side, `HttpOnly`+`Secure`+`SameSite` |
| stateless session | HMAC-SHA256 + `subtle.ConstantTimeCompare` |
| API tokens | random, prefixed, **hashed at rest** |
| refresh | `conf.Client(ctx, tok)` |

## Sources

- [`golang.org/x/oauth2` — pkg.go.dev/golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
- [`go-oidc` — pkg.go.dev/github.com/coreos/go-oidc/v3/oidc](https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc)
- [OAuth 2.0 for browser-based apps — datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps)
- [PKCE, RFC 7636 — datatracker.ietf.org/doc/html/rfc7636](https://datatracker.ietf.org/doc/html/rfc7636)
- [OpenID Connect Core — openid.net/specs/openid-connect-core-1_0.html](https://openid.net/specs/openid-connect-core-1_0.html)
- [`http.Cookie` — pkg.go.dev/net/http#Cookie](https://pkg.go.dev/net/http#Cookie)
