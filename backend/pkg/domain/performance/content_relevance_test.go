package performance

import "testing"

func TestAnnotateRelevanceAddsScoresOnlyForCoursesWithADepartment(t *testing.T) {
	input := &StudentInput{Courses: []CourseRef{
		{
			ID:         "c1",
			Department: "Bilgisayar Programcılığı",
			Topics: []TopicRef{
				{ID: "t-prog", Name: "Değişkenler ve veri tipleri", Description: "Programlamada değişken tanımlama, veri tipleri ve algoritma mantığı.", Weight: 1},
				{ID: "t-hist", Name: "Osmanlı tarihi", Description: "Tanzimat dönemi reformları ve padişahlar.", Weight: 1},
			},
		},
		{
			ID:     "c2",
			Topics: []TopicRef{{ID: "t-other", Name: "Konu", Description: "Bölümü olmayan bir dersin konusu.", Weight: 1}},
		},
	}}
	result := &StudentResult{
		Courses: []CourseResult{
			{CourseID: "c1", Topics: []TopicResult{{TopicID: "t-prog"}, {TopicID: "t-hist"}}},
			{CourseID: "c2", Topics: []TopicResult{{TopicID: "t-other"}}},
		},
		WeakAreas: []TopicResult{{TopicID: "t-hist"}, {TopicID: "t-unknown"}},
	}

	annotateRelevance(input, result)

	first := result.Courses[0]
	if first.DepartmentFit == nil || *first.DepartmentFit <= 0 {
		t.Fatalf("course with a department should get a positive fit, got %v", first.DepartmentFit)
	}
	if first.Topics[0].Relevance == nil || first.Topics[1].Relevance == nil {
		t.Fatal("every topic of an annotated course should get a relevance value")
	}
	if *first.Topics[0].Relevance <= *first.Topics[1].Relevance {
		t.Fatalf("programming topic (%.1f) must outrank history (%.1f)", *first.Topics[0].Relevance, *first.Topics[1].Relevance)
	}
	if *first.Topics[1].Relevance != 0 {
		t.Fatalf("unrelated topic should score 0, got %.1f", *first.Topics[1].Relevance)
	}
	second := result.Courses[1]
	if second.DepartmentFit != nil || second.Topics[0].Relevance != nil {
		t.Fatal("a course without a department must stay unannotated (nil), not 0")
	}
	if result.WeakAreas[0].Relevance == nil {
		t.Fatal("weak areas that match a scored topic should be annotated")
	}
	if result.WeakAreas[1].Relevance != nil {
		t.Fatal("weak areas with an unknown topic must stay nil")
	}
}

func TestAnnotateRelevanceIsSafeWithNilsAndNoDepartments(t *testing.T) {
	annotateRelevance(nil, nil)
	annotateRelevance(&StudentInput{}, nil)
	result := &StudentResult{Courses: []CourseResult{{CourseID: "c"}}}
	annotateRelevance(&StudentInput{Courses: []CourseRef{{ID: "c"}}}, result)
	if result.Courses[0].DepartmentFit != nil {
		t.Fatal("no department means no annotation")
	}
}
