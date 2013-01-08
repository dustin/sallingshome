package sallingshome

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"

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
	return
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
