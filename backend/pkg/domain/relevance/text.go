package relevance

import (
	"strings"
	"unicode"
)

// Türkçe için hafif metin işleme: küçük harfe çevirme, ASCII katlama, durak
// sözcükleri ve ilk-5-karakter kök bulma (F5). F5, Türkçe bilgi erişiminde
// morfolojik analiz gerektirmeyen, yaygın ve yeterince iyi bir yaklaşımdır:
// "yazılım", "yazılımı", "yazılımlar" aynı köke düşer.
// Sınırlama: aynı 5 harfle başlayan farklı sözcükler (örn. "bilgisayar" ve
// "bilgilendirme") aynı köke düşer; bu yöntemin bilinen ödünleşimidir.

const stemLength = 5
const minTokenLength = 3

var foldReplacer = strings.NewReplacer(
	"ı", "i", "ş", "s", "ğ", "g", "ü", "u", "ö", "o", "ç", "c",
	"â", "a", "î", "i", "û", "u", "ê", "e",
)

// lowerTR, Türkçe büyük/küçük harf kuralını uygular (İ→i, I→ı); standart
// strings.ToLower "İ"yi "i̇" (birleşik nokta) yapacağından önce elle çevrilir.
func lowerTR(value string) string {
	value = strings.NewReplacer("İ", "i", "I", "ı").Replace(value)
	return strings.ToLower(value)
}

// Fold, metni karşılaştırma için küçük harfli ASCII'ye indirger.
func Fold(value string) string {
	return foldReplacer.Replace(lowerTR(value))
}

// stopwords, ASCII katlanmış Türkçe durak sözcükleridir.
var stopwords = func() map[string]struct{} {
	words := []string{
		"ve", "ile", "veya", "ya", "da", "de", "ki", "mi", "mu", "bu", "su", "o", "bir", "iki",
		"icin", "gibi", "kadar", "daha", "en", "cok", "az", "ise", "ama", "fakat", "ancak",
		"hem", "ne", "nasil", "neden", "nicin", "hangi", "her", "tum", "butun", "bazi", "baska",
		"olarak", "olan", "olur", "olmak", "olup", "oldu", "olduğu", "olduklari", "ise", "iken",
		"uzerine", "uzerinde", "icinde", "arasinda", "sonra", "once", "gore", "ayni", "diger",
		"konu", "konusu", "konular", "ders", "dersi", "dersin", "anlatir", "anlatilir", "anlatilmaktadir",
		"islenir", "islenmektedir", "kapsar", "kapsamaktadir", "hakkinda", "uzerine", "yer", "alir",
		"genel", "giris", "ogrenci", "ogrenciler", "ogrenir", "ogrenilir", "ogretilir",
	}
	set := make(map[string]struct{}, len(words))
	for _, word := range words {
		set[Fold(word)] = struct{}{}
	}
	return set
}()

func isStopword(token string) bool {
	_, found := stopwords[token]
	return found
}

// Stem, katlanmış bir sözcüğü ilk beş karaktere indirir.
func Stem(token string) string {
	runes := []rune(token)
	if len(runes) > stemLength {
		return string(runes[:stemLength])
	}
	return token
}

type token struct {
	Stem    string // karşılaştırma anahtarı (katlanmış, köklenmiş)
	Surface string // kullanıcıya gösterilecek hâli (Türkçe karakterli, küçük harf)
}

func tokenizePairs(text string) []token {
	fields := strings.FieldsFunc(lowerTR(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]token, 0, len(fields))
	for _, field := range fields {
		folded := foldReplacer.Replace(field)
		if len([]rune(folded)) < minTokenLength || isStopword(folded) || isNumber(folded) {
			continue
		}
		tokens = append(tokens, token{Stem: Stem(folded), Surface: field})
	}
	return tokens
}

// Tokenize, metni katlanmış, durak sözcüksüz ve köklenmiş jetonlara böler.
func Tokenize(text string) []string {
	pairs := tokenizePairs(text)
	stems := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		stems = append(stems, pair.Stem)
	}
	return stems
}

func isNumber(token string) bool {
	for _, r := range token {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return token != ""
}

// splitKeywords, virgül/noktalı virgül/satır sonuyla ayrılmış anahtar kelime
// listesini temiz ifadelere böler.
func splitKeywords(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\r' })
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
