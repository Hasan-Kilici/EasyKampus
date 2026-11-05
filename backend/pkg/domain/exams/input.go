// Package exams, sınavın (quiz + sorular + seçenekler) bütünsel iş akışıdır:
// atomik oluşturma/düzenleme, yayın öncesi doğrulama, onay, öğrenci için
// cevap anahtarsız görünüm, sonuç ve öğretmen özeti. Kurallar saf fonksiyonlardır;
// kalıcılık Store arayüzü ile soyutlanır.
package exams

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxQuestions      = 200
	MinOptions        = 2
	MaxOptions        = 10
	MaxQuestionPoints = 100.0
	MaxTotalPoints    = 10000.0
	MaxNameLength     = 200
	MaxDuration       = 600
	MaxAttemptsLimit  = 20
)

// OptionInput bir seçenektir; Answer, doğru cevabı işaretler.
type OptionInput struct {
	Content string `json:"content"`
	Answer  bool   `json:"answer"`
}

// QuestionInput bir sorudur. Sıra, dizideki sıradır.
type QuestionInput struct {
	TopicID  string        `json:"topic_id"`
	Question string        `json:"question"`
	Points   float64       `json:"points"`
	Options  []OptionInput `json:"options"`
}

// Input, bir sınavın tamamıdır.
type Input struct {
	CourseID        string          `json:"course_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	OpenAt          *time.Time      `json:"open_at"`
	CloseAt         *time.Time      `json:"close_at"`
	DurationMinutes int             `json:"duration_minutes"`
	MaxAttempts     int             `json:"max_attempts"`
	Questions       []QuestionInput `json:"questions"`
}

// Issue, kullanıcıya gösterilecek bir eksik/hatadır; Path alan yolunu verir
// (örn. "questions[2].options").
type Issue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// TotalPoints, soru puanlarının toplamıdır (0 puanlı soru 1 sayılır).
func TotalPoints(in Input) float64 {
	total := 0.0
	for _, question := range in.Questions {
		total += effectivePoints(question.Points)
	}
	return total
}

func effectivePoints(points float64) float64 {
	if points == 0 {
		return 1
	}
	return points
}

// CheckDraft, taslak kaydı için gereken yapısal kuralları denetler: eksik
// sorular/seçenekler taslakta serbesttir, ama sınırlar ve tutarsızlıklar değil.
func CheckDraft(in Input) []Issue { return check(in, false) }

// Check, yayına/onaya göndermek için gereken tam kuralları denetler.
func Check(in Input) []Issue { return check(in, true) }

func check(in Input, strict bool) []Issue {
	var issues []Issue
	add := func(path, format string, args ...any) {
		issues = append(issues, Issue{Path: path, Message: fmt.Sprintf(format, args...)})
	}

	name := strings.TrimSpace(in.Name)
	switch {
	case name == "":
		add("name", "Sınav adı boş olamaz.")
	case utf8.RuneCountInString(name) > MaxNameLength:
		add("name", "Sınav adı en fazla %d karakter olabilir.", MaxNameLength)
	}
	if utf8.RuneCountInString(in.Description) > 2000 {
		add("description", "Açıklama en fazla 2000 karakter olabilir.")
	}
	if in.DurationMinutes < 0 || in.DurationMinutes > MaxDuration {
		add("duration_minutes", "Süre 0–%d dakika arasında olmalı (0 = süresiz).", MaxDuration)
	}
	if in.MaxAttempts < 1 || in.MaxAttempts > MaxAttemptsLimit {
		add("max_attempts", "Deneme hakkı 1–%d arasında olmalı.", MaxAttemptsLimit)
	}
	if in.OpenAt != nil && in.CloseAt != nil && in.CloseAt.Before(*in.OpenAt) {
		add("close_at", "Kapanış zamanı açılış zamanından önce olamaz.")
	}
	if len(in.Questions) > MaxQuestions {
		add("questions", "En fazla %d soru eklenebilir.", MaxQuestions)
	}
	if strict && len(in.Questions) == 0 {
		add("questions", "En az bir soru ekleyin.")
	}
	if TotalPoints(in) > MaxTotalPoints {
		add("questions", "Toplam puan %.0f'ı aşamaz.", MaxTotalPoints)
	}

	for index, question := range in.Questions {
		number := index + 1
		base := fmt.Sprintf("questions[%d]", index)
		if question.Points < 0 || question.Points > MaxQuestionPoints {
			add(base+".points", "Soru %d: puan 0–%.0f arasında olmalı.", number, MaxQuestionPoints)
		}
		if len(question.Options) > MaxOptions {
			add(base+".options", "Soru %d: en fazla %d seçenek olabilir.", number, MaxOptions)
		}
		if !strict {
			continue
		}
		if strings.TrimSpace(question.Question) == "" {
			add(base+".question", "Soru %d: soru metni boş.", number)
		}
		if len(question.Options) < MinOptions {
			add(base+".options", "Soru %d: en az %d seçenek gerekli.", number, MinOptions)
		}
		seen := make(map[string]struct{}, len(question.Options))
		correct := 0
		for position, option := range question.Options {
			content := strings.TrimSpace(option.Content)
			if content == "" {
				add(fmt.Sprintf("%s.options[%d].content", base, position), "Soru %d: %d. seçenek boş.", number, position+1)
				continue
			}
			key := strings.ToLower(content)
			if _, duplicate := seen[key]; duplicate {
				add(fmt.Sprintf("%s.options[%d].content", base, position), "Soru %d: aynı seçenek birden fazla kez yazılmış.", number)
			}
			seen[key] = struct{}{}
			if option.Answer {
				correct++
			}
		}
		if correct != 1 {
			add(base+".options", "Soru %d: tam olarak bir doğru seçenek işaretleyin.", number)
		}
	}
	return issues
}
