package sallingshome

import (
	"time"

	"appengine/datastore"
)

type Task struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Period      int       `json:"period"`
	RType       string    `json:"repeatType"`
	Assignee    string    `json:"assignee"`
	Value       int       `json:"value"` // cents
	Prev        time.Time `json:"prev"`
	Next        time.Time `json:next"`
	Disabled    bool      `json:"disabled"`

	Key *datastore.Key `datastore:"-"`
}

func (t *Task) updateTime() {
	t.Next = time.Now().Add(time.Hour * 24 * time.Duration(t.Period))

	h, m, s := t.Next.Clock()
	d := time.Duration(time.Hour*time.Duration(h)) +
		time.Duration(time.Minute*time.Duration(m)) +
		time.Duration(time.Second*time.Duration(s)) +
		time.Duration(t.Next.Nanosecond())

	t.Next = t.Next.Add(-d)
}

type User struct {
	Name     string
	Email    string
	Disabled bool

	Key *datastore.Key `datastore:"-"`
}

type LoggedTask struct {
	Task      *datastore.Key
	User      *datastore.Key
	Completed time.Time
	PaidTime  time.Time
	Paid      bool
}
