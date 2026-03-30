package model

type Line struct {
	LineID       int      `json:"line_id"`
	LineCode     string   `json:"line_code"`
	LineName     string   `json:"line_name"`
	LineColor    *string  `json:"line_color"`
	StartStation *string  `json:"start_station"`
	EndStation   *string  `json:"end_station"`
	OpenedDate   *string  `json:"opened_date"`
	LineLengthKm *float64 `json:"line_length_km"`
	OperatorName *string  `json:"operator_name"`
	StationCount int      `json:"station_count"`
}
