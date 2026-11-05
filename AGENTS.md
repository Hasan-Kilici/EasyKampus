# AGENTS.md — LMS (Kampüs Yönetimi + Performans + EFOR)

Bu dosya bu depoda çalışan her AI ajanı için bağlayıcı çalışma kılavuzudur. Önce **"Değişmez Kurallar"** ve **"Vercel Kısıtları"** bölümlerini oku; kalan bölümler başvuru içindir.

## 1. Projenin nihai amacı

Okul/kampüs odaklı bir LMS. Ürün üç ayağa dayanır:

1. **Kampüs yönetimi:** Okul → Kampüs → Sınıf → Şube hiyerarşisi; okul, kampüs, sınıf ve şube ekleme/yönetme; roller ve kapsamlı yetkilendirme.
2. **Performans:** Ders/bölüm bazlı öğrenci ve öğretmen performansı; içerik puanlandırma (içeriğin bölümle ilişkisi üzerinden).
3. **EFOR sistemi.**
4. **Online sınav (quiz) sistemi:** sınav kurma, onay, öğrencinin çevrimiçi çözmesi, otomatik puanlama ve sonucun performansa yansıması.

Hedefteki ek özellikler arasında sınıf numarasına göre **otomatik öğrenci atama** vardır (tek → A grubu, çift → B grubu).

Bu bölümdeki her şey hedeftir; hepsi kodda yoktur. Gerçek durum için bölüm 2'ye bak.

### 1.1 ⭐ Şu anki öncelik: Performans sistemi

**Performans sistemi projenin en yüksek öncelikli parçasıdır** ve proje şu an onu **canlıda (Vercel'de) aktif etmeye** odaklanmıştır. Başka bir iş açıkça istenmedikçe, belirsiz durumlarda kararı performans sistemini çalışır hale getirecek yönde ver.

Durum (rapora ve koda göre):

- **Yerelde tamam ve doğrulanmış:** FastAPI servisi (16 test geçiyor), sözleşme v1, backend entegrasyonu, snapshot kaydı/budama, önbellek, 0–100 oranlar. Yerel smoke testinde öğrenci ve öğretmen analizi çalışıyor; servis kapalıyken backend doğru şekilde 503 dönüyor.
- **Canlıda aktif etme sürüyor:** Backend ile performans servisi Vercel'de **iki ayrı function** olarak çalışır; aralarındaki HTTP çağrısı canlıda henüz kanıtlanmış sayılmaz.

**Kodda yapılanlar** (birim testle doğrulandı; canlıda doğrulanmadı):

- **Servis adresi otomatik bulunur.** `PERFORMANCE_SERVICE_URL` tanımlıysa o kullanılır. Tanımsızsa: Vercel dışında ve `APP_ENV=production` değilken yerel varsayılan `http://127.0.0.1:8000` kullanılır (eski bir `.env` yüzünden adresin boş kalıp her çağrıda hiç istek atmadan 503 dönmesi bu yüzden artık olmaz). Tanımsız ve Vercel'de çalışılıyorsa servis aynı projenin kendi adresinde aranır: production'da `VERCEL_PROJECT_PRODUCTION_URL`, değilse `VERCEL_URL` (`internal/config/performance.go`). `.env.example`'daki yerel değer (`http://127.0.0.1:8000`) Vercel'e **kopyalanmamalı**; Vercel'de bu değişkeni boş bırak ya da gerçek adresi ver.
- **Zaman aşımı:** `PERFORMANCE_TIMEOUT` yoksa varsayılan yerelde 15 sn, Vercel'de **8 sn**.
- **Hata eşlemesi düzeltildi** (`internal/analytics/client.go`): 401/403/404/408/429, 3xx, 5xx ve "2xx ama JSON olmayan gövde" (örn. koruma/giriş sayfası) artık `ErrUnavailable` → **503** olur ve log'a düşer. Yalnızca gerçek sözleşme/doğrulama hataları (400/413/422) `ErrValidation` kalır. Yönlendirme takip edilmez.
- **Herkese açık uç nokta koruması (opsiyonel):** Vercel'de `/api/analyze/*` herkese açık URL'dedir; bu, ücretsiz kotayı tüketmeye açıktır. `PERFORMANCE_INTERNAL_TOKEN` tanımlanırsa Python servisi `X-Internal-Token` başlığını zorunlu kılar (yoksa 401) ve backend aynı değeri gönderir. Tanımsızsa davranış eskisi gibidir (yerel geliştirme). `/api/health` açık kalır.
- **Dağıtım koruması bypass'ı (opsiyonel):** `PERFORMANCE_BYPASS_SECRET` (veya Vercel'in `VERCEL_AUTOMATION_BYPASS_SECRET` değişkeni) tanımlıysa `x-vercel-protection-bypass` başlığı gönderilir.
- **Teşhis logları:** Backend açılışta `analytics: performance service at <url>` (veya adres boşsa `PERFORMANCE_SERVICE_URL is empty` uyarısı) yazar; istek sırasında bağlantı hatası `analytics <yol> request failed: ...`, kötü yanıt `analytics <yol> responded <kod>: ...` olarak loglanır. 503 gördüğünde önce bunlara bak.
- **Python giriş noktası** (`performance/api/index.py`) `performance/` klasörünü `sys.path`'e kendisi ekler; `requirements.txt` başındaki UTF-8 BOM silindi.

**Yerelde `503 dependency unavailable` alırsan** (sırayla kontrol et):

1. Python servisi çalışıyor mu? Tarayıcıda `http://127.0.0.1:8000/api/health` `{"status":"ok",...}` dönmeli. Başlatmak için: `cd performance` → `python -m app.main` (sanal ortamdaysan önce etkinleştir; bağımlılıklar için `pip install -r requirements-dev.txt`, bu dosya `requirements.txt`'i de çeker).
2. Backend açılış log'u `analytics: performance service at http://127.0.0.1:8000` diyor mu? `is empty` diyorsa `APP_ENV=production` ayarlı ya da `PERFORMANCE_SERVICE_URL` bilerek boş.
3. İstek anındaki log: `request failed: ... connection refused` → servis kapalı/port yanlış; `responded 401/403/404/5xx` → servise ulaşıldı ama kabul etmedi (token, yanlış adres, servis çökmesi); `non-JSON body` → adreste başka bir şey yanıt veriyor.
4. `PERFORMANCE_INTERNAL_TOKEN` yalnızca bir tarafta tanımlıysa servis 401 verir; iki tarafta aynı olmalı ya da ikisinde de tanımsız olmalı.
5. Eski `backend/.env` dosyası `.env.example`'daki yeni anahtarları içermeyebilir; karşılaştır.

**Canlıda hâlâ doğrulanması gerekenler** (sahibi Vercel'de dener):

1. **Rota sırası:** `vercel.json` içinde `/api/analyze/*` ve `/api/health` Python'a gider; canlıda `/api/health` backend'i değil performans servisini yoklar. Sırayı bozma.
2. **Deployment Protection:** Preview dağıtımlarında koruma açıksa servis çağrısı 401 alır (artık 503 görünür, log'da `analytics ... responded 401`). Çözüm: bypass secret'ı tanımla ya da production alan adını kullan.
3. **Süre bütçesi ve soğuk başlangıç:** İlk performans isteği iki function'ın soğuk başlangıcını içerir. Hesabı her istekte yaptırma; snapshot ve cache'i kullan.
4. **Cache instance başınadır.** `Invalidate*` çağrıları yalnızca o instance'ın belleğini temizler; diğer instance'lar bayat sonucu en çok `PERFORMANCE_CACHE_TTL` (5 dk) kadar sunabilir. `cached=true` canlıda her zaman dönmeyebilir; bu hata değildir.
5. **Python bağımlılıkları:** `performance/requirements.txt` (fastapi, pydantic **2.13.5**, `uvicorn[standard]`). Test bağımlılıkları (`httpx`, `pytest`) yalnızca `requirements-dev.txt`'tedir; function paketine taşıma. Python sürümü Vercel'de sabitlenmemiştir; sürüm kayarsa pin'ler uyumsuz olabilir.
6. **Snapshot'lar veritabanına yazılır.** Canlı veritabanı kalıcılığı çözülmeden (bölüm 4'teki kritik soru) snapshot kalıcılığı güvenilir değildir; bu performansı aktif etmenin önünde bir engeldir. Ajan olarak kendi başına çözmeye çalışma, sahibine sor.
7. **Contract sürümü** başlığı (`X-Contract-Version`) her çağrıda gider. İki tarafın sözleşmesini birlikte değiştir (bölüm 8).

Performans ile ilgili her değişiklikte sırayla doğrula: `pytest` (performance/), `go test ./...` (backend/), mümkünse yerel smoke (servis açıkken öğrenci/öğretmen analizi + ikinci çağrıda `cached=true`; servis kapalıyken 503).

## 2. Mevcut durum ve hedef farkı

| Konu | Durum | Not |
|---|---|---|
| Okul/kampüs/sınıf/ders/ödev/kayıt/program/devamsızlık/not/not defteri/quiz/video | **Var** | Backend + frontend CRUD ve toplu işlemler (kayıt, devamsızlık, not). |
| Performans analizi (öğrenci/öğretmen, snapshot, 0–100 oranlar) | **Var** | Ayrı FastAPI servisi, sözleşme v1. |
| Yetki kapsamı (scope) | **Var** | Boş kapsam = hiçbir satır (bölüm 7). |
| **EFOR sistemi** | **Kısmen (iş yükü → AKTS)** | Konu/ödev/proje saatleri ve AKTS hesabı var (bölüm 2.2). EFOR'un başka bir boyutu (örn. öğrenci eforu) tanımlanmadı; `activities` paketi ham girdi olabilir (varsayım). |
| **Tek/çift otomatik şube ataması** | **Kodda yok** | Bugün `workflows/student_registrerar.go` öğrenciyi kampüsteki en eski sınıfa koyar, yoksa "Genel" sınıfı açar. `Class.No` tek harftir (A, B, …). *Hangi numaranın* (öğrenci numarası mı?) paritesine bakılacağı belirsiz: uygulamadan önce sahibine sor. |
| **Öğrenci erişimi (IDOR) ve rol bazlı liste kapsamı** | **Düzeltildi (kod tarafı)** | Bölüm 7.1. Go handler/middleware/repository değişiklikleri bu ortamda derlenemedi; ilk `go build/test` sende. |
| **Konu ↔ bölüm ilgisi (içerik puanı)** | **Var (sezgisel, TF-IDF)** | Dersin `department` alanı ile konu adı/açıklaması arasında TF-IDF + kosinüs (bölüm 2.2). Mevcut performans puanlarını değiştirmez. |
| **Sınıf sayfasından öğrenci analizi ve öğretmen notu** | **Var** | Sınıf → Öğrenciler sekmesinde "Analiz" ve "Not ekle"; "Öğretmen notları" sekmesi. |
| **Sınav iş akışı (backend)** | **Var** | Atomik oluşturma/düzenleme, doğrulama, onay, süre dolumu, cevap anahtarı sızıntısı kapatıldı (bölüm 2.1). |
| **Online sınav ekleme/çözme arayüzü (frontend)** | **Var (tarayıcıda denenmedi)** | Oluşturucu, çözme ekranı, sonuç ve personel özeti (bölüm 2.1). `svelte-check`/build geçti; gerçek tarayıcı testi yapılmadı. |

**Kural:** Bu üç belirsiz konuda kural uydurma. Tanım yoksa kodu yazmadan önce sor, ya da "TODO / doğrulanmadı" diye işaretle.

### 2.1 Online sınav sistemi (uygulandı)

**Akış:** Öğretmen/yönetici sınavı **tek formda** (bilgiler + sorular + seçenekler) kurar → taslak kaydedilir → öğretmen "onaya gönder", yönetici "yayınla/onayla" → kayıtlı öğrenci çözer → sunucu puanlar → sonuç öğrenciye, özet öğretmene gider.

**Varsayımlar (sahibi onaylamadı; değiştirmeden önce sor):**

- **Süre dolunca otomatik teslim:** süresi dolan deneme kayıtlı cevaplarıyla puanlanıp `completed` olur (`completed_at` = bitiş anı). `abandoned` durumu artık hiçbir akışta atanmaz.
- **Sonuç:** puan teslimden hemen sonra görünür; **doğru cevaplar ve soru bazında doğruluk** yalnızca sınav kapandıysa (`close_at` geçtiyse) ya da öğrencinin deneme hakkı bittiyse açılır (`exams.ShouldReveal`).
- **Soru türü yalnızca tek doğru cevaplı çoktan seçmelidir** (2–10 seçenek, tam bir doğru). Karıştırma yok.
- **Çözülmeye başlanmış sınavın soruları değiştirilemez** (403); yeni sınav kopyalanarak yapılır. Düzenleme sınavı taslağa döndürür.
- **Onay/ret yalnızca yöneticidir;** öğretmen yalnızca onaya gönderir. Soruların durumu sınavın durumunu izler.
- Sınav notu otomatik olarak not defterine (`grades`) yazılmaz.

**Backend:**

- `pkg/domain/exams` (saf kurallar + `Store` arayüzü): `Check`/`CheckDraft` (Türkçe `Issue` listesi), `Compose`, `Replace`, `Review`, `Detail`, `Results`; öğrenci görünümü `Detail.ForStudent()` (cevap anahtarı yok), `BuildResult(attempt, detail, reveal)`, `Summarize`. Kalıcılık `internal/database/exam_repository.go` (tek veritabanı işlemi; hata olursa hiçbiri kalmaz).
- Uçlar: `POST /classes/:id/exams`, `GET|PUT /classes/:id/exams/:quizID`, `POST .../review`, `GET .../results` (yalnızca admin/öğretmen); öğrenci için `GET /attempts/:id/exam` (sürerken, 409 = bitti) ve `GET /attempts/:id/result`. Doğrulama hatası **422 + `issues`** döner.
- `attempts.Service`: `Start` sürmekte olan denemeyi **sürdürür** (yenisini açmaz), hazır olmayan sınavı başlatmaz (≥1 soru, her soruda ≥2 seçenek ve tam 1 doğru), `Submit` idempotenttir, süre dolumu **tembel** işlenir (`Start/Answer/Submit/GetByID/List`; Vercel'de arka plan işi ve sık cron yok). Süre = `min(başlangıç+süre, close_at)`; süre 0 iken de `close_at` geçerlidir.
- **Sızıntı kapatıldı:** öğrenciye dönen deneme ve cevap çıktılarında `correct`/`points` yoktur (`attempts.QuizAttempt.View`); öğrenci `questions`/`options` uçlarına (403) ve taslak sınavlara erişemez; `quizes`/`questions` `review` ucunda onay/ret yönetici işidir (öğretmen kendi sınavını onaylayamaz).
- `Quiz.Deadline` ve `attempts` süre hesabındaki "süre 0 iken kapanış yok sayılıyordu" hatası düzeltildi.

**Frontend:** `ExamBuilder` (oluştur/düzenle, anlık eksik listesi), `ExamRunner` (sunucu saatine göre sayaç, her seçimde otomatik kayıt, temizle, teslim onayı, süre bitince otomatik teslim), `ExamResult`. Sayfalar: `/quizzes` (liste), `/quizzes/new?class=`, `/quizzes/:class/:quiz` (öğrenci: başla/devam/sonuç; personel: sorular + sonuçlar), `/quizzes/:class/:quiz/edit`, `/attempts/:id` (sürüyorsa çözme ekranı, bittiyse sonuç). Saf mantık `src/lib/exam.ts` (+ `exam.spec.ts`).

**Kurallar:**

- Cevap anahtarını (`option.answer`, `correct`, `points`) öğrenciye **asla** döndürme; yeni uçlarda `ForStudent`/`View(false)`/`BuildResult(..., reveal)` kullan ve test ekle.
- Süre ve puanı yalnızca sunucuda hesapla; istemci sayacı sadece göstergedir.
- Eski düşük seviye uçlar (`/quizzes/:id/questions`, `.../options`) personel için durur ama arayüz artık kullanmaz; yeni özellikleri `exams` üzerine yaz.
- Vercel: cevap kaydı tek yazmadır (SQLite tek yazar), istemci aynı soru için istekleri sıralı gönderir; sınav gövdesi Fiber `BodyLimit` (1 MB) altında kalmalı (200 soru ×10 seçenek rahatça sığar).

**Sahibine sorulacaklar:** süre bitince otomatik teslim doğru mu? Doğru cevaplar kapanışta/hak bitince açılsın mı? Soru/seçenek karıştırma, çoklu doğru, açık uçlu soru gerekir mi? Sınav notu not defterine otomatik yazılsın mı? Çözülmüş sınavın kopyalanması gerekiyor mu?

### 2.2 Konu–bölüm ilgisi (TF-IDF) ve EFOR / AKTS (uygulandı)

**Varsayımlar (sahibi onaylamadı; değiştirmeden önce sor):**

- **"Bölüm" ayrı bir varlık değil, dersin bir alanıdır:** `Course.Department` (örn. "Bilgisayar Programcılığı") ve `Course.DepartmentKeywords` (virgülle ayrılmış ek terimler). Ayrı bir `Department` tablosu yoktur.
- **EFOR = iş yükü:** `Topic.StudyHours` (konu başına saat) + `Course.HomeworkHours` + `Course.ProjectHours`. **AKTS = toplam saat / 30** (`DefaultHoursPerECTS`; istekte `hours_per_ects` ile 25–30 aralığında değişir; AKTS en yakın 0,5'e yuvarlanır, ham değer `ects_exact`). Yalnızca **aktif** konular sayılır.
- **Konu açıklaması zorunludur:** `Topic.Description` en az 10, en çok 4000 karakter; `StudyHours` 0–500. Eski konuları düzenlerken açıklama istenir.

**Algoritma (`pkg/domain/relevance`, yalnızca standart kütüphane):** Türkçe küçük harf + ASCII katlama, durak sözcükleri, ilk-5-karakter kökleme (F5) → TF-IDF (alt-doğrusal TF × düzgünleştirilmiş IDF, L2 normalize) → bölüm profili ile konu belgesi arasında kosinüs benzerliği. Profil = bölüm adı (×3) + `DepartmentKeywords` (×2) + `lexicon.go` içindeki bölüm sözlüğü (×1; örn. "Bilgisayar Programcılığı" → yazılım, donanım, denetim, algoritma…). Konu belgesi = ad (×2) + açıklama (×1). Puan = `min(100, kosinüs / 0,35 × 100)`; bantlar: ≥60 yüksek, ≥30 orta, >0 düşük, 0 ilgisiz. **Eşikler ve 0,35 doygunluk değeri ayarlanabilir sezgiseldir**, bilimsel bir kalibrasyon değildir.

**Neden Go'da, Python servisinde değil:** ek function çağrısı ve soğuk başlangıç gerektirmez (Vercel Hobby kotası), performans servisi kapalıyken de çalışır ve gerçekten test edilebilir. Python'a taşımak istenirse `relevance.Analyze` sözleşmesi korunmalı.

**Nerede kullanılır:**

- `GET /api/courses/:courseID/content-analysis` → `relevance.Result` (`ready`, `fit`, `department_terms`, konu başına `score`/`band`/`matched_terms`). Bölüm veya anahtar kelime yoksa `ready=false`.
- `GET /api/courses/:courseID/workload[?hours_per_ects=25..30]` → `courses.Workload`.
- Öğrenci performans sonucu: `performance.annotateRelevance` (backend, motor çağrısından sonra) `courses[].department_fit` ve `topics[]/weak_areas[].relevance` alanlarını doldurur; **bölümü olmayan derslerde alan `null` kalır, 0 değildir.** Python servisi bu alanları üretmez, yalnızca sözleşme eşleşsin diye girdi alanlarını kabul eder.
- Frontend: Ders → "Konular" sekmesi (bölüm uyumu kartı, konu başına ilgi ve eşleşen terimler), Ders → "EFOR / AKTS" sekmesi (`WorkloadPanel`), Dersler formunda bölüm ve saat alanları.

**Kurallar:**

- Mevcut performans **puanlarını** (öğrenci/öğretmen genel puanı) bölüm ilgisine göre değiştirme; yalnızca ek alan olarak göster. Puana katmak sahibinin kararıdır.
- Sözlüğe terim eklemek `lexicon.go`'da yapılır; bölüme özgü terimleri sahibi `department_keywords` ile kendisi ekler. Yeni bir sözlük girişinde `All` alanı **köklenmiş** (ilk 5 karakter) sözcükleri bekler (`elektrik` ve `elektronik` aynı kökü `elekt` paylaşır); yanlış eşleşmeyi önlemek için gerekirse `Substr` kullan.
- Bilinen sınırlama: F5 kökleme farklı sözcükleri aynı köke düşürebilir (örn. `bilgisayar`/`bilgilendirme`).
- `PUT .../topics/:id` ve `PUT /courses/:id` tüm alanları ister (açıklama dahil); kısmi güncelleme yoktur.

**Sahibine sorulacaklar:** "Bölüm" ayrı bir varlık olmalı mı (okula bağlı bölüm listesi, sınıfların bölümü)? AKTS saat karşılığı 30 mu 25 mi, yuvarlama nasıl olmalı? Bölüm ilgisi öğrenci/öğretmen performans puanına etki etmeli mi? Ödev ve proje süresi ders düzeyinde mi kalacak, yoksa tek tek ödev/proje kaydı mı tutulacak?

## 3. Değişmez Kurallar

1. **Ücretsiz kal.** Proje Vercel **Hobby** planında çalışır. Vercel'de faturalandırılan ya da ücretli plan gerektiren hiçbir şeyi önerme, ekleme veya kodu ona bağımlı yazma. Örnekler: Redis / Vercel KV / Upstash, Vercel Blob, Vercel Postgres ve ücretli Marketplace depolama, Pro'ya özel özellikler, sık cron. Yeni bir harici bağımlılık gerekiyorsa **önce sahibine sor**.
2. Redis ya da benzeri bir paylaşımlı önbellek/kuyruk **yoktur ve eklenmeyecek**. Önbellek ve rate limit süreç içi (in-memory) kalır.
3. Secret'ları (JWT_SECRET, R2 anahtarları, `.env`) asla loglama, yanıta yazma, commit etme veya dokümana kopyalama. Örnekler `backend/.env.example` içindedir.
4. Sahibinin onayı olmadan veritabanı dosyalarına (`lms.db`, `lms-backup.db`, `backup*.sql`) dokunma, silme veya içeriklerini çıktıya dökme. Gerçek öğrenci verisi içerebilirler.
5. Sorunun kapsamı dışında yeniden düzenleme (refactor) yapma. Küçük, hedefli değişiklik yap; doğrulama komutlarını çalıştır (bölüm 5).

## 4. Vercel Kısıtları (tasarım sınırları)

Dağıtım: `vercel.json` (kökte, `version: 2`) iki function tanımlar: `backend/api/index.go` (`@vercel/go`) ve `performance/api/index.py` (`@vercel/python`). Rota sırası **önemlidir**: `/api/analyze/*` ve `/api/health` Python servisine, geri kalan `/api/*` Go backend'e gider. Sırayı değiştirme. Frontend bu dosyada yoktur; SvelteKit `adapter-auto` ile ayrı derlenir (ayrı Vercel projesi olması muhtemeldir, doğrulanmadı).

Hobby planı için varsayımlar (Vercel dokümanlarına göre; limitler değişebilir, değiştirmeden önce Vercel panelinden/dokümandan doğrula):

- **Dosya sistemi salt-okunurdur.** Yalnızca `/tmp` yazılabilir (en çok ~500 MB) ve kalıcı değildir; soğuk başlangıçta ve instance'lar arasında kaybolur.
- **İstek ve yanıt gövdesi en çok 4,5 MB.** Büyük dosya backend'den geçirilmez. Video bu yüzden R2'ye tarayıcıdan **presigned URL ile doğrudan** yüklenir (`internal/storage/r2.go`, frontend `uploadToUrl`). Bu deseni koru. Fiber `BodyLimit` 1 MB'tır.
- **Function paketi en çok 250 MB** (açılmış). Gereksiz bağımlılık ekleme.
- **Süre sınırı kısadır.** Plan yaparken en kötü durumu varsay: ~10 sn. Hobby süresi Fluid compute ayarına bağlı olarak değişir. Hiçbir istek uzun iş yapmamalı.
- **Arka plan işi yok.** Yanıt döndükten sonra goroutine/iş çalışmaya devam etmeyebilir. Yanıttan sonra iş bırakma; iş istek içinde bitmeli.
- **Cron günde en çok bir kez.** Sık periyodik iş tasarlama; işi istek anında tembel (lazy) yap.
- **Paylaşımlı bellek yok.** Her instance kendi belleğine sahiptir. Sonuçları:
  - `internal/cache` (içerik, performans, oturum önbelleği) **best-effort**tir. Doğruluk veya güvenlik buna dayanmamalı; doğruluk kaynağı veritabanıdır.
  - Oturum iptali (logout) başka instance'larda en çok `SESSION_CACHE_TTL` (varsayılan 30 sn) gecikmeyle görünür. Bu bilinen ve kabul edilen bir ödünleşimdir.
  - Fiber `limiter` instance başına çalışır; rate limit kesin değildir, kötüye kullanımı hafifletir.
- **Soğuk başlangıç:** `backend/api/index.go` uygulamayı `sync.Once` ile bir kez kurar; kurulumda `AutoMigrate` ve `EnsureSuperAdmin` çalışır. Bu adımları hızlı ve idempotent tut.
- **İki function zinciri:** Go backend, performans servisini HTTP ile çağırır (`PERFORMANCE_TIMEOUT` varsayılan yerelde 15 sn, Vercel'de 8 sn). İki ayrı function, iki soğuk başlangıç ve iki kat süre tüketimi demektir. Performans hesabını her istekte çağırma; snapshot'ları ve cache'i kullan.

### ⚠ Çözülmemiş kritik soru: veritabanı kalıcılığı

Backend SQLite kullanır (`DATABASE_PATH=./lms.db`, WAL modu). Vercel'de depodaki `lms.db` **salt-okunurdur** ve `/tmp` kalıcı değildir; yani üretimde yazmalar kalıcı olmaz. Ayrıca `gorm.io/driver/sqlite` CGO gerektirir; `@vercel/go` altında derlenebildiği doğrulanmamıştır. Depoda üretim kalıcılığının nasıl çözüldüğüne dair bir iz yok.

**Ajan olarak:** bunu varsayma ve kendi başına bir veritabanı hizmeti seçme. Üretim veri stratejisi sahibine sorulacak bir konudur. Önerilen her seçenek ücretsiz ve Vercel'de faturalandırılmayan bir hizmet olmalı, ve **önce onay** alınmalıdır.

## 5. Mimari ve komutlar

```
backend/      Go 1.22 + Fiber v2 + GORM (SQLite)      module: lms/backend
  api/index.go            Vercel giriş noktası (adaptor.FiberApp)
  cmd/main.go             Yerel çalıştırma (127.0.0.1:APP_PORT, vars. 3000)
  internal/app            Bağımlılık kablolaması (app.New)
  internal/api/{handlers,middleware,routes}
  internal/database       Repository'ler, sqlite.go (DSN/PRAGMA), performance_input.go
  internal/cache          Bellek içi LRU (içerik + oturum)
  internal/analytics      Performans servisi istemcisi
  internal/storage        R2 presign
  internal/workflows      Çok adımlı işlemler (öğrenci kaydı, okul başlatma)
  pkg/core                Hata türleri, JWT
  pkg/domain/<alan>/      model.go, service.go, validation.go
  pkg/domain/relevance    Konu–bölüm TF-IDF (stdlib; text.go, lexicon.go, analyze.go)
  docs/                   Üretilmiş Swagger (elle düzenleme)
performance/  FastAPI + Pydantic v2 (app/, api/index.py, tests/)
frontend/     SvelteKit 3, Svelte 5 (runes), Tailwind 4, zod
```

Doğrulama komutları (değişiklik yaptığın bileşende çalıştır):

```bash
# Backend (backend/ dizininde)
gofmt -l .            # çıktı boş olmalı
go build ./... && go vet ./... && go test ./...
go run ./cmd          # yerel sunucu

# Performans (performance/ dizininde)
pip install -r requirements-dev.txt   # requirements.txt'i de içerir (Pydantic 2.13.5 pini)
pytest                # tümü geçmeli (şu an 16)
python -m app.main    # yerel performans servisi (127.0.0.1:8000)

# Frontend (frontend/ dizininde)
npm run check         # 0 hata, 0 uyarı
npm run build
```

`npm test` / `test:e2e` Playwright tarayıcı indirir ve ağırdır; istenmedikçe çalıştırma.

## 6. Backend kuralları

- **Katmanlar:** handler → `pkg/domain/<alan>` servisi → `internal/database` repository. İş kuralı servise/validation'a girer, handler ince kalır.
- **JSON:** tüm alanlar `snake_case`. Hassas alanlar gizli kalır (`User.EmailHash/PasswordHash/Email`, `base.Model.DeletedAt`).
- **Yanıt zarfı:** `{"data": ..., "error": ...}`. Hata türleri `pkg/core/errors.go` (`ErrNotFound`, `ErrForbidden`, `ErrValidation`, `ErrUnavailable`, …); performans servisine ulaşılamazsa `ErrUnavailable` → 503.
- **`quizes` paket adı yazım hatası değildir, kasıtlıdır.** Düzeltme.
- **Swagger:** Her handler'a `@Summary/@Description/@Tags/@Param/@Success/@Failure/@Security/@Router` ekle. `docs/` üretilmiş çıktıdır; annotation değiştirince yeniden üret. Üretici komutu depoda yok (rapor geçici bir betikten bahsediyor); bir `Makefile`/betik eklemek açık bir iştir.
- **Alan kuralları (API'nin zorunlu kıldıkları):**
  - `term` (dönem) atama, not, kayıt ve program için **zorunludur**.
  - Sınıf: `name`, `level`, tek harfli `no`, okul ve kampüs zorunlu. `no`/`code` büyük harfe çevrilir. `(school_id, level, no)` benzersizdir (indeks `Migrate` ile onarılır; soft-delete edilmiş bir şube aynı harfin yeniden açılmasını hâlâ engeller, bu indekste `deleted_at` koşulu yoktur).
  - Program `day_of_week`: **0 = Pazar … 6 = Cumartesi** (`0..6`).
  - Konu: `description` zorunlu (10–4000 karakter), `study_hours` 0–500. Ders: `department` ≤150, `department_keywords` ≤2000 karakter, `homework_hours`/`project_hours` 0–1000 (bölüm 2.2).
  - Devamsızlık oranı `AttendedRate()` ile 0–100 yuvarlanmış yüzdedir.
  - Kullanıcı oluşturmada geçerli e-posta ve parola zorunlu.
- **Yinelenen etkinlik:** `activities` yazımı `(student_id, occurred_at)` üzerinde `OnConflict DoNothing` (canlı satır koşuluyla) kullanır.
- **Rate limit:** varsayılan 300/dk; `RATE_LIMIT=0` kapatır. Giriş uç noktası 10/dk, performans `PERFORMANCE_RATE_LIMIT` (vars. 30).
- **UNIQUE indeks etiketini değiştirirsen eski veritabanını da onar.** GORM `AutoMigrate` aynı adlı indeks varsa onu yeniden oluşturmaz; etiket düzelse bile mevcut `lms.db`'de eski (yanlış) indeks kalır. Bu yüzden her bileşik UNIQUE indeksin **tüm** alanlarında aynı ad, `priority:N` ve (kısmi indeksse) aynı `where:` olmalı; mevcut dosyalar için `internal/database/index_repair.go` içindeki `uniqueIndexSpecs` listesine giriş ekle (`Migrate` bunu çalıştırır, tekrar çalıştırılabilir). Geçmişte `idx_snapshot_courses`/`idx_snapshot_topics` yalnızca `snapshot_id` üzerindeydi (bir snapshot'a tek ders → `UNIQUE constraint failed: snapshot_courses.snapshot_id`) ve `idx_classes_school_level_no` `level` içermiyordu (aynı okulda 1-A varken 2-A açılamıyordu); ikisi de onarıldı. Yeni bir UNIQUE etiketi eklerken `sqlite_master`'daki gerçek tanımı kontrol et.
- **SQLite:** PRAGMA'lar DSN içindedir (`sqliteDSN`) çünkü bağlantı havuzundaki her bağlantıya uygulanmalıdır. Bunu `db.Exec(PRAGMA)`'ya geri taşıma. Pool: 4 açık / 4 boşta.

## 7. Yetkilendirme ve kapsam (scope)

Roller: `superadmin`, `admin`, `teacher`, `student`. Okul personeli rolleri: `director`, `registrar`. Kapsam middleware'i `internal/api/middleware/auth.go` içindedir (`ClassAccess`, `ClassManagementAccess`, `TeacherSubjectAccess`, `AttemptStartAccess`, `ListScopeAccess`, `ScopedClassIDs`, …).

**En önemli kural:** Süperadmin olmayan ve izinli sınıfı olmayan kullanıcının kapsamı **boştur ve hiçbir satır görmemelidir**. Repository'ler `ScopeApplied` bayrağını kullanır; kapsam uygulanmışsa ve `ClassIDs` boşsa `WHERE 1=0` gelir. `len(ClassIDs) > 0` ise filtre uygula, değilse uygulama gibi bir mantık **veri sızıntısıdır**. Yeni liste/sorgu eklerken bu davranışı koru ve `internal/database/scope_test.go` benzeri bir test ekle.

Yeni bir kaynak ya da repository eklerken: kapsamı repository seviyesinde uygula (`class_id IN ?`), yalnızca handler'da kontrol etmeye güvenme.

### 7.1 IDOR kuralları (öğrenci, sınıf, ders)

Sınıf kapsamı tek başına yetmez: aynı sınıftaki öğrenciler birbirinin kaydını görmemelidir. **Her yeni liste/okuma ucunda şunları uygula:**

- **Öğrenci yalnızca kendi kaydını görür.** Not, devamsızlık, öğretmen notu ve deneme listelerinde öğrenci filtresi `studentSubject(c, requested)` ile **kendi kimliğine sabitlenir** (`pkg/domain/scopeguard.StudentSubject`); başkasının kimliği istenirse 403. Bu, `?student_id=`, `/classes/:id/grades|notes|attendance` ve `/students/:id/*` yollarının hepsi için geçerlidir.
- **Öğretmen notları:** öğrenci yalnızca `visible_to_student` işaretli notları görür (`sharedNotesOnly`, `MayReadNote`); özel not için `GET /notes/:id` 404 döner.
- **Genel liste uçları** (`/schedules`, `/grades`, `/notes`, `/attendance`, `/attempts`) artık öğrenciye de açıktır (yalnızca GET), çünkü kapsam ve öğrenci filtresi zorunlu uygulanır: `ListScopeAccess` öğrenci için kayıtlı olduğu sınıflara sınırlar (`ScopedClassIDs` öğrenci dalı). `/grades/export`, `/attendance/summary` ve `/classes/:id/enrollments` (sınıf listesi) yalnızca admin/öğretmendir.
- **Ders programı:** öğrenci kayıtlı olduğu sınıfın programını okuyabilir (`/schedules`, `/classes/:id/schedules`, `/schedules/:id`); yazma yalnızca ilgili sınıfın öğretmen/yöneticisindedir ve **yalnızca sınıf sayfasından** yapılır (şube kendi programını düzenler).
- **Okul/kampüs/ders listeleri rol bazlı süzülür** (`internal/database/listscope`, `ListParams.Viewer`): yönetici personeli olduğu okulları, öğretmen ders verdiği, öğrenci kayıtlı olduğu sınıfların okul/kampüs/derslerini görür. İç çağrılar (`Viewer == nil`) sınırsızdır; bunu handler'dan **nil bırakma**.
- **Dersler sınıf düzeyinde otomatiktir:** öğrenci sınıfa (şubeye) kaydedilince sınıfa atanmış tüm derslere erişir; sonradan eklenen dersler de kendiliğinden görünür. Kayıt `course_id = NULL` (sınıf düzeyi) yapılır; arayüz ders seçtirmez.
- Arayüz yalnızca yardımcıdır: `src/lib/access.ts` (URL koruması + menü) rolü sunucuyla tutarlı kılar, ama **asıl koruma sunucudadır**. Yeni sayfa eklerken `access.ts` kuralını ve `nav.ts` girdisini birlikte güncelle (`access.spec.ts` ikisinin uyumunu denetler).
- Yeni bir uç eklerken kendine sor: "Bu ucu başka bir sınıfın/okulun öğrencisi, ya da aynı sınıftan başka bir öğrenci çağırırsa ne döner?" Cevap "başkasının verisi" ise düzelt ve test ekle.


## 8. Performans servisi sözleşmesi

> Bu sistem projenin birinci önceliğidir. Canlıya alma kontrol listesi için bölüm 1.1'e bak.

- Sözleşme sürümü **1**: `GET /api/health`, `POST /api/analyze/student`, `POST /api/analyze/teacher`.
- İki tarafı **birlikte** değiştir: backend `pkg/domain/performance/contract.go` ↔ `performance/app/models.py`. Alan değişikliğinde sürümü artır ve iki tarafın testlerini çalıştır.
- Oranlar 0–100 aralığındadır.
- `/api/analyze/*` uç noktaları `PERFORMANCE_INTERNAL_TOKEN` tanımlıysa `X-Internal-Token` ister; backend istemcisi (`WithInternalToken`) aynı değeri gönderir. Bu iki tarafı birlikte değiştir. `/api/health` her zaman açıktır.
- Backend istemcisinin hata eşlemesi (`unavailableStatus`) bilinçlidir: altyapı kaynaklı yanıtlar 503'e, sözleşme hataları `ErrValidation`'a gider. `client_test.go` bunu kilitler; değiştirirsen testi güncelle.
- Önbellek anahtarları: `performance:student:<uuid>`, `performance:teacher:<uuid>` (en çok 10 snapshot saklanır). İkinci çağrıda `cached=true` dönmelidir; bu bugün elle doğrulanmıştır, otomatik teste dökülmemiştir (açık iş).
- Bağımlılıklar `performance/requirements.txt` ile eşleşmeli (Pydantic **2.13.5**). Çalışma zamanı pinleri ile geliştirme ortamını ayrıştırma.
- Vercel'de `uvicorn` yalnızca yerel çalıştırma içindir; function paketine gereksiz ağırlık ekleme.

## 9. Frontend kuralları

- **Tasarım dili (klasik, kurumsal):** beyaz/zinc yüzeyler, 1px kenarlık, hafif gölge, 8px yarıçap, mavi marka rengi (`brand-*`, `app.css` `@theme`). Ortak sınıflar `src/lib/ui.ts` (`input`, `textarea`, `solid`, `ghost`, `card`, `notice`); sayfalarda renk/kenarlık **sabit yazma**, bu belirteçleri kullan. (Eski sarı vurgu ve kalın kenarlıklar kaldırıldı; `#facc15` kullanma.) `card` ile birlikte `border-<renk>` verme (çakışır); uyarı kutusu için `notice`.
- **Ders programı tablosu:** `ScheduleTable` + `lib/schedule.ts` (40 dk ders + 10 dk teneffüs, satır = ders saati, sütun = gün). Program yalnızca sınıf sayfasında düzenlenir; `/schedule` salt okunurdur.
- **Panel hızlı erişimi:** `QuickLinks` yan menüyle aynı kuraldan (`navFor`) beslenir.

- **Arayüz dili Türkçedir.** Etiketler, hata mesajları ve yorumlar Türkçe. **Dosyaları her zaman UTF-8 (BOM'suz) yaz.** PowerShell `Get-Content`/`Set-Content` varsayılanları dosyayı bozar (daha önce çift kodlama oldu: `Ã–ÄŸrenci`). Düzenleyici araçlarını kullan veya açıkça `UTF8Encoding($false)` ver.
- Svelte 5 **runes** modu zorunludur (`vite.config.ts` içinde). `$lib` → `src/lib`.
- API istemcisi `src/lib/api/client.ts`: taban adres `VITE_API_URL` (vars. `http://localhost:3000`) + `/api`. Token `localStorage`/`sessionStorage` içindedir. 401 gelirse oturum temizlenir.
- Form/CRUD deseni: `CrudPage` + `FormDialog` + `options.ts` içindeki `Field` tanımları (`requiredOnCreate`, `requiredOnUpdate`, `default`, async `options`). Sınıf sayfasında dönem için ortak `TERM_FIELD` (zorunlu, mevcut akademik dönem önceden dolu) kullan.
- Sınıfa özel seçenekler `classCourseOptions(classId)` ve `classStudentOptions(classId)` ile gelir; tüm dersleri sunma (sınıfın sunmadığı derse kayıt 422 verir).
- Toplu paneller (`BulkEnrollPanel`, `BulkAttendancePanel`, `BulkGradesPanel`): öğrenci listesini `student_id` ile tekilleştir (bir sınıf ders başına birden fazla kayıt tutabilir), 0–100 doğrula.
- Yeni sayfa/liste eklerken `DataTable` seçilebilir (`selectable`, `bind:selected`, `rowKey`) deseni mevcuttur.
- Tarayıcıda 4,5 MB'ı aşabilecek bir yükleme varsa Vercel function'ından **geçirme**; R2 presign desenini kullan.

## 10. Çalışma ortamı

- Geliştirici ortamı **Windows**. Dosya yollarını göreli ve ayırıcıdan bağımsız yaz; `C:\Users\ACER\...` gibi mutlak yolları koda veya dokümana gömme.
- Depo bir **git deposu değil** (rapora göre). `.venv`, `node_modules` ve geçici çıktıları sürüm kontrolü varsayma; bunları arşive/çıktıya dahil etme.
- Yerel Uvicorn süreçleri geride kalabilir; iş bitince kapat.

## 11. Açık işler (sahibinin raporundan)

0. **Swagger yeniden üretilmeli:** `workload` ve `content-analysis` uç noktaları ile konu/ders alanları annotation'larda var ama `backend/docs/` bu ortamda yeniden üretilemedi.
1. ⭐ **Performans sistemini canlıda aktif et** (bölüm 1.1 kontrol listesi). En yüksek öncelik.
2. ⭐ **Üretim veri kalıcılığı kararı** (bölüm 4'teki kritik soru): performans snapshot'ları ve tüm yazmalar buna bağlı.
3. **Sınav sistemini gerçek ortamda dene** (bölüm 2.1): `go build/test` (stub dışı), tarayıcıda oluşturucu → onay → çözme → sonuç akışı; Swagger yeniden üretimi.
4. **Arayüzü sadeleştir** (sınav ve diğer ekranlar).
5. İkinci performans çağrısında `cached=true` kontrolünü teste dök.
6. Yetki testleri: `AttemptWriteAccess`, `ClassManagementAccess`, `HasTeacherSubjectAccess` için personel (staff) dalları.
7. Swagger üretimini tekrarlanabilir hale getir (betik/Makefile).
8. **EFOR tanımı**, **tek/çift şube ataması**, **ders/bölüm ↔ içerik puanlama kuralı**: sahibinden netleştir, sonra uygula (EFOR ve içerik puanlama performans sistemiyle iç içe olduğundan önce performans canlıya alınmalı).

## 12. Ajanlar için kısa hatırlatma

- **Performans sistemi birinci önceliktir;** canlıda aktif edilmesine hizmet etmeyen işi öne alma (bölüm 1.1).
- Emin değilsen sor; uydurma. Özellikle bölüm 2'deki üç belirsiz konuda.
- Redis/KV/Blob/Postgres gibi Vercel'de ücretli bir şey önerme (bölüm 3).
- Sınav işinde cevap anahtarını öğrenciye sızdırma; süre ve puanı sunucuda tut; süre dolumunu tembel işle (bölüm 2.1).
- Kapsam/yetki ve boş-kapsam davranışını bozma (bölüm 7).
- Değişiklikten sonra ilgili doğrulama komutlarını çalıştır ve sonucu bildir.
- Dosyalarda Türkçe karakterleri koru (UTF-8).
