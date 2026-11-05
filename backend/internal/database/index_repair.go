package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// GORM'un AutoMigrate'i, aynı ada sahip bir indeks zaten varsa onu yeniden
// oluşturmaz. Bir modeldeki UNIQUE indeks etiketi sonradan düzeltilse bile
// eski veritabanı dosyalarında yanlış (eksik kolonlu) indeks kalır. Burada
// bilinen yanlış indeksler, beklenen kolon listesiyle yeniden oluşturulur.
// Yeni indeks eskisinden daha gevşektir; mevcut veri asla yeni kuralı ihlal etmez.
type uniqueIndexSpec struct {
	Name    string
	Table   string
	Columns []string
	Where   string
}

var uniqueIndexSpecs = []uniqueIndexSpec{
	// Bir performans snapshot'ı birden çok ders satırı içerebilir.
	{Name: "idx_snapshot_courses", Table: "snapshot_courses", Columns: []string{"snapshot_id", "course_id"}, Where: "deleted_at IS NULL"},
	{Name: "idx_snapshot_topics", Table: "snapshot_topics", Columns: []string{"snapshot_id", "course_id", "topic_id"}, Where: "deleted_at IS NULL"},
	// Aynı okulda farklı seviyelerde aynı harfli şube (1-A, 2-A) olabilir.
	{Name: "idx_classes_school_level_no", Table: "classes", Columns: []string{"school_id", "level", "no"}},
}

func normalizeIndexSQL(definition string) string {
	replacer := strings.NewReplacer("`", "", "\"", "", "[", "", "]", "", " ", "", "\t", "", "\n", "", "\r", "")
	return strings.ToLower(replacer.Replace(definition))
}

// indexCoversColumns, indeks tanımının kolon listesinin tam olarak beklenen
// liste olup olmadığını söyler: "(a,b)" aranır, "(a,b,c)" veya "(x,a,b)" eşleşmez.
func indexCoversColumns(definition string, columns []string) bool {
	return strings.Contains(normalizeIndexSQL(definition), "("+strings.Join(columns, ",")+")")
}

func (spec uniqueIndexSpec) createStatement() string {
	statement := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s(%s)", spec.Name, spec.Table, strings.Join(spec.Columns, ", "))
	if spec.Where != "" {
		statement += " WHERE " + spec.Where
	}
	return statement
}

// RepairUniqueIndexes tekrar çalıştırılabilir (idempotent) ve doğru indeksler
// için yalnızca bir sorgu çalıştırır.
func RepairUniqueIndexes(db *gorm.DB) error {
	for _, spec := range uniqueIndexSpecs {
		var definition string
		if err := db.Raw("SELECT COALESCE(sql, '') FROM sqlite_master WHERE type = 'index' AND name = ?", spec.Name).Scan(&definition).Error; err != nil {
			return fmt.Errorf("inspect index %s: %w", spec.Name, err)
		}
		if definition != "" && indexCoversColumns(definition, spec.Columns) {
			continue
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("DROP INDEX IF EXISTS " + spec.Name).Error; err != nil {
				return err
			}
			return tx.Exec(spec.createStatement()).Error
		})
		if err != nil {
			return fmt.Errorf("repair index %s: %w", spec.Name, err)
		}
	}
	return nil
}
