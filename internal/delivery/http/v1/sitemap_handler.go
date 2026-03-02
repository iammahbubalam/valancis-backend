package v1

import (
	"encoding/xml"
	"net/http"
	"valancis-backend/internal/usecase"
)

type SitemapHandler struct {
	usecase *usecase.SitemapUsecase
}

func NewSitemapHandler(uc *usecase.SitemapUsecase) *SitemapHandler {
	return &SitemapHandler{usecase: uc}
}

type URLSet struct {
	XMLName xml.Name  `xml:"urlset"`
	Xmlns   string    `xml:"xmlns,attr"`
	URLs    []URLItem `xml:"url"`
}

type URLItem struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float32 `xml:"priority,omitempty"`
}

func (h *SitemapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	items, err := h.usecase.GenerateSitemap(r.Context())
	if err != nil {
		http.Error(w, "Failed to generate sitemap: "+err.Error(), http.StatusInternalServerError)
		return
	}

	urlSet := URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]URLItem, len(items)),
	}

	for i, item := range items {
		urlSet.URLs[i] = URLItem{
			Loc:        item.Loc,
			LastMod:    item.LastMod,
			ChangeFreq: item.ChangeFreq,
			Priority:   item.Priority,
		}
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(urlSet); err != nil {
		http.Error(w, "Failed to encode sitemap", http.StatusInternalServerError)
	}
}
