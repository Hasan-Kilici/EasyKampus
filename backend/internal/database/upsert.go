package database

import (
	"gorm.io/gorm/clause"
)

// liveRowTarget, partial unique index'lerde ON CONFLICT hedefinin
// "WHERE deleted_at IS NULL" kısmını üretir. GORM'un OnConflict.Where alanı
// DO UPDATE sonrasına yazıldığı için hedef koşulu TargetWhere olarak verilir;
// aksi halde SQLite eşleşen kısıt bulamaz.
func liveRowTarget() clause.Where {
	return clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}}
}

func liveRowConflict(columns []string, updates clause.Set) clause.OnConflict {
	target := make([]clause.Column, 0, len(columns))
	for _, name := range columns {
		target = append(target, clause.Column{Name: name})
	}
	return clause.OnConflict{Columns: target, DoUpdates: updates, TargetWhere: liveRowTarget()}
}
