package sallingshome

import (
	"fmt"
	"html/template"
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
	templates, err = template.ParseGlob(fmt.Sprintf("templates/%c.html", '*'))
	if err != nil {
		panic("Couldn't parse templates.")
	}
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/complete", serveComplete)
	http.HandleFunc("/logout", logoutRedirect)
}

func logoutRedirect(w http.ResponseWriter, r *http.Request) {
	url, _ := user.LogoutURL(appengine.NewContext(r), "/")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func iterateUserTasks(c appengine.Context, u User) chan Task {
	ch := make(chan Task)

	wg := sync.WaitGroup{}

	querier := func(assignee string) {
		defer wg.Done()

		q := datastore.NewQuery("Task").
			Filter("Disabled = ", false).
			Filter("Assignee = ", assignee).
			Order("Name")

		for t := q.Run(c); ; {
			var x Task
			k, err := t.Next(&x)
			if err == datastore.Done {
				break
			}
			x.Key = k
			ch <- x
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	wg.Add(2)
	go querier("")
	go querier(u.Name)

	return ch
}

// Get the user record for the datastore user.
func getUser(c appengine.Context, u *user.User) (rv User, err error) {
	k := datastore.NewKey(c, "User", u.Email, 0, nil)
	err = datastore.Get(c, k, &rv)
	rv.Key = k
	return
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

	r.ParseForm()
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
	storeKeys := make([]*datastore.Key, len(taskIds))
	vals := []interface{}{}
	copy(storeKeys, taskIds)
	for i := range tasks {
		tasks[i].updateTime()
		tasks[i].Prev = now
		storeKeys = append(storeKeys,
			datastore.NewIncompleteKey(c, "LoggedTask", nil))

		vals = append(vals, &tasks[i])
	}

	for i := range tasks {
		vals = append(vals, &LoggedTask{
			Task:      taskIds[i],
			User:      su.Key,
			Completed: now,
		})
	}

	c.Infof("Putting %#v in %v", vals, storeKeys)

	_, err = datastore.PutMulti(c, storeKeys, vals)
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
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

	templates.ExecuteTemplate(w, "index.html",
		map[string]interface{}{
			"user":  u,
			"tasks": iterateUserTasks(c, su),
		})
}
