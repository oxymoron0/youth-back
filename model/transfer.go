package model

type Transfer struct {
	TransferID    int    `json:"transfer_id"`
	FromStationID int    `json:"from_station_id"`
	FromStation   string `json:"from_station_name"`
	FromLineID    int    `json:"from_line_id"`
	FromLineName  string `json:"from_line_name"`
	ToStationID   int    `json:"to_station_id"`
	ToStation     string `json:"to_station_name"`
	ToLineID      int    `json:"to_line_id"`
	ToLineName    string `json:"to_line_name"`
	TransferTime  *int   `json:"transfer_time"`
}
