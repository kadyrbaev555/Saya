package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// The gateway forwards a confirmed search to HeadHunter; ranking stays with HH.
func vacancySearch(client *http.Client, token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Text string `json:"text"`
			Area string `json:"area"`
		}
		if c.ShouldBindJSON(&input) != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		text := strings.TrimSpace(input.Text)
		if text == "" || utf8.RuneCountInString(text) > 160 || input.Area != "160" {
			c.Status(http.StatusBadRequest)
			return
		}
		query := url.Values{"area": {input.Area}, "text": {text}, "per_page": {"10"}, "host": {"hh.kz"}}
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, "https://api.hh.ru/vacancies?"+query.Encode(), nil)
		if err != nil {
			c.Status(http.StatusBadGateway)
			return
		}
		req.Header.Set("User-Agent", "SAYA/1.3 (+https://github.com/kadyrbaev555/Saya)")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		var payload struct {
			Items *[]struct {
				Name     string `json:"name"`
				Employer struct {
					Name string `json:"name"`
				} `json:"employer"`
				URL string `json:"alternate_url"`
			} `json:"items"`
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
		if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &payload) != nil || payload.Items == nil {
			c.Status(http.StatusBadGateway)
			return
		}
		items := []gin.H{}
		for _, v := range *payload.Items {
			u, err := url.Parse(v.URL)
			if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || (u.Hostname() != "hh.kz" && u.Hostname() != "hh.ru" && !strings.HasSuffix(u.Hostname(), ".hh.kz") && !strings.HasSuffix(u.Hostname(), ".hh.ru")) || !strings.HasPrefix(u.Path, "/vacancy/") || v.Name == "" {
				continue
			}
			items = append(items, gin.H{"title": v.Name, "employer": v.Employer.Name, "url": v.URL})
		}
		if len(*payload.Items) > 0 && len(items) == 0 {
			c.Status(http.StatusBadGateway)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}
