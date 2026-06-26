package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kr-metro-api/model"
)

const (
	defaultListURL   = "https://soco.seoul.go.kr/youth/pgm/home/yohome/mainYoHomeListJson.json"
	defaultImageURL  = "https://soco.seoul.go.kr/cohome/cmmn/file/fileDown.do"
	defaultDetailURL = "https://soco.seoul.go.kr/youth/pgm/home/yohome/view.do"
	defaultTimeout   = 15 * time.Second
	imageTimeout     = 30 * time.Second
	userAgent        = "KR-Metro-Sync/1.0"
	requestBody      = "rowCount=200"
)

// Seoul coordinate bounds (matches etl/06_youth_housing.py validation).
const (
	detailLonMin, detailLonMax = 126.5, 127.5
	detailLatMin, detailLatMax = 37.0, 38.0
)

type HousingClient struct {
	httpClient *http.Client
	listURL    string
	imageURL   string
	detailURL  string
}

func NewHousingClient() *HousingClient {
	return &HousingClient{
		httpClient: &http.Client{Timeout: imageTimeout},
		listURL:    defaultListURL,
		imageURL:   defaultImageURL,
		detailURL:  defaultDetailURL,
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

// WithImageURL overrides the default image download URL (useful for testing).
func (c *HousingClient) WithImageURL(url string) *HousingClient {
	c.imageURL = url
	return c
}

// WithDetailURL overrides the default detail-page URL (useful for testing).
func (c *HousingClient) WithDetailURL(url string) *HousingClient {
	c.detailURL = url
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

// FetchImage downloads a single housing image via the source fileDown endpoint.
// Returns the raw bytes and the Content-Type reported by the source.
func (c *HousingClient) FetchImage(ctx context.Context, fileID string, fileSn int) ([]byte, string, error) {
	if fileID == "" {
		return nil, "", fmt.Errorf("empty fileID")
	}
	q := url.Values{}
	q.Set("atchFileId", fileID)
	q.Set("fileSn", strconv.Itoa(fileSn))
	reqURL := c.imageURL + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty image body")
	}

	// The source sometimes returns a non-image Content-Type (e.g.
	// application/octet-stream) for actual images. Prefer sniffing the bytes
	// whenever the header isn't an image type, so stored data is accurate.
	contentType := resp.Header.Get("Content-Type")
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	if !strings.HasPrefix(contentType, "image/") {
		if sniffed := http.DetectContentType(data); strings.HasPrefix(sniffed, "image/") || contentType == "" {
			contentType = sniffed
		}
	}
	return data, contentType, nil
}

// FetchDetail downloads a housing's detail page and parses the fields that the
// list API does not provide (coordinates, phone, homepage, dates, units, etc.).
func (c *HousingClient) FetchDetail(ctx context.Context, homeCode string) (model.HousingDetailFields, error) {
	var f model.HousingDetailFields
	if homeCode == "" {
		return f, fmt.Errorf("empty homeCode")
	}
	q := url.Values{}
	q.Set("menuNo", "400002")
	q.Set("homeCode", homeCode)
	reqURL := c.detailURL + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return f, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return f, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return f, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return f, fmt.Errorf("read body: %w", err)
	}
	return parseDetailHTML(string(body)), nil
}

// Detail-page parsing regexes (port of etl/06_youth_housing.py parse_detail_page).
var (
	reXpos   = regexp.MustCompile(`var\s+xpos\s*=\s*"([^"]+)"`)
	reYpos   = regexp.MustCompile(`var\s+ypos\s*=\s*"([^"]+)"`)
	rePBlock = regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`)
	reTag    = regexp.MustCompile(`(?s)<[^>]+>`)
	reWS     = regexp.MustCompile(`\s+`)
	reHref   = regexp.MustCompile(`(?i)href\s*=\s*"([^"]+)"`)
	reDate   = regexp.MustCompile(`(\d{4})[-./](\d{1,2})[-./](\d{1,2})`)
)

func normalizeDate(s string) string {
	m := reDate.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	mo, _ := strconv.Atoi(m[2])
	d, _ := strconv.Atoi(m[3])
	return fmt.Sprintf("%s-%02d-%02d", m[1], mo, d)
}

// parseDetailHTML extracts detail fields from a housing detail page's HTML.
func parseDetailHTML(htmlStr string) model.HousingDetailFields {
	var f model.HousingDetailFields

	if xm, ym := reXpos.FindStringSubmatch(htmlStr), reYpos.FindStringSubmatch(htmlStr); xm != nil && ym != nil {
		lon, errx := strconv.ParseFloat(xm[1], 64)
		lat, erry := strconv.ParseFloat(ym[1], 64)
		if errx == nil && erry == nil &&
			lon >= detailLonMin && lon <= detailLonMax &&
			lat >= detailLatMin && lat <= detailLatMax {
			f.Longitude = &lon
			f.Latitude = &lat
		}
	}

	setIf := func(dst *string, label, text string) {
		if *dst != "" || !strings.Contains(text, label) {
			return
		}
		if i := strings.Index(text, ":"); i >= 0 {
			if v := strings.TrimSpace(text[i+1:]); v != "" {
				*dst = v
			}
		}
	}

	for _, b := range rePBlock.FindAllStringSubmatch(htmlStr, -1) {
		inner := b[1]
		text := reWS.ReplaceAllString(strings.TrimSpace(html.UnescapeString(reTag.ReplaceAllString(inner, ""))), " ")
		setIf(&f.Phone, "대표전화", text)
		setIf(&f.FirstRecruitDate, "최초모집공고일", text)
		setIf(&f.MoveInDate, "입주(예정)일", text)
		if f.MoveInDate == "" {
			setIf(&f.MoveInDate, "입주예정일", text)
		}
		setIf(&f.TotalUnits, "규모", text)
		setIf(&f.Developer, "시행사", text)
		setIf(&f.Constructor, "시공사", text)
		if f.HomepageURL == "" && strings.Contains(text, "홈페이지") {
			if hm := reHref.FindStringSubmatch(inner); hm != nil {
				f.HomepageURL = hm[1]
			}
		}
	}

	f.FirstRecruitDate = normalizeDate(f.FirstRecruitDate)
	f.MoveInDate = normalizeDate(f.MoveInDate)
	return f
}
