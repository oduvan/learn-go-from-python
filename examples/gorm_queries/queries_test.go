// Verifies docs/13-third-party-libraries/08-gorm-queries-and-transactions.md
package gormqueries

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/oduvan/learn-go-from-python/examples/infra"
)

type Author struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name  string    `gorm:"not null;unique"`
	Books []Book    `gorm:"foreignKey:AuthorID"`
}

type Book struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title    string    `gorm:"not null"`
	Pages    int
	AuthorID uuid.UUID `gorm:"type:uuid;index"`
}

type Tag struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex"`
	Hits int    `gorm:"not null;default:0"`
}

type titleCount struct {
	Name  string
	Total int
}

type txKey struct{}

func WithTx(ctx context.Context, db *gorm.DB, fn func(context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func resolve(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

func open(t *testing.T) (*gorm.DB, Author) {
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
	a := Author{Name: "Ada"}
	db.Create(&a)
	for _, b := range []Book{
		{Title: "Alpha", Pages: 100, AuthorID: a.ID},
		{Title: "Beta", Pages: 300, AuthorID: a.ID},
		{Title: "Gamma", Pages: 50, AuthorID: a.ID},
	} {
		db.Create(&b)
	}
	return db, a
}

func TestChainedConditions(t *testing.T) {
	db, _ := open(t)
	var books []Book
	db.Where("pages > ?", 80).Where("title <> ?", "Gamma").
		Order("pages desc").Limit(2).Find(&books)

	if len(books) != 2 || books[0].Title != "Beta" || books[1].Title != "Alpha" {
		t.Errorf("got %v", titles(books))
	}
}

func TestOrAndIn(t *testing.T) {
	db, _ := open(t)
	var b1, b2 []Book
	db.Where("title = ?", "Alpha").Or("pages < ?", 60).Find(&b1)
	db.Where("title IN ?", []string{"Alpha", "Beta"}).Find(&b2)
	if len(b1) != 2 || len(b2) != 2 {
		t.Errorf("Or=%d In=%d, want 2 and 2", len(b1), len(b2))
	}
}

func TestAggregateIntoDTO(t *testing.T) {
	db, _ := open(t)
	var out []titleCount
	err := db.Model(&Author{}).
		Select("authors.name as name, count(books.id) as total").
		Joins("left join books on books.author_id = authors.id").
		Group("authors.name").Scan(&out).Error
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "Ada" || out[0].Total != 3 {
		t.Errorf("got %+v, want [{Ada 3}]", out)
	}
}

func TestRawAndExec(t *testing.T) {
	db, _ := open(t)
	var rc []titleCount
	err := db.Raw(`SELECT a.name, count(b.id) AS total FROM authors a
	               LEFT JOIN books b ON b.author_id = a.id GROUP BY a.name`).Scan(&rc).Error
	if err != nil || len(rc) != 1 || rc[0].Total != 3 {
		t.Fatalf("rc=%+v err=%v", rc, err)
	}
	r := db.Exec(`UPDATE books SET pages = pages + 1 WHERE pages > ?`, 80)
	if r.Error != nil || r.RowsAffected != 2 {
		t.Errorf("rows=%d err=%v, want 2", r.RowsAffected, r.Error)
	}
}

func TestPlaceholdersAreParameterised(t *testing.T) {
	db, _ := open(t)
	evil := "Alpha'; DROP TABLE books; --"
	var n, total int64
	db.Model(&Book{}).Where("title = ?", evil).Count(&n)
	db.Model(&Book{}).Count(&total)
	if n != 0 {
		t.Errorf("matched %d rows, want 0", n)
	}
	if total != 3 {
		t.Errorf("table has %d rows, want 3 — it should not have been dropped", total)
	}
}

func TestOnConflictUpsert(t *testing.T) {
	db, _ := open(t)
	if err := db.Create(&Tag{Name: "go", Hits: 1}).Error; err != nil {
		t.Fatal(err)
	}
	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"hits"}),
	}).Create(&Tag{Name: "go", Hits: 5}).Error
	if err != nil {
		t.Fatal(err)
	}
	var got Tag
	db.First(&got, "name = ?", "go")
	var n int64
	db.Model(&Tag{}).Count(&n)
	if got.Hits != 5 || n != 1 {
		t.Errorf("hits=%d rows=%d, want 5 and 1", got.Hits, n)
	}
}

func TestOnConflictDoNothing(t *testing.T) {
	db, _ := open(t)
	db.Create(&Tag{Name: "go"})
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&Tag{Name: "go"}).Error; err != nil {
		t.Fatalf("DoNothing returned %v, want nil", err)
	}
	var n int64
	db.Model(&Tag{}).Count(&n)
	if n != 1 {
		t.Errorf("rows = %d, want 1", n)
	}
}

func TestReturning(t *testing.T) {
	db, a := open(t)
	b := Book{Title: "returned", AuthorID: a.ID}
	err := db.Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&b).Error
	if err != nil || b.ID == uuid.Nil {
		t.Fatalf("err=%v id=%v", err, b.ID)
	}
}

func TestLockingForUpdate(t *testing.T) {
	db, _ := open(t)
	err := db.Transaction(func(tx *gorm.DB) error {
		var locked Book
		return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&locked, "title = ?", "Alpha").Error
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransactionCommitsAndRollsBack(t *testing.T) {
	db, a := open(t)
	ctx := context.Background()

	if err := WithTx(ctx, db, func(ctx context.Context) error {
		return resolve(ctx, db).Create(&Book{Title: "InTx", AuthorID: a.ID}).Error
	}); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&Book{}).Where("title = ?", "InTx").Count(&n)
	if n != 1 {
		t.Errorf("committed rows = %d, want 1", n)
	}

	err := WithTx(ctx, db, func(ctx context.Context) error {
		if e := resolve(ctx, db).Create(&Book{Title: "Doomed", AuthorID: a.ID}).Error; e != nil {
			return e
		}
		return errors.New("business rule")
	})
	if err == nil || err.Error() != "business rule" {
		t.Fatalf("err = %v", err)
	}
	db.Model(&Book{}).Where("title = ?", "Doomed").Count(&n)
	if n != 0 {
		t.Errorf("rolled-back rows = %d, want 0", n)
	}
}

// Nesting must join the outer transaction, not open a second connection
// and deadlock against it.
func TestNestedTransactionJoins(t *testing.T) {
	db, a := open(t)
	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- WithTx(ctx, db, func(ctx context.Context) error {
			return WithTx(ctx, db, func(ctx context.Context) error {
				return resolve(ctx, db).Create(&Book{Title: "Nested", AuthorID: a.ID}).Error
			})
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-timeoutAfterSeconds(10):
		t.Fatal("nested transaction deadlocked")
	}

	var n int64
	db.Model(&Book{}).Where("title = ?", "Nested").Count(&n)
	if n != 1 {
		t.Errorf("rows = %d, want 1", n)
	}
}

func TestToSQLDoesNotExecute(t *testing.T) {
	db, _ := open(t)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("pages > ?", 80).Find(&[]Book{})
	})
	if !strings.Contains(sql, "SELECT") || !strings.Contains(sql, "pages") {
		t.Errorf("ToSQL = %q", sql)
	}
}

func titles(bs []Book) []string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.Title
	}
	return out
}
