package relevance

import "strings"

// Bölüm adından anahtar kelime çıkarımı için küçük, elle derlenmiş sözlük.
// Her giriş, bölüm adında BULUNMASI gereken köklerin (ilk 5 karakter) tümünü
// ve bunlar eşleşince profile eklenen ilgili terimleri tanımlar. Bir bölüm adı
// birden çok girişle eşleşebilir; terimler birleştirilir.
//
// Bu sözlük bir başlangıç noktasıdır: okul, derste/bölümde sözlükte olmayan
// terimleri `department_keywords` alanına yazarak profile kendisi ekleyebilir
// (açık girilen anahtar kelimeler sözlük terimlerinden daha ağır tartılır).

type lexiconEntry struct {
	// All, bölüm adında bulunması gereken katlanmış sözcük kökleri.
	All []string
	// Substr, kök çakışmasını önlemek için (örn. "mobil" ≠ "mobilya") katlanmış
	// bölüm adında aranan alt dizgeler; verilirse hepsi bulunmalıdır.
	Substr []string
	// Terms, profile eklenen ilgili terimler (Türkçe karakterli yazılabilir).
	Terms []string
}

var departmentLexicon = []lexiconEntry{
	{All: []string{"bilgi", "progra"}, Terms: []string{
		"yazılım", "donanım", "denetim", "denetleyici", "algoritma", "programlama", "kod", "derleyici",
		"veri tabanı", "sql", "ağ", "işletim sistemi", "web", "mobil", "uygulama", "nesne", "fonksiyon",
		"değişken", "döngü", "test", "sürüm denetimi", "arayüz", "sunucu", "güvenlik",
	}},
	{All: []string{"bilis"}, Terms: []string{
		"bilişim", "yazılım", "donanım", "ağ", "veri", "sistem", "internet", "siber", "güvenlik",
		"veri tabanı", "sunucu", "bulut", "uygulama", "programlama", "işletim sistemi",
	}},
	{All: []string{"bilgi", "guven"}, Terms: []string{
		"siber", "güvenlik", "şifreleme", "kriptografi", "saldırı", "savunma", "ağ", "zafiyet", "güvenlik duvarı", "adli bilişim",
	}},
	{All: []string{"elekt"}, Terms: []string{
		"elektrik", "elektronik", "devre", "direnç", "kondansatör", "transistör", "akım", "gerilim", "mikrodenetleyici",
		"sensör", "motor", "pano", "otomasyon", "ölçüm", "kablo", "tesisat", "güç", "transformatör",
	}},
	{All: []string{"mekat"}, Terms: []string{
		"mekatronik", "robot", "otomasyon", "sensör", "plc", "motor", "pnömatik", "hidrolik", "devre", "mikrodenetleyici", "kontrol",
	}},
	{All: []string{"makin"}, Terms: []string{
		"makine", "imalat", "torna", "freze", "talaşlı", "kaynak", "tasarım", "malzeme", "dişli", "cnc", "teknik resim", "mekanik",
	}},
	{All: []string{"otomo"}, Terms: []string{
		"otomotiv", "motor", "araç", "şanzıman", "fren", "süspansiyon", "yakıt", "egzoz", "elektrik", "bakım", "onarım", "arıza",
	}},
	{All: []string{"insaa"}, Terms: []string{
		"inşaat", "yapı", "beton", "çelik", "temel", "statik", "proje", "şantiye", "malzeme", "ölçme", "harita", "mimari",
	}},
	{All: []string{"muhas"}, Terms: []string{
		"muhasebe", "vergi", "fatura", "defter", "bilanço", "gelir", "gider", "maliyet", "beyanname", "kdv", "finans", "bordro", "denetim",
	}},
	{All: []string{"banka"}, Terms: []string{
		"banka", "sigorta", "kredi", "mevduat", "finans", "faiz", "risk", "poliçe", "yatırım", "ödeme", "hesap",
	}},
	{All: []string{"buro"}, Terms: []string{
		"büro", "yönetim", "yazışma", "arşiv", "sekreterlik", "protokol", "dosyalama", "iletişim", "ofis", "toplantı", "klavye",
	}},
	{All: []string{"turiz"}, Terms: []string{
		"turizm", "otel", "konaklama", "rehber", "seyahat", "rezervasyon", "ön büro", "kat hizmetleri", "tur", "misafir",
	}},
	{All: []string{"gastr"}, Terms: []string{
		"gastronomi", "mutfak", "yemek", "pişirme", "menü", "hijyen", "servis", "gıda", "pastacılık", "aşçılık", "tarif",
	}},
	{All: []string{"ascil"}, Terms: []string{
		"aşçılık", "mutfak", "yemek", "pişirme", "menü", "hijyen", "servis", "gıda", "tarif", "gastronomi",
	}},
	{All: []string{"grafi"}, Terms: []string{
		"grafik", "tasarım", "görsel", "tipografi", "renk", "logo", "baskı", "illüstrasyon", "fotoğraf", "kompozisyon", "animasyon",
	}},
	{All: []string{"cocuk"}, Terms: []string{
		"çocuk", "gelişim", "eğitim", "oyun", "okul öncesi", "pedagoji", "beslenme", "bakım", "davranış", "öğrenme", "aile",
	}},
	{All: []string{"saglik"}, Terms: []string{
		"sağlık", "hasta", "hemşire", "ilk yardım", "anatomi", "fizyoloji", "hijyen", "tedavi", "ilaç", "enfeksiyon", "bakım",
	}},
	{All: []string{"acil"}, Terms: []string{
		"acil", "ilk yardım", "ambulans", "müdahale", "kurtarma", "travma", "resüsitasyon", "hasta", "taşıma", "afet",
	}},
	{All: []string{"labor"}, Terms: []string{
		"laboratuvar", "numune", "analiz", "deney", "kimya", "biyoloji", "mikroskop", "ölçüm", "reaktif", "kalibrasyon",
	}},
	{All: []string{"tarim"}, Terms: []string{
		"tarım", "bitki", "toprak", "gübre", "sulama", "hasat", "tohum", "hayvan", "üretim", "sera", "zirai", "bahçe",
	}},
	{Substr: []string{"mobilya"}, Terms: []string{
		"mobilya", "ahşap", "ağaç", "dekorasyon", "tasarım", "kesim", "montaj", "malzeme", "imalat", "cila",
	}},
	{All: []string{"moda"}, Terms: []string{
		"moda", "tekstil", "kumaş", "dikiş", "kalıp", "giyim", "model", "tasarım", "konfeksiyon", "iplik",
	}},
	{All: []string{"lojis"}, Terms: []string{
		"lojistik", "depo", "taşıma", "sevkiyat", "stok", "tedarik", "envanter", "nakliye", "gümrük", "dağıtım",
	}},
	{All: []string{"sagli", "guven"}, Terms: []string{
		"iş sağlığı", "güvenlik", "risk", "kaza", "önlem", "mevzuat", "ergonomi", "tehlike", "eğitim", "denetim",
	}},
	{All: []string{"halkl"}, Terms: []string{
		"halkla ilişkiler", "iletişim", "tanıtım", "kampanya", "medya", "kriz", "kurumsal", "etkinlik", "sunum", "imaj",
	}},
	{All: []string{"radyo"}, Terms: []string{
		"radyo", "televizyon", "yayın", "kamera", "kurgu", "ses", "görüntü", "prodüksiyon", "senaryo", "haber",
	}},
	{All: []string{"dis", "ticar"}, Terms: []string{
		"dış ticaret", "ihracat", "ithalat", "gümrük", "lojistik", "sözleşme", "akreditif", "navlun", "pazarlama", "döviz",
	}},
}

// expandDepartment, bölüm adındaki köklerle eşleşen sözlük girişlerinin
// terimlerini birleştirip döndürür.
func expandDepartment(department string) []string {
	present := make(map[string]struct{})
	for _, stem := range Tokenize(department) {
		present[stem] = struct{}{}
	}
	foldedDepartment := Fold(department)
	seen := make(map[string]struct{})
	var terms []string
	for _, entry := range departmentLexicon {
		matched := true
		for _, need := range entry.All {
			if _, ok := present[Stem(need)]; !ok {
				matched = false
				break
			}
		}
		for _, need := range entry.Substr {
			if !strings.Contains(foldedDepartment, need) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		for _, term := range entry.Terms {
			if _, dup := seen[term]; dup {
				continue
			}
			seen[term] = struct{}{}
			terms = append(terms, term)
		}
	}
	return terms
}
