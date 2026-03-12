package model

import "encoding/json"

type CEPResult struct {
	Provider string          `json:"provider"`
	Data     json.RawMessage `json:"data"`
}
