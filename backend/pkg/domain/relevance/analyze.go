// Package relevance, bir dersin konularının bölümle ne kadar ilgili olduğunu
// TF-IDF ağırlıklı kosinüs benzerliğiyle ölçer. Yalnızca standart kütüphane
// kullanır; bu yüzden ayrı bir servis çağrısı olmadan (Vercel'de ek function ve
// soğuk başlangıç maliyeti olmadan) backend içinde çalışır.
package relevance

import (
	"math"
	"sort"
	"strings"
)

const (
	MethodTFIDFCosine = "tfidf-cosine"

	BandHigh   = "high"
	BandMedium = "medium"
	BandLow    = "low"
	BandNone   = "none"

	// saturationCosine: kosinüs benzerliği bu değere ulaşınca puan 100 olur.
	// Kısa konu metinleri ile bölüm profili arasındaki ham kosinüs değerleri
	// doğası gereği küçüktür (0.3–0.4 çok güçlü bir eşleşmedir); bu yüzden ham
	// değer 0–100 aralığına doğrusal olarak ölçeklenir. Ayarlanabilir sezgiseldir.
	saturationCosine = 0.35
	highThreshold    = 60.0
	mediumThreshold  = 30.0

	// Profil terimlerinin ağırlıkları: bölüm adı > açık girilen anahtar kelime > sözlük.
	departmentWeight = 3.0
	keywordWeight    = 2.0
	lexiconWeight    = 1.0
	// Konu adı, açıklamadan daha ağır tartılır.
	topicNameWeight        = 2.0
	topicDescriptionWeight = 1.0

	maxMatchedTerms = 5
	maxProfileTerms = 40
)

// Topic, puanlanacak bir konudur.
type Topic struct {
	ID          string
	Name        string
	Description string
	Weight      float64
}

// TopicScore, bir konunun bölümle ilgisidir.
type TopicScore struct {
	TopicID      string   `json:"topic_id"`
	Score        float64  `json:"score"`
	Band         string   `json:"band"`
	MatchedTerms []string `json:"matched_terms"`
}

// Result, bir dersin içerik–bölüm analizidir.
type Result struct {
	Method string `json:"method"`
	// Ready, bölüm adı veya anahtar kelime verilmişse true'dur; aksi halde
	// bölüm bilgisi olmadığından puan hesaplanamaz.
	Ready      bool         `json:"ready"`
	Department string       `json:"department"`
	Terms      []string     `json:"department_terms"`
	Fit        float64      `json:"fit"`
	Topics     []TopicScore `json:"topics"`
}

type document struct {
	counts map[string]float64
	// surface: kök → gösterim hâli (ilk görülen)
	surface map[string]string
}

func newDocument() *document {
	return &document{counts: map[string]float64{}, surface: map[string]string{}}
}

func (doc *document) add(text string, weight float64) {
	for _, tok := range tokenizePairs(text) {
		doc.counts[tok.Stem] += weight
		if _, seen := doc.surface[tok.Stem]; !seen {
			doc.surface[tok.Stem] = tok.Surface
		}
	}
}

// Analyze, konuları bölüm profiline göre puanlar.
func Analyze(department, keywords string, topics []Topic) Result {
	department = strings.TrimSpace(department)
	result := Result{
		Method:     MethodTFIDFCosine,
		Department: department,
		Terms:      []string{},
		Topics:     make([]TopicScore, 0, len(topics)),
	}

	profile := newDocument()
	profile.add(department, departmentWeight)
	displayTerms := orderedSet{}
	for _, tok := range tokenizePairs(department) {
		displayTerms.add(tok.Surface)
	}
	for _, phrase := range splitKeywords(keywords) {
		profile.add(phrase, keywordWeight)
		displayTerms.add(strings.TrimSpace(lowerTR(phrase)))
	}
	for _, term := range expandDepartment(department) {
		profile.add(term, lexiconWeight)
		displayTerms.add(term)
	}
	result.Ready = len(profile.counts) > 0
	result.Terms = displayTerms.limited(maxProfileTerms)

	docs := make([]*document, len(topics))
	for index, topic := range topics {
		doc := newDocument()
		doc.add(topic.Name, topicNameWeight)
		doc.add(topic.Description, topicDescriptionWeight)
		docs[index] = doc
	}

	if !result.Ready {
		for _, topic := range topics {
			result.Topics = append(result.Topics, TopicScore{TopicID: topic.ID, Band: BandNone, MatchedTerms: []string{}})
		}
		return result
	}

	idf := inverseDocumentFrequency(append([]*document{profile}, docs...))
	profileVector := vectorize(profile, idf)

	var weightedSum, weightTotal float64
	allZero := true
	for _, topic := range topics {
		if topic.Weight > 0 {
			allZero = false
			break
		}
	}
	for index, topic := range topics {
		vector := vectorize(docs[index], idf)
		cosine, matched := similarity(profileVector, vector, profile, docs[index])
		score := math.Min(100, cosine/saturationCosine*100)
		score = round1(score)
		result.Topics = append(result.Topics, TopicScore{
			TopicID:      topic.ID,
			Score:        score,
			Band:         band(score),
			MatchedTerms: matched,
		})
		weight := topic.Weight
		if allZero {
			weight = 1
		}
		if weight > 0 {
			weightedSum += score * weight
			weightTotal += weight
		}
	}
	if weightTotal > 0 {
		result.Fit = round1(weightedSum / weightTotal)
	}
	return result
}

func band(score float64) string {
	switch {
	case score >= highThreshold:
		return BandHigh
	case score >= mediumThreshold:
		return BandMedium
	case score > 0:
		return BandLow
	default:
		return BandNone
	}
}

func inverseDocumentFrequency(docs []*document) map[string]float64 {
	documentFrequency := map[string]float64{}
	for _, doc := range docs {
		for stem := range doc.counts {
			documentFrequency[stem]++
		}
	}
	total := float64(len(docs))
	idf := make(map[string]float64, len(documentFrequency))
	for stem, frequency := range documentFrequency {
		idf[stem] = math.Log((1+total)/(1+frequency)) + 1 // düzgünleştirilmiş IDF
	}
	return idf
}

// vectorize: alt-doğrusal TF (1+ln n) × IDF, L2 normalize.
func vectorize(doc *document, idf map[string]float64) map[string]float64 {
	vector := make(map[string]float64, len(doc.counts))
	var norm float64
	for stem, count := range doc.counts {
		value := (1 + math.Log(count)) * idf[stem]
		vector[stem] = value
		norm += value * value
	}
	if norm == 0 {
		return vector
	}
	norm = math.Sqrt(norm)
	for stem := range vector {
		vector[stem] /= norm
	}
	return vector
}

type contribution struct {
	stem  string
	value float64
}

func similarity(a, b map[string]float64, profile, topic *document) (float64, []string) {
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	var shared []contribution
	for stem, value := range a {
		if other, ok := b[stem]; ok {
			product := value * other
			dot += product
			shared = append(shared, contribution{stem: stem, value: product})
		}
	}
	sort.Slice(shared, func(i, j int) bool {
		if shared[i].value != shared[j].value {
			return shared[i].value > shared[j].value
		}
		return shared[i].stem < shared[j].stem
	})
	matched := make([]string, 0, maxMatchedTerms)
	for _, item := range shared {
		if len(matched) == maxMatchedTerms {
			break
		}
		word := topic.surface[item.stem]
		if word == "" {
			word = profile.surface[item.stem]
		}
		if word == "" {
			word = item.stem
		}
		matched = append(matched, word)
	}
	return dot, matched
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }

type orderedSet struct {
	seen  map[string]struct{}
	items []string
}

func (set *orderedSet) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if set.seen == nil {
		set.seen = map[string]struct{}{}
	}
	if _, dup := set.seen[value]; dup {
		return
	}
	set.seen[value] = struct{}{}
	set.items = append(set.items, value)
}

func (set *orderedSet) limited(max int) []string {
	if len(set.items) > max {
		return append([]string(nil), set.items[:max]...)
	}
	if set.items == nil {
		return []string{}
	}
	return set.items
}
