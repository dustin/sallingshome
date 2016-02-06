package sallingshome

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
)

var templates = template.Must(template.New("").Funcs(map[string]interface{}{
	"agecss": func(t time.Time) string {
		if t.Before(time.Now().Add(time.Hour * -24 * 14)) {
			return "old"
		}
		return ""
	},
	"money": moneyFmt,
}).ParseGlob("templates/*"))

func init() {
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

	mustEncode(c, w, user.Current(c))
}

func iterateUserTasks(c context.Context, u User, auto bool) chan Task {
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
				log.Errorf(c, "Error retrieving tasks: %v", err)
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
	go querier(u.Email)

	return ch
}

func getUserByEmail(c context.Context, e string) (rv User, err error) {
	k := datastore.NewKey(c, "User", e, 0, nil)
	err = datastore.Get(c, k, &rv)
	rv.Key = k
	return
}

// Get the user record for the datastore user.
func getUser(c context.Context, u *user.User) (rv User, err error) {
	return getUserByEmail(c, u.Email)
}

func mustEncode(c context.Context, w io.Writer, i interface{}) {
	if headered, ok := w.(http.ResponseWriter); ok {
		headered.Header().Set("Cache-Control", "no-cache")
		headered.Header().Set("Content-type", "application/json")
	}

	if err := json.NewEncoder(w).Encode(i); err != nil {
		log.Errorf(c, "Error json encoding: %v", err)
		if h, ok := w.(http.ResponseWriter); ok {
			http.Error(h, err.Error(), 500)
		}
		return
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

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		fmt.Fprintf(w, "Can't parse form", u)
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

	log.Infof(c, "Doing tasks for %v:  %v", su, taskIds)

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

	log.Infof(c, "Putting %#v in %v", vals, storeKeys)

	_, err = datastore.PutMulti(c, storeKeys, vals)
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func execTemplate(c context.Context, w io.Writer, name string,
	obj interface{}) error {

	err := templates.ExecuteTemplate(w, name, obj)

	if err != nil {
		log.Errorf(c, "Error executing template %v: %v", name, err)
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

	log.Infof(c, "Got a request from %v", u)

	execTemplate(c, w, "index.html",
		map[string]interface{}{
			"user":  u,
			"tasks": iterateUserTasks(c, su, false),
		})
}
