package performance

import (
	"strings"

	"lms/backend/pkg/domain/relevance"
)

// annotateRelevance, öğrenci sonucuna konu–bölüm ilgisini (TF-IDF/kosinüs) ekler.
// Hesap backend içinde yapılır (ek servis çağrısı yok) ve yalnızca bölüm adı veya
// anahtar kelimesi tanımlı dersler için çalışır; diğerlerinin alanları boş kalır.
// Mevcut performans puanlarına dokunmaz: yalnızca `relevance` ve
// `department_fit` alanlarını doldurur.
func annotateRelevance(input *StudentInput, result *StudentResult) {
	if input == nil || result == nil {
		return
	}
	topicScores := make(map[string]float64)
	courseFit := make(map[string]float64)
	for _, course := range input.Courses {
		if strings.TrimSpace(course.Department) == "" && strings.TrimSpace(course.DepartmentKeywords) == "" {
			continue
		}
		topics := make([]relevance.Topic, 0, len(course.Topics))
		for _, topic := range course.Topics {
			topics = append(topics, relevance.Topic{ID: topic.ID, Name: topic.Name, Description: topic.Description, Weight: topic.Weight})
		}
		analysis := relevance.Analyze(course.Department, course.DepartmentKeywords, topics)
		if !analysis.Ready {
			continue
		}
		courseFit[course.ID] = analysis.Fit
		for _, item := range analysis.Topics {
			topicScores[item.TopicID] = item.Score
		}
	}
	if len(courseFit) == 0 {
		return
	}
	for index := range result.Courses {
		course := &result.Courses[index]
		fit, found := courseFit[course.CourseID]
		if !found {
			continue
		}
		value := fit
		course.DepartmentFit = &value
		for position := range course.Topics {
			setRelevance(&course.Topics[position], topicScores)
		}
	}
	for index := range result.WeakAreas {
		setRelevance(&result.WeakAreas[index], topicScores)
	}
}

func setRelevance(topic *TopicResult, scores map[string]float64) {
	if score, found := scores[topic.TopicID]; found {
		value := score
		topic.Relevance = &value
	}
}
