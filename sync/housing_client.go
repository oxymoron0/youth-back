package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kr-metro-api/model"
)

const (
	defaultListURL = "https://soco.seoul.go.kr/youth/pgm/home/yohome/mainYoHomeListJson.json"
	defaultTimeout = 15 * time.Second
	userAgent      = "KR-Metro-Sync/1.0"
	requestBody    = "rowCount=200"
)

type HousingClient struct {
	httpClient *http.Client
	listURL    string
}

func NewHousingClient() *HousingClient {
	return &HousingClient{
		httpClient: &http.Client{Timeout: defaultTimeout},
		listURL:    defaultListURL,
	}
}

// WithHTTPClient overrides the default HTTP client (useful for testing).
func (c *HousingClient) WithHTTPClient(client *http.Client) *HousingClient {
	c.httpClient = client
	return c
}

// WithListURL overrides the default list API URL (useful for testing).
func (c *HousingClient) WithListURL(url string) *HousingClient {
	c.listURL = url
	return c
}

type listAPIResponse struct {
	ResultList []model.HousingSyncItem `json:"resultList"`
}

func (c *HousingClient) FetchList(ctx context.Context) ([]model.HousingSyncItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.listURL, strings.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result listAPIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}

	if len(result.ResultList) == 0 {
		return nil, fmt.Errorf("empty result list (possible API issue)")
	}

	return result.ResultList, nil
}
