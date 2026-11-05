package relevance

import (
	"reflect"
	"strings"
	"testing"
)

func TestFoldHandlesTurkishCase(t *testing.T) {
	cases := map[string]string{
		"İstanbul":      "istanbul",
		"IŞIK":          "isik",
		"Öğrenci":       "ogrenci",
		"Çalışma Ğ Ü":   "calisma g u",
		"Yazılım":       "yazilim",
		"PROGRAMCILIĞI": "programciligi",
	}
	for input, want := range cases {
		if got := Fold(input); got != want {
			t.Errorf("Fold(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTokenizeDropsStopwordsNumbersAndShortTokens(t *testing.T) {
	got := Tokenize("Yazılımı ve donanımlar için 2026 bir ab")
	want := []string{"yazil", "donan"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tokenize = %v, want %v", got, want)
	}
}

func TestStemmingGroupsInflections(t *testing.T) {
	a, b, c := Tokenize("yazılım"), Tokenize("yazılımı"), Tokenize("yazılımlarımız")
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(a, c) {
		t.Fatalf("inflections should share a stem: %v %v %v", a, b, c)
	}
	if !reflect.DeepEqual(Tokenize("programcılığı"), Tokenize("programlama")) {
		t.Fatal("programcılığı and programlama should share the F5 stem")
	}
}

func TestExpandDepartment(t *testing.T) {
	terms := strings.Join(expandDepartment("Bilgisayar Programcılığı"), "|")
	for _, want := range []string{"yazılım", "donanım", "denetim", "algoritma"} {
		if !strings.Contains(terms, want) {
			t.Errorf("Bilgisayar Programcılığı should expand to include %q, got %s", want, terms)
		}
	}
	if len(expandDepartment("Bilişim Teknolojileri")) == 0 {
		t.Error("Bilişim Teknolojileri should match the lexicon")
	}
	if len(expandDepartment("Elektrik-Elektronik Teknolojisi")) == 0 {
		t.Error("Elektrik-Elektronik should match the lexicon")
	}
	if len(expandDepartment("Bilinmeyen Bir Bölüm Adı")) != 0 {
		t.Error("unknown departments must not expand")
	}
	if !strings.Contains(strings.Join(expandDepartment("Mobilya ve Dekorasyon"), "|"), "ahşap") {
		t.Error("Mobilya ve Dekorasyon should expand to woodworking terms")
	}
	if strings.Contains(strings.Join(expandDepartment("Mobil Uygulama Geliştirme"), "|"), "ahşap") {
		t.Error("'mobil' must not be confused with 'mobilya'")
	}
	if len(expandDepartment("İş Sağlığı ve Güvenliği")) == 0 {
		t.Error("İş Sağlığı ve Güvenliği should match the lexicon")
	}
}

func score(result Result, id string) TopicScore {
	for _, item := range result.Topics {
		if item.TopicID == id {
			return item
		}
	}
	return TopicScore{TopicID: "missing"}
}

func TestAnalyzeRanksRelevantTopicsAboveIrrelevantOnes(t *testing.T) {
	topics := []Topic{
		{ID: "vars", Name: "Değişkenler ve veri tipleri", Description: "Programlamada değişken tanımlama, tamsayı ve metin veri tipleri, algoritma mantığı.", Weight: 1},
		{ID: "net", Name: "Ağ temelleri", Description: "Bilgisayar ağlarında IP adresleme, yönlendirme ve sunucu kavramları.", Weight: 1},
		{ID: "hist", Name: "Osmanlı tarihi", Description: "Tanzimat dönemi reformları, padişahlar ve devlet yapısı.", Weight: 1},
	}
	result := Analyze("Bilgisayar Programcılığı", "", topics)
	if !result.Ready || result.Method != MethodTFIDFCosine {
		t.Fatalf("unexpected header: %+v", result)
	}
	vars, net, hist := score(result, "vars"), score(result, "net"), score(result, "hist")
	t.Logf("vars=%.1f(%s) net=%.1f(%s) hist=%.1f(%s) fit=%.1f", vars.Score, vars.Band, net.Score, net.Band, hist.Score, hist.Band, result.Fit)
	if vars.Score <= net.Score || net.Score <= hist.Score {
		t.Fatalf("expected vars > net > hist, got %.1f %.1f %.1f", vars.Score, net.Score, hist.Score)
	}
	if hist.Score != 0 || hist.Band != BandNone {
		t.Fatalf("an unrelated topic must score 0, got %.1f (%s)", hist.Score, hist.Band)
	}
	if vars.Band != BandHigh && vars.Band != BandMedium {
		t.Fatalf("a clearly relevant topic should be medium/high, got %s (%.1f)", vars.Band, vars.Score)
	}
	if len(vars.MatchedTerms) == 0 {
		t.Fatal("matched terms should explain the score")
	}
	joined := strings.Join(vars.MatchedTerms, " ")
	if !strings.Contains(joined, "değişken") && !strings.Contains(joined, "algoritma") {
		t.Fatalf("matched terms should keep Turkish surface forms, got %v", vars.MatchedTerms)
	}
}

func TestExplicitKeywordsExtendTheProfile(t *testing.T) {
	topics := []Topic{
		{ID: "hid", Name: "Hidrolik sistemler", Description: "Hidrolik pompalar ve pnömatik devre elemanları.", Weight: 1},
		{ID: "oth", Name: "Edebiyat", Description: "Roman ve hikâye türleri.", Weight: 1},
	}
	without := Analyze("Özel Bir Alan", "", topics)
	with := Analyze("Özel Bir Alan", "hidrolik, pnömatik", topics)
	if score(without, "hid").Score != 0 {
		t.Fatalf("unknown department without keywords must score 0, got %.1f", score(without, "hid").Score)
	}
	if score(with, "hid").Score <= 0 || score(with, "oth").Score != 0 {
		t.Fatalf("keywords should lift only the related topic: %.1f / %.1f", score(with, "hid").Score, score(with, "oth").Score)
	}
}

func TestAnalyzeWithoutDepartmentIsNotReady(t *testing.T) {
	result := Analyze("  ", "", []Topic{{ID: "a", Name: "Programlama", Description: "Kod yazma", Weight: 1}})
	if result.Ready || result.Fit != 0 || len(result.Topics) != 1 || result.Topics[0].Band != BandNone {
		t.Fatalf("expected not ready with zero scores, got %+v", result)
	}
	if result.Terms == nil || result.Topics[0].MatchedTerms == nil {
		t.Fatal("slices must be non-nil so JSON renders [] rather than null")
	}
}

func TestFitIsWeightedByTopicWeight(t *testing.T) {
	topics := []Topic{
		{ID: "rel", Name: "Yazılım geliştirme", Description: "Programlama ve algoritma", Weight: 3},
		{ID: "irr", Name: "Osmanlı tarihi", Description: "Tanzimat dönemi", Weight: 1},
	}
	result := Analyze("Bilgisayar Programcılığı", "", topics)
	rel := score(result, "rel").Score
	want := round1(rel * 3 / 4)
	if result.Fit != want {
		t.Fatalf("fit = %.1f, want %.1f (relevant score %.1f weighted 3 of 4)", result.Fit, want, rel)
	}
	// Tüm ağırlıklar sıfırsa eşit ağırlık kullanılır.
	topics[0].Weight, topics[1].Weight = 0, 0
	equal := Analyze("Bilgisayar Programcılığı", "", topics)
	if equal.Fit != round1(rel/2) {
		t.Fatalf("equal-weight fit = %.1f, want %.1f", equal.Fit, round1(rel/2))
	}
}

func TestAnalyzeIsDeterministicAndHandlesEmptyTopics(t *testing.T) {
	topics := []Topic{
		{ID: "a", Name: "Veri tabanı", Description: "SQL sorguları ve tablolar", Weight: 1},
		{ID: "b", Name: "Ağ", Description: "Sunucu ve istemci", Weight: 1},
	}
	first := Analyze("Bilgisayar Programcılığı", "web", topics)
	second := Analyze("Bilgisayar Programcılığı", "web", topics)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("Analyze must be deterministic")
	}
	empty := Analyze("Bilgisayar Programcılığı", "", nil)
	if !empty.Ready || len(empty.Topics) != 0 || empty.Fit != 0 {
		t.Fatalf("no topics should yield an empty, ready result: %+v", empty)
	}
	blank := Analyze("Bilgisayar Programcılığı", "", []Topic{{ID: "x", Name: "", Description: "", Weight: 1}})
	if blank.Topics[0].Score != 0 || blank.Topics[0].Band != BandNone {
		t.Fatalf("a topic without text must score 0: %+v", blank.Topics[0])
	}
}

func TestScoresStayWithinBounds(t *testing.T) {
	topics := []Topic{{ID: "x", Name: "Yazılım donanım denetim algoritma programlama", Description: strings.Repeat("yazılım donanım algoritma ", 20), Weight: 1}}
	result := Analyze("Bilgisayar Programcılığı", "yazılım, donanım", topics)
	got := result.Topics[0].Score
	if got < 0 || got > 100 || result.Fit < 0 || result.Fit > 100 {
		t.Fatalf("scores must stay within 0..100, got %.1f / %.1f", got, result.Fit)
	}
	t.Logf("saturated topic score = %.1f", got)
}
