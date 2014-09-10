package sallingshome

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").Funcs(map[string]interface{}{
		"agecss": func(t time.Time) string {
			if t.Before(time.Now().Add(time.Hour * -24 * 14)) {
				return "old"
			}
			return ""
		},
		"money": moneyFmt,
	}).ParseGlob("templates/*")
	if err != nil {
		panic("Couldn't parse templates: " + err.Error())
	}
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/api/currentuser/", currentUser)
	http.HandleFunc("/complete", serveComplete)
	http.HandleFunc("/logout", logoutRedirect)
}

func moneyFmt(i int) string {
	dollars := i / 100
	cents := i % 100
	return fmt.Sprintf("$%d.%02d", dollars, cents)
}

func logoutRedirect(w http.ResponseWriter, r *http.Request) {
	url, _ := user.LogoutURL(appengine.NewContext(r), "/")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func currentUser(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	mustEncode(w, user.Current(c))
}

func iterateUserTasks(c appengine.Context, u User, auto bool) chan Task {
	ch := make(chan Task)

	wg := sync.WaitGroup{}
	now := time.Now()

	querier := func(assignee string) {
		defer wg.Done()

		q := datastore.NewQuery("Task").
			Filter("Next < ", now).
			Filter("Disabled = ", false).
			Filter("Assignee = ", assignee).
			Filter("Automatic = ", auto).
			Order("Next")

		for t := q.Run(c); ; {
			var x Task
			k, err := t.Next(&x)
			if err == datastore.Done {
				break
			} else if err != nil {
				c.Errorf("Error retrieving tasks: %v", err)
				return
			}
			x.Key = k
			ch <- x
		}
	}

	wg.Add(2)
	go func() {
		wg.Wait()
		close(ch)
	}()

	go querier("")
	go querier(u.Name)

	return ch
}

func getUserByEmail(c appengine.Context, e string) (rv User, err error) {
	k := datastore.NewKey(c, "User", e, 0, nil)
	err = datastore.Get(c, k, &rv)
	rv.Key = k
	return
}

// Get the user record for the datastore user.
func getUser(c appengine.Context, u *user.User) (rv User, err error) {
	return getUserByEmail(c, u.Email)
}

func mustEncode(w io.Writer, i interface{}) {
	if headered, ok := w.(http.ResponseWriter); ok {
		headered.Header().Set("Cache-Control", "no-cache")
		headered.Header().Set("Content-type", "application/json")
	}

	e := json.NewEncoder(w)
	if err := e.Encode(i); err != nil {
		panic(err)
	}
}

func serveComplete(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)
	su, err := getUser(c, u)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "You are not permitted, %v", u)
		return
	}

	taskIds := []*datastore.Key{}
	for _, s := range r.Form["task"] {
		k, err := datastore.DecodeKey(s)
		if err != nil {
			panic(err)
		}
		taskIds = append(taskIds, k)
	}

	c.Infof("Doing tasks for %v:  %v", su, taskIds)

	tasks := make([]Task, len(taskIds))
	err = datastore.GetMulti(c, taskIds, tasks)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	storeKeys := make([]*datastore.Key, 0, 2*len(taskIds))
	vals := []interface{}{}
	for i := range tasks {
		if tasks[i].Next.Before(now) {
			tasks[i].updateTime()
			tasks[i].Prev = now
			storeKeys = append(storeKeys, taskIds[i])

			vals = append(vals, &tasks[i])

			storeKeys = append(storeKeys,
				datastore.NewIncompleteKey(c, "LoggedTask", nil))
			vals = append(vals, &LoggedTask{
				Task:      taskIds[i],
				User:      su.Key,
				Completed: now,
				Who:       su.Name,
				Name:      tasks[i].Name,
				Amount:    tasks[i].Value,
			})
		}
	}

	c.Infof("Putting %#v in %v", vals, storeKeys)

	_, err = datastore.PutMulti(c, storeKeys, vals)
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func execTemplate(c appengine.Context, w io.Writer, name string,
	obj interface{}) error {

	err := templates.ExecuteTemplate(w, name, obj)

	if err != nil {
		c.Errorf("Error executing template %v: %v", name, err)
		if wh, ok := w.(http.ResponseWriter); ok {
			http.Error(wh, "Error executing template", 500)
		}
	}
	return err
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)
	su, err := getUser(c, u)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "You are not permitted, %v", u)
		return
	}

	c.Infof("Got a request from %v", u)

	execTemplate(c, w, "index.html",
		map[string]interface{}{
			"user":  u,
			"tasks": iterateUserTasks(c, su, false),
		})
}
