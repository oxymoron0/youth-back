package model

import "encoding/json"

type FeatureCollection struct {
	Type     string          `json:"type"`
	Features json.RawMessage `json:"features"`
}
