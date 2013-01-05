package sallingshome

import (
	"time"
)

type Task struct {
	Name   string    `json:"name"`
	Period int       `json:"period"`
	Value  int       `json:"value"` // cents
	Next   time.Time `json:next"`
}
