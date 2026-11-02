package fixture

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
)

// FakeWiki imita o Wikidata (/w/api.php: wbsearchentities e wbgetentities) e
// o resumo da Wikipedia (/<língua>/api/rest_v1/page/summary/<título>) com
// dois jogos: Sonic 2 (dois jogos com o mesmo nome: o do Mega Drive e o de
// 8 bits, desempata pela plataforma; resumo em português) e Crash Bandicoot
// (só tem página em inglês). Requests conta as chamadas; Agent guarda o
// último User-Agent.
type FakeWiki struct {
	Requests atomic.Int64
	Agent    atomic.Value
}

// NewFakeWiki cria o servidor falso.
func NewFakeWiki() *FakeWiki { return &FakeWiki{} }

type fakeEntity struct {
	label     string
	instance  string
	platforms []string
	date      string
	genres    []string
	links     map[string]string
}

var fakeEntities = map[string]fakeEntity{
	"Q35407":   {label: "Sonic the Hedgehog 2", instance: "Q7889", platforms: []string{"Q209868", "Q751719"}, date: "+1992-10-16T00:00:00Z", genres: []string{"Q828322"}, links: map[string]string{"enwiki": "Sonic the Hedgehog 2 (8-bit video game)"}},
	"Q35627":   {label: "Sonic the Hedgehog 2", instance: "Q7889", platforms: []string{"Q10676"}, date: "+1992-11-21T00:00:00Z", genres: []string{"Q828322"}, links: map[string]string{"ptwiki": "Sonic the Hedgehog 2", "enwiki": "Sonic the Hedgehog 2"}},
	"Q105970":  {label: "Sonic the Hedgehog 2", instance: "Q11424"},
	"Q1061":    {label: "Crash Bandicoot", instance: "Q7889", platforms: []string{"Q10677"}, date: "+1996-09-09T00:00:00Z", genres: []string{"Q828322", "Q4282636"}, links: map[string]string{"enwiki": "Crash Bandicoot (video game)"}},
	"Q828322":  {label: "jogo eletrônico de plataforma"},
	"Q4282636": {label: "jogo eletrônico de tiro"},
}

var fakeSummaries = map[string]string{
	"pt/Sonic_the_Hedgehog_2":                    "Sonic the Hedgehog 2 é um jogo de plataforma de 1992 da Sega para o Mega Drive.",
	"en/Sonic_the_Hedgehog_2":                    "Sonic the Hedgehog 2 is a 1992 platform game developed by Sega.",
	"en/Crash_Bandicoot_(video_game)":            "Crash Bandicoot is a 1996 platform game developed by Naughty Dog.",
	"en/Sonic_the_Hedgehog_2_(8-bit_video_game)": "The 8-bit version of Sonic 2.",
}

func (f *FakeWiki) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.Requests.Add(1)
	f.Agent.Store(r.UserAgent())
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/w/api.php" {
		f.api(w, r)
		return
	}
	// /<língua>/api/rest_v1/page/summary/<título>
	lang, rest, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/api/rest_v1/page/summary/")
	text, ok := fakeSummaries[lang+"/"+rest]
	if !ok {
		http.NotFound(w, r)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"type": "standard", "title": rest, "extract": text,
		"content_urls": map[string]any{"desktop": map[string]string{"page": "https://" + lang + ".wikipedia.org/wiki/" + rest}},
	})
}

func item(id string) map[string]any {
	return map[string]any{"mainsnak": map[string]any{"datavalue": map[string]any{"value": map[string]string{"id": id}}}}
}

func (f *FakeWiki) api(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	switch q.Get("action") {
	case "wbsearchentities":
		want := strings.ToLower(q.Get("search"))
		var hits []map[string]string
		// o de 8 bits vem primeiro: quem decide é a plataforma
		for _, id := range []string{"Q35407", "Q105970", "Q35627", "Q1061"} {
			if e := fakeEntities[id]; strings.Contains(strings.ToLower(e.label), want) {
				hits = append(hits, map[string]string{"id": id, "label": e.label})
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"search": hits})
	case "wbgetentities":
		out := map[string]any{}
		for _, id := range strings.Split(q.Get("ids"), "|") {
			e, ok := fakeEntities[id]
			if !ok {
				continue
			}
			claims := map[string]any{}
			if e.instance != "" {
				claims["P31"] = []any{item(e.instance)}
			}
			var plats, genres []any
			for _, p := range e.platforms {
				plats = append(plats, item(p))
			}
			for _, g := range e.genres {
				genres = append(genres, item(g))
			}
			claims["P400"], claims["P136"] = plats, genres
			if e.date != "" {
				claims["P577"] = []any{map[string]any{"mainsnak": map[string]any{"datavalue": map[string]any{"value": map[string]any{"time": e.date, "precision": 11}}}}}
			}
			links := map[string]any{}
			for site, title := range e.links {
				links[site] = map[string]string{"site": site, "title": title}
			}
			out[id] = map[string]any{"id": id, "claims": claims, "sitelinks": links, "labels": map[string]any{"pt": map[string]string{"value": e.label}, "en": map[string]string{"value": e.label}}}
		}
		json.NewEncoder(w).Encode(map[string]any{"entities": out, "success": 1})
	default:
		http.Error(w, "ação desconhecida", http.StatusBadRequest)
	}
}
