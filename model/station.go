package model

import "encoding/json"

type Station struct {
	StationID     int      `json:"station_id"`
	StationCode   string   `json:"station_code"`
	StationName   string   `json:"station_name"`
	StationNameEn *string  `json:"station_name_en"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	IsTransfer    bool     `json:"is_transfer"`
}

type StationDetail struct {
	StationID     int              `json:"station_id"`
	StationCode   string           `json:"station_code"`
	StationName   string           `json:"station_name"`
	StationNameEn *string          `json:"station_name_en"`
	StationNameCn *string          `json:"station_name_cn"`
	Address       *string          `json:"address"`
	Phone         *string          `json:"phone"`
	Latitude      *float64         `json:"latitude"`
	Longitude     *float64         `json:"longitude"`
	IsTransfer    bool             `json:"is_transfer"`
	Lines         []StationLine    `json:"lines"`
	Transfers     []StationTransfer `json:"transfers"`
	Exits         []StationExit    `json:"exits"`
}

type StationLine struct {
	LineID    int     `json:"line_id"`
	LineCode  string  `json:"line_code"`
	LineName  string  `json:"line_name"`
	LineColor *string `json:"line_color"`
}

type StationTransfer struct {
	TransferID   int    `json:"transfer_id"`
	ToStationID  int    `json:"to_station_id"`
	ToStation    string `json:"to_station_name"`
	FromLineID   int    `json:"from_line_id"`
	FromLineName string `json:"from_line_name"`
	ToLineID     int    `json:"to_line_id"`
	ToLineName   string `json:"to_line_name"`
	TransferTime *int   `json:"transfer_time"`
}

type StationExit struct {
	ExitID       int      `json:"exit_id"`
	ExitNumber   string   `json:"exit_number"`
	ExitName     *string  `json:"exit_name"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	IsAccessible *bool    `json:"is_accessible"`
}

type SearchResult struct {
	StationID       int             `json:"station_id"`
	StationCode     string          `json:"station_code"`
	StationName     string          `json:"station_name"`
	StationNameEn   *string         `json:"station_name_en"`
	Latitude        *float64        `json:"latitude"`
	Longitude       *float64        `json:"longitude"`
	IsTransfer      bool            `json:"is_transfer"`
	SimilarityScore float32         `json:"similarity_score"`
	Lines           json.RawMessage `json:"lines"`
}
