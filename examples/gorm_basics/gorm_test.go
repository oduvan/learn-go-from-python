// Verifies docs/13-third-party-libraries/07-gorm-basics.md
package gormbasics

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/oduvan/learn-go-from-python/examples/infra"
)

type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type Author struct {
	Base
	Name  string `gorm:"not null;unique"`
	Books []Book `gorm:"foreignKey:AuthorID"`
}

type Book struct {
	Base
	Title    string    `gorm:"not null"`
	AuthorID uuid.UUID `gorm:"type:uuid;index"`
	Draft    bool      `gorm:"not null;default:true"`
	Notes    *string
	Ignored  string `gorm:"-"`
}

func (b *Book) BeforeCreate(tx *gorm.DB) error {
	if b.Title == "" {
		return errors.New("title is required")
	}
	return nil
}

type Tag struct {
	ID    int64          `gorm:"primaryKey"`
	Name  string         `gorm:"uniqueIndex"`
	Langs pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
}

func open(t *testing.T) *gorm.DB {
	t.Helper()
	infra.RequirePostgres(t)

	db, err := gorm.Open(postgres.New(postgres.Config{DSN: infra.PostgresDSN()}), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`DROP TABLE IF EXISTS books, authors, tags CASCADE`)
	if err := db.AutoMigrate(&Author{}, &Book{}, &Tag{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateFillsKeyAndTimestamps(t *testing.T) {
	db := open(t)
	a := Author{Name: "Ada"}
	res := db.Create(&a)
	if res.Error != nil || res.RowsAffected != 1 {
		t.Fatalf("err=%v rows=%d", res.Error, res.RowsAffected)
	}
	if a.ID == uuid.Nil || a.CreatedAt.IsZero() {
		t.Errorf("Create did not populate the key/timestamps: %+v", a)
	}
}

// The article's headline trap, and the three fixes that do NOT work.
func TestDefaultTagSwallowsAZeroValue(t *testing.T) {
	db := open(t)
	a := Author{Name: "Ada"}
	db.Create(&a)

	check := func(label string, mk func() *Book) bool {
		t.Helper()
		b := mk()
		var back Book
		db.First(&back, "id = ?", b.ID)
		t.Logf("  %-28s Draft=%v", label, back.Draft)
		return back.Draft
	}

	plain := check("plain struct", func() *Book {
		b := &Book{Title: "plain", AuthorID: a.ID, Draft: false}
		db.Create(b)
		return b
	})
	if !plain {
		t.Error("plain struct stored false — the article says the default wins")
	}

	sel := check(`Select("Title","Draft")`, func() *Book {
		b := &Book{Title: "sel", AuthorID: a.ID, Draft: false}
		db.Select("Title", "AuthorID", "Draft").Create(b)
		return b
	})
	if !sel {
		t.Error("Select fixed it — the article says it does not; update the article")
	}

	star := check(`Select("*")`, func() *Book {
		b := &Book{Title: "star", AuthorID: a.ID, Draft: false}
		db.Select("*").Create(b)
		return b
	})
	if !star {
		t.Error(`Select("*") fixed it — the article says it does not`)
	}
}

func TestMapCreateStoresTheZeroValue(t *testing.T) {
	db := open(t)
	a := Author{Name: "Ada"}
	db.Create(&a)

	id := uuid.New()
	if err := db.Model(&Book{}).Create(map[string]any{
		"id": id, "title": "viamap", "author_id": a.ID, "draft": false,
	}).Error; err != nil {
		t.Fatal(err)
	}
	var back Book
	db.First(&back, "id = ?", id)
	if back.Draft {
		t.Error("Draft = true; the article says a map create stores false")
	}
}

func TestUpdatesWithStructSkipsZeroValues(t *testing.T) {
	db := open(t)
	a := Author{Name: "Ada"}
	db.Create(&a)
	b := Book{Title: "orig", AuthorID: a.ID}
	db.Create(&b)

	db.Model(&b).Updates(Book{Title: "renamed", Draft: false})
	var after Book
	db.First(&after, "id = ?", b.ID)
	if after.Title != "renamed" {
		t.Errorf("Title = %q, want renamed", after.Title)
	}
	if !after.Draft {
		t.Error("Draft became false; the article says a struct update skips it")
	}

	db.Model(&b).Updates(map[string]any{"draft": false})
	db.First(&after, "id = ?", b.ID)
	if after.Draft {
		t.Error("a map update should have set Draft to false")
	}
}

func TestFirstReturnsErrRecordNotFound(t *testing.T) {
	db := open(t)
	var b Book
	err := db.First(&b, "title = ?", "nope").Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("err = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestFindDoesNotErrorOnEmpty(t *testing.T) {
	db := open(t)
	var books []Book
	r := db.Where("title = ?", "nope").Find(&books)
	if r.Error != nil {
		t.Errorf("Find returned %v, want nil", r.Error)
	}
	if r.RowsAffected != 0 || len(books) != 0 {
		t.Errorf("rows=%d len=%d", r.RowsAffected, len(books))
	}
}

func TestHookRejects(t *testing.T) {
	db := open(t)
	a := Author{Name: "Ada"}
	db.Create(&a)
	err := db.Create(&Book{AuthorID: a.ID}).Error
	if err == nil || err.Error() != "title is required" {
		t.Fatalf("err = %v, want \"title is required\"", err)
	}
}

func TestUniqueViolationSurfaces(t *testing.T) {
	db := open(t)
	db.Create(&Author{Name: "Ada"})
	err := db.Create(&Author{Name: "Ada"}).Error
	if err == nil || !strings.Contains(err.Error(), "SQLSTATE 23505") {
		t.Fatalf("err = %v, want a 23505 unique violation", err)
	}
}

func TestPreloadIsNotLazy(t *testing.T) {
	db := open(t)
	a := Author{Name: "Ada"}
	db.Create(&a)
	db.Create(&Book{Title: "one", AuthorID: a.ID})

	var without Author
	db.First(&without, "id = ?", a.ID)
	if len(without.Books) != 0 {
		t.Error("Books populated without Preload; GORM does not lazy-load")
	}

	var with Author
	db.Preload("Books").First(&with, "id = ?", a.ID)
	if len(with.Books) != 1 {
		t.Errorf("Preload gave %d books, want 1", len(with.Books))
	}
}

// Added after the Nexus audit: GORM on pgx still needs lib/pq for arrays.
func TestPqStringArrayRoundTrip(t *testing.T) {
	db := open(t)
	tag := Tag{Name: "go", Langs: pq.StringArray{"go", "python"}}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatal(err)
	}
	var back Tag
	db.First(&back, "name = ?", "go")
	if len(back.Langs) != 2 || back.Langs[0] != "go" || back.Langs[1] != "python" {
		t.Errorf("Langs = %v, want [go python]", back.Langs)
	}
}
