package sallingshome

import (
	"time"

	"appengine/datastore"
)

type Task struct {
	Name     string    `json:"name"`
	Period   int       `json:"period"`
	RType    string    `json:"repeatType"`
	Assignee string    `json:"assignee"`
	Value    int       `json:"value"` // cents
	Next     time.Time `json:next"`
	Disabled bool      `json:"disabled"`

	Key *datastore.Key `datastore:"-"`
}

type LoggedTask struct {
	Task      *datastore.Key
	User      *datastore.Key
	Completed time.Time
	Paid      *time.Time
}
