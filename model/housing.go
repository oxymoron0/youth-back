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
}

// HousingSyncResult represents the outcome of a single sync cycle.
type HousingSyncResult struct {
	FetchedCount int       `json:"fetched_count"`
	UpdatedCount int       `json:"updated_count"`
	NewCount     int       `json:"new_count"`
	Duration     string    `json:"duration"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	Error        string    `json:"error,omitempty"`
}

type HousingListItem struct {
	HomeCode     string   `json:"home_code"`
	HomeName     string   `json:"home_name"`
	SupplyStatus string   `json:"supply_status"`
	AddressGu    *string  `json:"address_gu"`
	Longitude    *float64 `json:"longitude"`
	Latitude     *float64 `json:"latitude"`
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
}

type NearbyStation struct {
	StationID   int      `json:"station_id"`
	StationName string   `json:"station_name"`
	LineNames   string   `json:"line_names"`
	DistanceM   float64  `json:"distance_m"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}
