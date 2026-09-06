package sgorm

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type userExample struct {
	Model `gorm:"embedded"`

	Name   string `gorm:"type:varchar(40);unique_index;not null" json:"name"`
	Age    int    `gorm:"not null" json:"age"`
	Gender string `gorm:"type:varchar(10);not null" json:"gender"`
}

func TestGetTableName(t *testing.T) {
	name := GetTableName(&userExample{})
	assert.NotEmpty(t, name)

	name = GetTableName(userExample{})
	assert.NotEmpty(t, name)

	name = GetTableName("table")
	assert.Empty(t, name)
}

func TestBaseModelsDeleteWithoutDeletedAtColumn(t *testing.T) {
	models := []struct {
		name  string
		value any
	}{
		{"camel case", &struct {
			BaseModel
			Name string
		}{Name: "test"}},
		{"snake case", &struct {
			BaseModel2
			Name string
		}{Name: "test"}},
	}
	for _, model := range models {
		t.Run(model.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
			require.NoError(t, err)
			connection, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = connection.Close() })
			require.NoError(t, db.Exec("CREATE TABLE records (id INTEGER PRIMARY KEY, created_at DATETIME, updated_at DATETIME, name TEXT)").Error)
			require.NoError(t, db.Table("records").Create(model.value).Error)
			require.NoError(t, db.Table("records").First(model.value).Error)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return tx.Table("records").Delete(model.value).Error }))
			var count int64
			require.NoError(t, db.Table("records").Count(&count).Error)
			require.Zero(t, count)
		})
	}
}
