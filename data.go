package sallingshome

import (
	"time"

	"google.golang.org/appengine/datastore"
)

type Task struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Period      int       `json:"period"`
	RType       string    `json:"repeatType"`
	Assignee    string    `json:"assignee"`
	Value       int       `json:"value"` // cents
	Prev        time.Time `json:"prev"`
	Next        time.Time `json:"next"`
	Disabled    bool      `json:"disabled"`
	Automatic   bool      `json:"automatic"`

	Key *datastore.Key `datastore:"-"`
}

func (t *Task) setKey(to *datastore.Key) {
	t.Key = to
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
	Name     string `json:"name"`
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`

	Key       *datastore.Key `datastore:"-"`
	Projected int64          `datastore:"-"`
	Earned    int64          `datastore:"-"`
}

func (u *User) setKey(to *datastore.Key) {
	u.Key = to
}

type LoggedTask struct {
	Task      *datastore.Key
	User      *datastore.Key
	Completed time.Time `json:"completed_at"`
	PaidTime  time.Time `json:"paid_at"`
	Paid      bool      `json:"paid"`

	// Copied from the task for ease of use
	Who    string `json:"who"`
	Name   string `json:"name"`
	Amount int    `json:"amount"`

	Key *datastore.Key `datastore:"-",json:"key"`
}

func (l *LoggedTask) setKey(to *datastore.Key) {
	l.Key = to
}

type Keyable interface {
	setKey(*datastore.Key)
}
