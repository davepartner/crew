package research

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Result struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet"`
	Provider string `json:"provider"`
}

type Quote struct {
	Ticker    string  `json:"ticker"`
	Price     float64 `json:"price"`
	Previous  float64 `json:"previous_close"`
	ChangePct float64 `json:"change_percent"`
	Currency  string  `json:"currency"`
	AsOf      string  `json:"as_of"`
	URL       string  `json:"url"`
}

var tags = regexp.MustCompile(`<[^>]+>`)

func Search(ctx context.Context, query, braveKey string) []Result {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	if braveKey != "" {
		if results := brave(ctx, query, braveKey); len(results) > 0 {
			return results
		}
	}

	return news(ctx, query)
}

func brave(ctx context.Context, query, key string) []Result {
	endpoint := "https://api.search.brave.com/res/v1/web/search?count=5&q=" + url.QueryEscape(query)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Subscription-Token", key)

	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		return nil
	}
	defer response.Body.Close()

	var payload struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil
	}

	out := make([]Result, 0, len(payload.Web.Results))
	for _, item := range payload.Web.Results {
		out = append(out, Result{
			Title: item.Title, URL: item.URL,
			Snippet:  strings.Join(strings.Fields(strip(item.Description)), " "),
			Provider: "Brave Search",
		})
	}
	return out
}

func news(ctx context.Context, query string) []Result {
	endpoint := "https://news.google.com/rss/search?q=" + url.QueryEscape(query) +
		"&hl=en-US&gl=US&ceid=US:en"
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CrewResearch/1.0)")

	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		return nil
	}
	defer response.Body.Close()

	var feed struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				PubDate     string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&feed); err != nil {
		return nil
	}

	out := make([]Result, 0, 5)
	for _, item := range feed.Channel.Items {
		if len(out) >= 5 {
			break
		}
		snippet := strings.Join(strings.Fields(strip(item.Description)), " ")
		if len(snippet) > 280 {
			snippet = snippet[:280]
		}
		out = append(out, Result{
			Title: strings.TrimSpace(item.Title), URL: item.Link,
			Snippet: snippet, Provider: "Google News",
		})
	}
	return out
}

func Market(ctx context.Context, ticker string) (Quote, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return Quote{}, fmt.Errorf("no ticker")
	}

	endpoint := "https://query1.finance.yahoo.com/v8/finance/chart/" + url.PathEscape(ticker) + "?range=1d&interval=1d"
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CrewResearch/1.0)")

	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return Quote{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("quote unavailable")
	}

	var payload struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Symbol             string  `json:"symbol"`
					Currency           string  `json:"currency"`
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					ChartPreviousClose float64 `json:"chartPreviousClose"`
					ShortName          string  `json:"shortName"`
					RegularMarketTime  int64   `json:"regularMarketTime"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil || len(payload.Chart.Result) == 0 {
		return Quote{}, fmt.Errorf("quote unavailable")
	}

	meta := payload.Chart.Result[0].Meta
	quote := Quote{
		Ticker: meta.Symbol, Price: meta.RegularMarketPrice, Previous: meta.ChartPreviousClose,
		Currency: meta.Currency, URL: "https://finance.yahoo.com/quote/" + url.PathEscape(ticker),
	}
	if quote.Previous > 0 {
		quote.ChangePct = (quote.Price - quote.Previous) / quote.Previous * 100
	}
	if meta.RegularMarketTime > 0 {
		quote.AsOf = time.Unix(meta.RegularMarketTime, 0).UTC().Format(time.RFC3339)
	}
	return quote, nil
}

func strip(value string) string {
	return html.UnescapeString(tags.ReplaceAllString(value, " "))
}
