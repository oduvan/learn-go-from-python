// Verifies docs/13-third-party-libraries/12-testify.md
package testifydemo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ErrEmpty = errors.New("empty")

type User struct {
	Name string
	Tags []string
}

func Load(id string) (User, error) {
	if id == "" {
		return User{}, ErrEmpty
	}
	return User{Name: "Ada", Tags: []string{"a", "b"}}, nil
}

func TestRequireThenAssert(t *testing.T) {
	u, err := Load("1")
	require.NoError(t, err)
	require.Equal(t, "Ada", u.Name)

	assert.Len(t, u.Tags, 2)
	assert.Contains(t, u.Tags, "a")
	assert.ElementsMatch(t, []string{"b", "a"}, u.Tags)
}

func TestErrorHelpersUnwrap(t *testing.T) {
	_, err := Load("")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmpty)

	wrapped := errors.Join(errors.New("context"), ErrEmpty)
	assert.ErrorIs(t, wrapped, ErrEmpty)
}

func TestJSONEqIgnoresKeyOrder(t *testing.T) {
	assert.JSONEq(t, `{"a":1,"b":2}`, `{"b":2,"a":1}`)
}

func TestInDelta(t *testing.T) {
	assert.InDelta(t, 3.14159, 3.1416, 0.001)
}

// The article warns that assert.Equal distinguishes a nil slice from an
// empty one, and that assert.Empty is what you want instead. Both halves
// are asserted here without failing the suite.
func TestEqualDistinguishesNilFromEmpty(t *testing.T) {
	var nilSlice []string
	emptySlice := []string{}

	if assert.ObjectsAreEqual(nilSlice, emptySlice) {
		t.Error("assert.Equal now treats nil and empty as equal; the article says it does not")
	}
	assert.Empty(t, nilSlice)
	assert.Empty(t, emptySlice)
}
