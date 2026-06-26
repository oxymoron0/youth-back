package model

import "time"

// HousingSyncItem maps the Seoul API list response fields.
type HousingSyncItem struct {
	HomeCode     string `json:"homeCode"`
	HomeName     string `json:"homeName"`
	Address      string `json:"adres"`
	AddressGu    string `json:"adresGu"`
	OptionSubway string `json:"optionSubway"`
	SupplyStatus string `json:"supplyStatus"`
	DepositLow   any    `json:"moneyDepositLow"`
	RentalLow    any    `json:"moneyRentalLow"`
	// Representative image reference (used to fetch the image via fileDown.do).
	FileID string `json:"fileId"`
	FileSn int    `json:"fileSn"`
}

// HousingSyncResult represents the outcome of a single sync cycle.
type HousingSyncResult struct {
	// APICount is the number of housings returned by the list API this cycle.
	// Tracked per sync so an increase vs the previous sync can be detected.
	APICount     int       `json:"api_count"`
	FetchedCount int       `json:"fetched_count"`
	UpdatedCount int       `json:"updated_count"`
	NewCount     int       `json:"new_count"`
	Duration     string    `json:"duration"`
	DurationMs   int64     `json:"duration_ms"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	Error        string    `json:"error,omitempty"`
}

// HousingDetailFields holds values scraped from a housing's detail page
// (coordinates + fields not present in the list API). Empty string / nil
// means "not found" and is treated as "leave existing value".
type HousingDetailFields struct {
	Longitude        *float64
	Latitude         *float64
	Phone            string
	HomepageURL      string
	FirstRecruitDate string // YYYY-MM-DD or ""
	MoveInDate       string // YYYY-MM-DD or ""
	TotalUnits       string
	Developer        string
	Constructor      string
	// 최저 보증금/월임대료 (목록 API가 null일 때 상세 호실표에서 파싱).
	DepositLow *int64
	RentalLow  *int64
}

type HousingListItem struct {
	HomeCode     string   `json:"home_code"`
	HomeName     string   `json:"home_name"`
	SupplyStatus string   `json:"supply_status"`
	AddressGu    *string  `json:"address_gu"`
	AddressDong  *string  `json:"address_dong"`
	Longitude    *float64 `json:"longitude"`
	Latitude     *float64 `json:"latitude"`
	DepositLow   *int64   `json:"deposit_low"`
	RentalLow    *int64   `json:"rental_low"`
}

// HousingCoordTarget identifies a housing that still needs coordinates
// (geocoding fallback input).
type HousingCoordTarget struct {
	HomeCode string
	Address  string
}

// HousingDongTarget identifies a housing that has coordinates but no dong yet
// (reverse-geocoding input).
type HousingDongTarget struct {
	HomeCode  string
	Longitude float64
	Latitude  float64
}

type HousingDetail struct {
	HousingID        int      `json:"housing_id"`
	HomeCode         string   `json:"home_code"`
	HomeName         string   `json:"home_name"`
	Address          *string  `json:"address"`
	AddressGu        *string  `json:"address_gu"`
	OptionSubway     *string  `json:"option_subway"`
	SupplyStatus     string   `json:"supply_status"`
	DepositLow       *int64   `json:"deposit_low"`
	RentalLow        *int64   `json:"rental_low"`
	HomepageURL      *string  `json:"homepage_url"`
	Phone            *string  `json:"phone"`
	FirstRecruitDate *string  `json:"first_recruit_date"`
	MoveInDate       *string  `json:"move_in_date"`
	TotalUnits       *string  `json:"total_units"`
	Developer        *string  `json:"developer"`
	Constructor      *string  `json:"constructor"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	HasImage         bool     `json:"has_image"`
}

// HousingImage holds a stored representative image and its HTTP cache metadata.
type HousingImage struct {
	HomeCode    string
	FileID      string
	FileSn      int
	ContentType string
	ETag        string
	Data        []byte
}

type NearbyStation struct {
	StationID   int      `json:"station_id"`
	StationName string   `json:"station_name"`
	LineNames   string   `json:"line_names"`
	DistanceM   float64  `json:"distance_m"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}
