package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// NCP Maps Geocoding REST API (server-side). Uses the current `maps.apigw`
	// gateway; the legacy `naveropenapi.apigw` domain returns 401 for Maps keys.
	defaultGeocodeURL = "https://maps.apigw.ntruss.com/map-geocode/v2/geocode"
	// NCP Maps Reverse Geocoding REST API (coordinates -> administrative region).
	defaultReverseGeocodeURL = "https://maps.apigw.ntruss.com/map-reversegeocode/v2/gc"
	geocodeTimeout           = 10 * time.Second
)

// Geocoder resolves an address to coordinates via the NCP Geocoding REST API.
// It is a no-op (Enabled()==false) when credentials are not configured.
type Geocoder struct {
	httpClient *http.Client
	url        string
	reverseURL string
	keyID      string
	keySecret  string
}

func NewGeocoder(keyID, keySecret string) *Geocoder {
	return &Geocoder{
		httpClient: &http.Client{Timeout: geocodeTimeout},
		url:        defaultGeocodeURL,
		reverseURL: defaultReverseGeocodeURL,
		keyID:      keyID,
		keySecret:  keySecret,
	}
}

// WithHTTPClient overrides the default HTTP client (useful for testing).
func (g *Geocoder) WithHTTPClient(c *http.Client) *Geocoder {
	g.httpClient = c
	return g
}

// WithURL overrides the default endpoint (useful for testing).
func (g *Geocoder) WithURL(u string) *Geocoder {
	g.url = u
	return g
}

// WithReverseURL overrides the default reverse-geocode endpoint (useful for testing).
func (g *Geocoder) WithReverseURL(u string) *Geocoder {
	g.reverseURL = u
	return g
}

// Enabled reports whether credentials are configured. Nil-safe.
func (g *Geocoder) Enabled() bool {
	return g != nil && g.keyID != "" && g.keySecret != ""
}

type geocodeResponse struct {
	Status    string `json:"status"`
	Addresses []struct {
		X string `json:"x"` // longitude
		Y string `json:"y"` // latitude
	} `json:"addresses"`
	ErrorMessage string `json:"errorMessage"`
}

// Geocode resolves an address to (longitude, latitude). ok=false when the API
// returned no match (not an error).
func (g *Geocoder) Geocode(ctx context.Context, address string) (lng, lat float64, ok bool, err error) {
	if address == "" {
		return 0, 0, false, fmt.Errorf("empty address")
	}

	q := url.Values{}
	q.Set("query", address)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.url+"?"+q.Encode(), nil)
	if err != nil {
		return 0, 0, false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-ncp-apigw-api-key-id", g.keyID)
	req.Header.Set("x-ncp-apigw-api-key", g.keySecret)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return 0, 0, false, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, false, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, 0, false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var gr geocodeResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return 0, 0, false, fmt.Errorf("json decode: %w", err)
	}
	if gr.Status != "OK" {
		return 0, 0, false, fmt.Errorf("geocode status %q: %s", gr.Status, gr.ErrorMessage)
	}
	if len(gr.Addresses) == 0 {
		return 0, 0, false, nil // valid response, no match
	}

	a := gr.Addresses[0]
	lng, err = strconv.ParseFloat(a.X, 64)
	if err != nil {
		return 0, 0, false, fmt.Errorf("parse longitude %q: %w", a.X, err)
	}
	lat, err = strconv.ParseFloat(a.Y, 64)
	if err != nil {
		return 0, 0, false, fmt.Errorf("parse latitude %q: %w", a.Y, err)
	}
	return lng, lat, true, nil
}

type reverseGeocodeResponse struct {
	Status struct {
		Code int    `json:"code"`
		Name string `json:"name"`
	} `json:"status"`
	Results []struct {
		Region struct {
			Area2 struct {
				Name string `json:"name"`
			} `json:"area2"` // 시군구 (자치구)
			Area3 struct {
				Name string `json:"name"`
			} `json:"area3"` // 읍면동 (법정동)
		} `json:"region"`
	} `json:"results"`
}

// ReverseGeocode resolves coordinates to (gu, dong) administrative names via the
// NCP Reverse Geocoding REST API (orders=legalcode → 법정동). ok=false when the
// API returned no region (not an error).
func (g *Geocoder) ReverseGeocode(ctx context.Context, lng, lat float64) (gu, dong string, ok bool, err error) {
	q := url.Values{}
	q.Set("coords", strconv.FormatFloat(lng, 'f', -1, 64)+","+strconv.FormatFloat(lat, 'f', -1, 64))
	q.Set("output", "json")
	q.Set("orders", "legalcode")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.reverseURL+"?"+q.Encode(), nil)
	if err != nil {
		return "", "", false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-ncp-apigw-api-key-id", g.keyID)
	req.Header.Set("x-ncp-apigw-api-key", g.keySecret)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", "", false, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", false, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var rr reverseGeocodeResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return "", "", false, fmt.Errorf("json decode: %w", err)
	}
	// status.code 3 = "no results" — a valid response with no match.
	if len(rr.Results) == 0 {
		return "", "", false, nil
	}

	region := rr.Results[0].Region
	gu = region.Area2.Name
	dong = region.Area3.Name
	if dong == "" {
		return gu, "", false, nil // region present but no dong resolved
	}
	return gu, dong, true, nil
}
