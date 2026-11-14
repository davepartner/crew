package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Rates map[string]float64

func Fetch(ctx context.Context, key, base string) (Rates, error) {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		base = "USD"
	}
	key = strings.TrimSpace(key)

	if key != "" {
		rates, err := fetch(ctx, "https://v6.exchangerate-api.com/v6/"+key+"/latest/"+base)
		if err == nil {
			return rates, nil
		}
	}

	// The public endpoint needs no key. It is the same provider and is enough
	// for reporting; a key is still offered for a supported SLA.
	return fetch(ctx, "https://open.er-api.com/v6/latest/"+base)
}

func fetch(ctx context.Context, endpoint string) (Rates, error) {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	request.Header.Set("Accept", "application/json")

	response, err := (&http.Client{Timeout: 12 * time.Second}).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange-rate API returned HTTP %d", response.StatusCode)
	}

	var payload struct {
		Result          string             `json:"result"`
		ConversionRates map[string]float64 `json:"conversion_rates"`
		Rates           map[string]float64 `json:"rates"`
		ErrorType       string             `json:"error-type"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Result != "success" {
		if payload.ErrorType != "" {
			return nil, fmt.Errorf("exchange-rate API: %s", payload.ErrorType)
		}
		return nil, fmt.Errorf("exchange-rate API returned no rates")
	}
	rates := payload.ConversionRates
	if len(rates) == 0 {
		rates = payload.Rates
	}
	if len(rates) == 0 {
		return nil, fmt.Errorf("exchange-rate API returned no rates")
	}
	return Rates(rates), nil
}

func (r Rates) Convert(amount float64, from, to string) (float64, bool) {
	from = code(from)
	to = code(to)
	if from == "" || to == "" {
		return 0, false
	}
	if from == to {
		return amount, true
	}
	fromRate, ok := r[from]
	if !ok || fromRate == 0 {
		return 0, false
	}
	toRate, ok := r[to]
	if !ok {
		return 0, false
	}
	return amount / fromRate * toRate, true
}

func code(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if index := strings.Index(value, " "); index > 0 {
		value = value[:index]
	}
	if index := strings.Index(value, "("); index > 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}
