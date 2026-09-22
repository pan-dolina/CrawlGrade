// Package testsite serves a small, deterministic website with one endpoint
// per condition CrawlGrade detects. Functional tests crawl it through the
// compiled binary; developers can run it with internal/tools/testsite.
//
// Absolute URLs in pages are built from the request's Host header, so the
// site works on any address and port. Nothing is random or time-dependent.
package testsite

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Page is one endpoint.
type Page struct {
	Status   int
	Location string
	Header   map[string]string
	Body     string
	Gzip     bool
}

// Handler returns the site.
func Handler() http.Handler {
	pages := Pages()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		p, ok := pages[key]
		if !ok {
			p, ok = pages[r.URL.Path]
		}
		if !ok && strings.HasPrefix(r.URL.Path, "/calendar") {
			p, ok = calendar(r), true
		}
		if !ok {
			p = Page{Status: http.StatusNotFound, Body: notFound}
		}
		base := "http://" + r.Host
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		for k, v := range p.Header {
			h.Set(k, strings.ReplaceAll(v, "{{base}}", base))
		}
		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", "text/html; charset=utf-8")
		}
		if p.Location != "" {
			h.Set("Location", strings.ReplaceAll(p.Location, "{{base}}", base))
		}
		body := []byte(strings.ReplaceAll(p.Body, "{{base}}", base))
		if p.Gzip {
			var buf bytes.Buffer
			zw := gzip.NewWriter(&buf)
			_, _ = zw.Write(body)
			_ = zw.Close()
			body = buf.Bytes()
			h.Set("Content-Encoding", "gzip")
		}
		status := p.Status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if r.Method == http.MethodGet {
			_, _ = w.Write(body) // #nosec G705 -- deliberately hostile local test fixtures, not a public web service.
		}
	})
}

// Paths returns every fixed endpoint, sorted.
func Paths() []string {
	var out []string
	for p := range Pages() {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

const notFound = `<!doctype html><html lang="pl"><head><title>Nie znaleziono</title></head><body><h1>404</h1></body></html>`

const nav = `<header class="site-header">
<nav class="main-nav"><ul>
<li><a href="/">Strona główna</a></li>
<li><a href="/good">Oferta salonu</a></li>
<li><a href="/keyword-page">Badanie wzroku</a></li>
<li><a href="/hreflang">Wersje językowe</a></li>
<li><a href="/breadcrumb-jsonld">Kontakt z salonem</a></li>
</ul></nav>
</header>`

const footer = `<footer class="site-footer">
<p>Salon Optyczny Przykład sp. z o.o., ul. Testowa 1, 00-001 Warszawa. Wszelkie prawa zastrzeżone.</p>
<p><a href="/good">Polityka prywatności</a> <a href="/breadcrumb-jsonld">Regulamin sklepu</a></p>
</footer>`

type doc struct {
	lang, title, head, main string
	noTitle                 bool
}

func (d doc) render() string {
	lang := d.lang
	if lang == "" {
		lang = "pl"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n", lang)
	if !d.noTitle {
		fmt.Fprintf(&b, "<title>%s</title>\n", d.title)
	}
	b.WriteString(d.head)
	b.WriteString("\n</head>\n<body>\n")
	b.WriteString(nav)
	b.WriteString("\n<main>\n")
	b.WriteString(d.main)
	b.WriteString("\n</main>\n")
	b.WriteString(footer)
	b.WriteString("\n</body>\n</html>\n")
	return b.String()
}

func html(d doc) Page { return Page{Body: d.render()} }

func canonical(path string) string {
	return `<link rel="canonical" href="{{base}}` + path + `">`
}

func description(s string) string {
	return `<meta name="description" content="` + s + `">`
}

const goodDescription = "Salon optyczny w Warszawie: badanie wzroku u optometrysty, okulary korekcyjne i progresywne, szeroki wybór oprawek."

const social = `<meta property="og:title" content="Salon Optyczny Przykład">
<meta property="og:description" content="Badanie wzroku i okulary korekcyjne w Warszawie.">
<meta property="og:image" content="{{base}}/img/og.jpg">
<meta property="og:url" content="{{base}}/good">
<meta property="og:type" content="website">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="Salon Optyczny Przykład">
<meta name="twitter:description" content="Badanie wzroku i okulary korekcyjne.">
<meta name="twitter:image" content="{{base}}/img/og.jpg">`

// sharedArticle is the main content of the duplicate-content fixtures.
const sharedArticle = `<h1>Jak dobrać oprawki do kształtu twarzy</h1>
<p>Dobór oprawek zaczyna się od oceny kształtu twarzy. Osoby o twarzy okrągłej zwykle wybierają oprawki prostokątne, które optycznie wydłużają rysy. Przy twarzy kwadratowej sprawdzają się oprawki owalne lub okrągłe, łagodzące mocną linię żuchwy.</p>
<p>Twarz pociągła dobrze wygląda w szerokich oprawkach z wyraźnym górnym brzegiem. Twarz w kształcie serca równoważą oprawki cięższe w dolnej części. Warto pamiętać, że szerokość oprawek powinna odpowiadać szerokości twarzy w najszerszym miejscu.</p>
<p>Znaczenie ma także kolor. Ciepłe odcienie oprawek pasują do cery o złotym podtonie, a chłodne do cery różowej. Optyk pomoże dopasować mostek i zauszniki tak, aby okulary nie uciskały nosa ani skroni podczas całodziennego noszenia.</p>
<p>Przed zakupem nowych oprawek dobrze jest wykonać aktualne badanie wzroku. Moc soczewek wpływa na grubość szkieł, a to z kolei na wybór odpowiedniej oprawy. Grube soczewki lepiej maskują mniejsze oprawki o pełnym obramowaniu.</p>`

// Pages returns the fixed endpoints.
func Pages() map[string]Page {
	p := map[string]Page{}

	p["/"] = html(doc{
		title: "Salon Optyczny Przykład – badanie wzroku i okulary w Warszawie",
		head: description("Salon optyczny w centrum Warszawy. Badanie wzroku, okulary korekcyjne, soczewki kontaktowe i naprawa okularów.") +
			canonical("/") + "\n" + social + `
<link rel="alternate" hreflang="pl" href="{{base}}/">
<script type="application/ld+json">
{"@context":"https://schema.org","@graph":[
 {"@type":"Organization","name":"Salon Optyczny Przykład","url":"{{base}}/","logo":"{{base}}/img/logo.png"},
 {"@type":"WebSite","name":"Salon Optyczny Przykład","url":"{{base}}/"}
]}
</script>`,
		main: `<h1>Salon optyczny Przykład</h1>
<p>Zapraszamy do salonu optycznego w centrum Warszawy. Wykonujemy badanie wzroku, dobieramy okulary korekcyjne i soczewki kontaktowe.</p>
<img src="/img/salon.jpg" alt="Wnętrze salonu optycznego" width="800" height="600">
<h2>Strony testowe</h2>
<ul class="fixtures">
<li><a href="/good">Dobra strona</a></li>
<li><a href="/missing-title">Brak tytułu</a></li>
<li><a href="/duplicate-title">Zduplikowany tytuł</a></li>
<li><a href="/noindex">Strona noindex</a></li>
<li><a href="/redirect">Przekierowanie</a></li>
<li><a href="/redirect-chain">Łańcuch przekierowań</a></li>
<li><a href="/redirect-loop">Pętla przekierowań</a></li>
<li><a href="/broken">Nieistniejąca strona</a></li>
<li><a href="/canonical-good">Poprawny canonical</a></li>
<li><a href="/canonical-loop">Pętla canonical</a></li>
<li><a href="/canonical-mismatch">Canonical do innej strony</a></li>
<li><a href="/hreflang">Hreflang</a></li>
	<li><a href="/social">Dane społecznościowe</a></li>
	<li><a href="/social-bad">Złe społecznościowe</a></li>
	<li><a href="/link-a">Link A</a></li>
	<li><a href="/link-b">Link B</a></li>
<li><a href="/breadcrumb-jsonld">Okruszki JSON-LD</a></li>
<li><a href="/breadcrumb-invalid">Błędne okruszki</a></li>
<li><a href="/duplicate-content-a">Artykuł A</a></li>
<li><a href="/duplicate-content-b">Artykuł B</a></li>
<li><a href="/near-duplicate">Artykuł C</a></li>
<li><a href="/keyword-page">Badanie wzroku i okulary</a></li>
<li><a href="/headings">Nagłówki</a></li>
<li><a href="/images">Obrazy</a></li>
<li><a href="/xss">Wrogie dane</a></li>
<li><a href="/gzip">Kompresja</a></li>
<li><a href="/ssrf-redirect">Przekierowanie do metadanych</a></li>
<li><a href="/private/secret">Obszar prywatny</a></li>
<li><a href="/docs/cennik.pdf">Cennik PDF</a></li>
<li><a href="https://external.example/partner" rel="sponsored nofollow">Partner</a></li>
<li><a href="/good" rel="nofollow"></a></li>
</ul>`,
	})

	p["/good"] = html(doc{
		title: "Oferta salonu optycznego – okulary korekcyjne i oprawki",
		head: description(goodDescription) + canonical("/good") + "\n" + social + `
<meta name="robots" content="index, follow, max-image-preview:large">
<script type="application/ld+json">{"@context":"https://schema.org","@type":"WebPage","name":"Oferta salonu","url":"{{base}}/good"}</script>`,
		main: `<h1>Oferta salonu optycznego</h1>
<p>W naszej ofercie znajdziesz okulary korekcyjne, okulary przeciwsłoneczne z filtrem UV oraz nowoczesne oprawki renomowanych marek.</p>
<h2>Okulary korekcyjne</h2>
<p>Każde okulary korekcyjne wykonujemy na podstawie aktualnego badania wzroku.</p>
<h2>Oprawki</h2>
<p>Oprawki metalowe, acetatowe i tytanowe dobieramy do kształtu twarzy.</p>
<img src="/img/oprawki.jpg" alt="Oprawki na półce" width="640" height="480" loading="lazy">`,
	})

	p["/missing-title"] = html(doc{
		noTitle: true,
		head:    description("Strona bez znacznika title.") + canonical("/missing-title"),
		main:    `<h1>Strona bez tytułu</h1><p>Ta strona celowo nie ma elementu title w nagłówku dokumentu.</p>`,
	})

	p["/duplicate-title"] = html(doc{
		title: "Oferta salonu optycznego – okulary korekcyjne i oprawki",
		head:  description(goodDescription) + canonical("/duplicate-title"),
		main:  `<h1>Druga oferta</h1><p>Ta strona ma ten sam tytuł i opis co strona z ofertą salonu, ale inną treść o naprawie okularów.</p>`,
	})

	p["/noindex"] = html(doc{
		title: "Strona wyłączona z indeksowania",
		head:  description("Strona z dyrektywą noindex.") + canonical("/noindex") + `<meta name="robots" content="noindex, follow">`,
		main:  `<h1>Wyłączona z indeksu</h1><p>Ta strona prosi roboty wyszukiwarek, aby jej nie indeksowały.</p>`,
	})

	p["/redirect"] = Page{Status: http.StatusMovedPermanently, Location: "/good"}
	p["/redirect-chain"] = Page{Status: http.StatusMovedPermanently, Location: "/redirect-chain-2"}
	p["/redirect-chain-2"] = Page{Status: http.StatusFound, Location: "/redirect-chain-3"}
	p["/redirect-chain-3"] = Page{Status: http.StatusMovedPermanently, Location: "{{base}}/good"}
	p["/redirect-loop"] = Page{Status: http.StatusFound, Location: "/redirect-loop-b"}
	p["/redirect-loop-b"] = Page{Status: http.StatusFound, Location: "/redirect-loop"}
	p["/ssrf-redirect"] = Page{Status: http.StatusFound, Location: "http://169.254.169.254/latest/meta-data/"}
	p["/broken"] = Page{Status: http.StatusNotFound, Body: notFound}

	p["/canonical-good"] = html(doc{
		title: "Poprawny adres kanoniczny",
		head:  description("Strona z poprawnym, samoodwołującym się adresem kanonicznym.") + canonical("/canonical-good"),
		main:  `<h1>Poprawny canonical</h1><p>Adres kanoniczny tej strony wskazuje na nią samą.</p>`,
	})
	p["/canonical-loop"] = html(doc{
		title: "Pętla canonical A",
		head:  description("Canonical wskazuje na stronę B.") + canonical("/canonical-loop-b"),
		main:  `<h1>Pętla canonical A</h1><p>Zobacz <a href="/canonical-loop-b">stronę B</a>.</p>`,
	})
	p["/canonical-loop-b"] = html(doc{
		title: "Pętla canonical B",
		head:  description("Canonical wskazuje na stronę A.") + canonical("/canonical-loop"),
		main:  `<h1>Pętla canonical B</h1><p>Zobacz <a href="/canonical-loop">stronę A</a>.</p>`,
	})
	p["/canonical-mismatch"] = html(doc{
		title: "Canonical do innej strony",
		head: description("Strona wskazuje inną stronę jako kanoniczny.") +
			`<link rel="canonical" href="/good"><link rel="canonical" href="{{base}}/canonical-good">`,
		main: `<h1>Canonical do innej strony</h1><p>Ta strona ma dwa elementy canonical, w tym jeden względny.</p>`,
	})

	// A page with consistent, valid social preview metadata.
	p["/social"] = html(doc{
		title: "Dane preview społecznościowego",
		head: description("Strona z danymi preview dla sieci społecznościowych.") +
			canonical("/social") + "\n" +
			`<meta property="og:title" content="Salon Optyczny Przykład">` +
			`<meta property="og:description" content="Badanie wzroku i okulary korekcyjne.">` +
			`<meta property="og:image" content="{{base}}/img/og.jpg">` +
			`<meta name="twitter:card" content="summary_large_image">` +
			`<meta name="twitter:title" content="Salon Optyczny Przykład">` +
			`<meta name="twitter:image" content="{{base}}/img/og.jpg">`,
		main: `<h1>Dane preview</h1><p>Strona ma zgodne dane Open Graph i Twitter Card.</p>`,
	})

	// A page with inconsistent and invalid social metadata.
	p["/social-bad"] = html(doc{
		title: "Złe dane preview społecznościowego",
		head: description("Strona z błędnymi danymi preview.") +
			canonical("/social-bad") + "\n" +
			`<meta property="og:title" content="Tytuł Open Graph">` +
			`<meta property="og:image" content="not-a-url">` +
			`<meta name="twitter:card" content="summary_large_image">` +
			`<meta name="twitter:title" content="Tytuł Twitter">`,
		main: `<h1>Złe dane preview</h1><p>Strona ma niezgodne i błędne dane preview.</p>`,
	})

	// Two pages that link to the same target with the same anchor text, so
	// the link graph reports a repeated anchor.
	p["/link-a"] = html(doc{
		title: "Strona linkowa A",
		head:  description("Strona linkowa A.") + canonical("/link-a"),
		main:  `<h1>Link A</h1><p><a href="/link-target">see this page</a></p>`,
	})
	p["/link-b"] = html(doc{
		title: "Strona linkowa B",
		head:  description("Strona linkowa B.") + canonical("/link-b"),
		main:  `<h1>Link B</h1><p><a href="/link-target">see this page</a></p>`,
	})
	p["/link-target"] = html(doc{
		title: "Strona docelowa linków",
		head:  description("Strona docelowa linków.") + canonical("/link-target"),
		main:  `<h1>Link target</h1><p>Ta strona jest linkowana z dwóch stron tym samym tekstem.</p>`,
	})

	hreflangHead := `<link rel="alternate" hreflang="pl" href="{{base}}/hreflang">
<link rel="alternate" hreflang="en" href="{{base}}/hreflang-en">
<link rel="alternate" hreflang="de" href="{{base}}/hreflang-de">
<link rel="alternate" hreflang="en-UK" href="{{base}}/hreflang-en">
<link rel="alternate" hreflang="x-default" href="{{base}}/hreflang">
<link rel="alternate" hreflang="en" href="{{base}}/good">`
	p["/hreflang"] = html(doc{
		title: "Wersje językowe salonu",
		head:  description("Strona z wersjami językowymi.") + canonical("/hreflang") + hreflangHead,
		main:  `<h1>Wersje językowe</h1><p>Wybierz <a href="/hreflang-en">English</a> albo <a href="/hreflang-de">Deutsch</a>.</p>`,
	})
	p["/hreflang-en"] = html(doc{
		lang:  "en",
		title: "Optician in Warsaw – eye examination and prescription glasses",
		head: description("Eye examination and prescription glasses in Warsaw.") + canonical("/hreflang-en") + `
<link rel="alternate" hreflang="en" href="{{base}}/hreflang-en">
<link rel="alternate" hreflang="pl" href="{{base}}/hreflang">`,
		main: `<h1>Eye examination in Warsaw</h1>
<p>Our optometrist offers a comprehensive eye examination for adults and children. After the eye examination we help you choose prescription glasses and frames that fit your face.</p>
<p>The eye examination takes about thirty minutes and includes a check of visual acuity and eye pressure.</p>`,
	})
	p["/hreflang-de"] = html(doc{
		lang:  "pl-PL",
		title: "Augenoptiker in Warschau",
		head:  description("Sehtest und Brillen in Warschau.") + canonical("/hreflang-de") + `<link rel="alternate" hreflang="de" href="{{base}}/hreflang-de">`,
		main:  `<h1>Sehtest in Warschau</h1><p>Diese Seite verweist nicht zurück auf die polnische Version.</p>`,
	})

	p["/breadcrumb-jsonld"] = html(doc{
		title: "Kontakt z salonem optycznym",
		head: description("Dane kontaktowe salonu optycznego.") + canonical("/breadcrumb-jsonld") + `
<script type="application/ld+json">
{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
 {"@type":"ListItem","position":1,"name":"Strona główna","item":"{{base}}/"},
 {"@type":"ListItem","position":2,"name":"Salon","item":"{{base}}/good"},
 {"@type":"ListItem","position":3,"name":"Kontakt"}
]}
</script>
<script type="application/ld+json">{"@context":"https://schema.org","@type":"LocalBusiness","name":"Salon Optyczny Przykład","address":{"@type":"PostalAddress","streetAddress":"ul. Testowa 1","addressLocality":"Warszawa"},"telephone":"+48 22 000 00 00"}</script>`,
		main: `<ol class="breadcrumbs" itemscope itemtype="https://schema.org/BreadcrumbList">
<li itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem"><a itemprop="item" href="/"><span itemprop="name">Strona główna</span></a><meta itemprop="position" content="1"></li>
<li itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem"><a itemprop="item" href="/good"><span itemprop="name">Salon</span></a><meta itemprop="position" content="2"></li>
</ol>
<h1>Kontakt</h1><p>Salon jest otwarty od poniedziałku do soboty. Zapraszamy na badanie wzroku bez wcześniejszej rejestracji.</p>`,
	})

	p["/breadcrumb-invalid"] = html(doc{
		title: "Błędne dane okruszków",
		head: description("Strona z błędnymi danymi strukturalnymi.") + canonical("/breadcrumb-invalid") + `
<script type="application/ld+json">
{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[
 {"@type":"ListItem","position":1,"item":"{{base}}/"},
 {"@type":"ListItem","position":1,"name":"Duplikat","item":"{{base}}/"},
 {"@type":"ListItem","position":"trzy","name":"Zła pozycja","item":"not a url"},
 {"@type":"Thing","name":"Nie ListItem"}
]}
</script>
<script type="application/ld+json">{"@context":"https://schema.org","@type":"Product", "name": "Okulary" </script>
<script type="application/ld+json">{"@context":"https://schema.org","@type":"FAQPage","mainEntity":[{"@type":"Question","name":"Ile trwa badanie?"}]}</script>`,
		main: `<div itemscope itemtype="https://schema.org/BreadcrumbList"><span itemprop="itemListElement">Brak ListItem</span></div>
<h1>Błędne okruszki</h1><p>Dane strukturalne na tej stronie zawierają celowe błędy.</p>`,
	})

	p["/duplicate-content-a"] = html(doc{
		title: "Jak dobrać oprawki – poradnik A",
		head:  description("Poradnik doboru oprawek, wersja A.") + canonical("/duplicate-content-a"),
		main:  sharedArticle,
	})
	p["/duplicate-content-b"] = html(doc{
		title: "Jak dobrać oprawki – poradnik B",
		head:  description("Poradnik doboru oprawek, wersja B.") + canonical("/duplicate-content-b"),
		main:  sharedArticle,
	})
	p["/near-duplicate"] = html(doc{
		title: "Jak dobrać oprawki – poradnik C",
		head:  description("Poradnik doboru oprawek, wersja C.") + canonical("/near-duplicate"),
		main:  strings.Replace(sharedArticle, "Warto pamiętać, że szerokość", "Pamiętaj również, że szerokość", 1),
	})

	p["/keyword-page"] = html(doc{
		title: "Badanie wzroku i okulary korekcyjne – optometrysta w Warszawie",
		head: description("Badanie wzroku u optometrysty, okulary korekcyjne, okulary progresywne i oprawki.") + canonical("/keyword-page") + `
<script type="application/ld+json">{"@context":"https://schema.org","@type":"Article","headline":"Badanie wzroku u optometrysty","author":{"@type":"Person","name":"Anna Nowak"}}</script>`,
		main: `<h1>Badanie wzroku u optometrysty</h1>
<p>Badanie wzroku to pierwszy krok do dobrze dobranych okularów. Optometrysta sprawdza ostrość widzenia, widzenie obuoczne i ciśnienie w oku. Regularne badanie wzroku pozwala wcześnie wykryć wady refrakcji.</p>
<h2>Okulary korekcyjne</h2>
<p>Na podstawie wyniku badania wzroku optometrysta dobiera okulary korekcyjne. Okulary korekcyjne mogą mieć soczewki jednoogniskowe lub okulary progresywne dla osób z presbiopią.</p>
<h2>Okulary progresywne</h2>
<p>Okulary progresywne łączą korekcję do dali i do bliży. Dobre okulary progresywne wymagają precyzyjnego pomiaru, dlatego optometrysta mierzy rozstaw źrenic i wysokość montażu.</p>
<h2>Oprawki</h2>
<p>Oprawki dobieramy do kształtu twarzy i rodzaju soczewek. Lekkie oprawki tytanowe sprawdzają się przy okularach progresywnych. W salonie znajdziesz ponad tysiąc modeli oprawek.</p>
<p>Umów badanie wzroku u optometrysty i wybierz oprawki razem z nami.</p>`,
	})

	p["/headings"] = html(doc{
		title: "Struktura nagłówków",
		head:  description("Strona z błędną hierarchią nagłówków.") + canonical("/headings"),
		main:  `<h1>Pierwszy nagłówek</h1><h3>Przeskok poziomu</h3><h2>  </h2><h1>Drugi nagłówek H1</h1><p>Treść strony z nagłówkami.</p>`,
	})

	p["/images"] = html(doc{
		title: "Galeria oprawek",
		head:  description("Galeria zdjęć oprawek.") + canonical("/images"),
		main: `<h1>Galeria</h1>
<img src="/img/a.jpg" width="100" height="100">
<img src="/img/decor.png" alt="" width="10" height="10">
<img src="/img/b.jpg" alt="Oprawki tytanowe">
<img src="/missing.png" alt="Nieistniejący obraz" width="50" height="50" loading="lazy">
<img src="data:image/gif;base64,R0lGODlhAQABAAAAACw=" alt="piksel" width="1" height="1">
<p>Zdjęcia oprawek z naszego salonu.</p>`,
	})

	p["/xss"] = html(doc{
		title: `&lt;/title&gt;&lt;script&gt;alert("title")&lt;/script&gt; Wrogie dane`,
		head: `<meta name="description" content="&quot;&gt;&lt;img src=x onerror=alert('desc')&gt;">
<link rel="canonical" href="javascript:alert(document.cookie)">
<meta property="og:image" content="javascript:alert('og')">
<script type="application/ld+json">{"@context":"https://schema.org","@type":"Organization","name":"<\/script><script>alert('jsonld')<\/script>","url":"javascript:alert(1)"}</script>`,
		main: `<h1>&lt;svg onload=alert('h1')&gt;</h1>
<p>&lt;img src=x onerror=alert('term')&gt; onerror onerror onerror payload payload payload</p>
<a href="javascript:alert('link')">&lt;b onmouseover=alert('anchor')&gt;kliknij&lt;/b&gt;</a>
<a href="/xss?q=%22%3E%3Cscript%3Ealert(1)%3C/script%3E">"&gt;&lt;script&gt;alert('href')&lt;/script&gt;</a>
<img src="/img/x.png" alt="&quot; onerror=&quot;alert('alt')">`,
	})

	p["/gzip"] = Page{Gzip: true, Body: doc{
		title: "Strona kompresowana gzip",
		head:  description("Strona wysyłana z kompresją gzip.") + canonical("/gzip"),
		main:  `<h1>Kompresja</h1><p>Ta odpowiedź jest kompresowana algorytmem gzip.</p>`,
	}.render()}

	p["/orphan"] = html(doc{
		title: "Strona osierocona",
		head:  description("Strona obecna tylko w mapie witryny.") + canonical("/orphan"),
		main:  `<h1>Strona osierocona</h1><p>Do tej strony nie prowadzi żaden link wewnętrzny.</p>`,
	})

	p["/private/secret"] = html(doc{title: "Prywatne", main: `<h1>Prywatne</h1>`})

	p["/docs/cennik.pdf"] = Page{Header: map[string]string{"Content-Type": "application/pdf"}, Body: "%PDF-1.4 test"}
	for _, img := range []string{"/img/salon.jpg", "/img/oprawki.jpg", "/img/og.jpg", "/img/logo.png", "/img/a.jpg", "/img/decor.png", "/img/b.jpg", "/img/x.png"} {
		p[img] = Page{Header: map[string]string{"Content-Type": "image/jpeg"}, Body: "img"}
	}

	p["/robots.txt"] = Page{
		Header: map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		Body: `# CrawlGrade test site
User-agent: *
Disallow: /private/

Sitemap: {{base}}/sitemap.xml
`,
	}

	p["/sitemap.xml"] = Page{
		Header: map[string]string{"Content-Type": "application/xml"},
		Body: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url><loc>{{base}}/</loc><lastmod>2026-09-01</lastmod></url>
<url><loc>{{base}}/good</loc></url>
<url><loc>{{base}}/keyword-page</loc></url>
<url><loc>{{base}}/noindex</loc></url>
<url><loc>{{base}}/redirect</loc></url>
<url><loc>{{base}}/broken</loc></url>
<url><loc>{{base}}/canonical-mismatch</loc></url>
<url><loc>{{base}}/orphan</loc></url>
<url><loc>{{base}}/duplicate-content-a</loc></url>
</urlset>
`,
	}
	return p
}

// calendar is an infinite calendar trap: every month links to the next.
func calendar(r *http.Request) Page {
	month := r.URL.Query().Get("month")
	var y, m int
	if _, err := fmt.Sscanf(month, "%d-%d", &y, &m); err != nil || m < 1 || m > 12 {
		y, m = 2026, 1
	}
	ny, nm := y, m+1
	if nm > 12 {
		ny, nm = y+1, 1
	}
	return html(doc{
		title: fmt.Sprintf("Kalendarz wizyt %04d-%02d", y, m),
		head:  description("Kalendarz wolnych terminów badania wzroku."),
		main:  fmt.Sprintf(`<h1>Terminy %04d-%02d</h1><p><a href="/calendar?month=%04d-%02d">Następny miesiąc</a></p>`, y, m, ny, nm),
	})
}
