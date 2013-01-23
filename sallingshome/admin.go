package sallingshome

import (
	"net/http"
	"strconv"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

func init() {
	http.HandleFunc("/admin/tasks/new", adminNewTask)
	http.HandleFunc("/admin/tasks/toggle", adminToggleTask)
	http.HandleFunc("/admin/tasks/", adminListTasks)

	http.HandleFunc("/admin/users/", adminListUsers)
	http.HandleFunc("/admin/users/new", adminNewUser)

	http.HandleFunc("/admin/topay/", adminListUnpaid)
	http.HandleFunc("/admin/topay/update/", adminMarkPaid)

	http.HandleFunc("/admin/", serveAdmin)
}

func iterateTasks(c appengine.Context) chan Task {
	ch := make(chan Task)

	q := datastore.NewQuery("Task").Order("Disabled").Order("Name")

	go func() {
		defer close(ch)
		for t := q.Run(c); ; {
			var x Task
			k, err := t.Next(&x)
			if err != nil {
				break
			}
			x.Key = k
			ch <- x
		}
	}()

	return ch
}

func adminToggleTask(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	k, err := datastore.DecodeKey(r.FormValue("id"))
	if err != nil {
		panic(err)
	}

	c.Infof("Toggling object with key %v", k)

	task := &Task{}
	if err := datastore.Get(c, k, task); err != nil {
		panic(err)
	}

	task.Disabled = !task.Disabled

	if _, err := datastore.Put(c, k, task); err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/admin/tasks/", 307)
}

func adminNewTask(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	r.ParseForm()

	asInt := func(s string) int {
		i, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}
		return i
	}

	task := Task{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Assignee:    r.FormValue("assignee"),
		RType:       r.FormValue("rtype"),
		Period:      asInt(r.FormValue("period")),
		Value:       asInt(r.FormValue("value")),
		Next:        time.Now().UTC(),
	}

	k, err := datastore.Put(c,
		datastore.NewIncompleteKey(c, "Task", nil), &task)
	if err != nil {
		c.Warningf("Error storing task:  %v", err)
		panic(err)
	}
	c.Infof("Stored new thing with key %v", k)

	http.Redirect(w, r, "/admin/tasks/", 307)
}

func adminListTasks(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	templates.ExecuteTemplate(w, "admin_tasks.html",
		map[string]interface{}{
			"results":   iterateTasks(c),
			"assignees": iterateUsers(c),
		})
}

func iterateUsers(c appengine.Context) chan User {
	ch := make(chan User)

	q := datastore.NewQuery("User").Order("Name")

	go func() {
		defer close(ch)
		for t := q.Run(c); ; {
			var x User
			k, err := t.Next(&x)
			if err != nil {
				break
			}
			x.Key = k
			ch <- x
		}
	}()

	return ch
}

func adminNewUser(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	r.ParseForm()

	user := User{
		Name:  r.FormValue("name"),
		Email: r.FormValue("email"),
	}

	k, err := datastore.Put(c,
		datastore.NewKey(c, "User", user.Email, 0, nil), &user)
	if err != nil {
		c.Warningf("Error storing user:  %v", err)
		panic(err)
	}
	c.Infof("Stored new thing with key %v", k)

	http.Redirect(w, r, "/admin/users/", 307)
}

func adminListUsers(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "admin_users.html",
		map[string]interface{}{
			"results": iterateUsers(appengine.NewContext(r)),
		})
}

func iterateTopay(c appengine.Context) chan LoggedTask {
	q := datastore.NewQuery("LoggedTask").
		Filter("Paid = ", false).
		Order("Completed")

	ch := make(chan LoggedTask)
	go func() {
		defer close(ch)
		for t := q.Run(c); ; {
			var x LoggedTask
			k, err := t.Next(&x)
			if err != nil {
				break
			}
			x.Key = k
			ch <- x
		}
	}()

	return ch
}

func adminListUnpaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	templates.ExecuteTemplate(w, "admin_topay.html",
		map[string]interface{}{
			"topay": iterateTopay(c),
		})
}

func adminMarkPaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	r.ParseForm()
	keys := make([]*datastore.Key, 0, len(r.Form["pay"]))
	for _, s := range r.Form["pay"] {
		k, err := datastore.DecodeKey(s)
		if err != nil {
			panic(err)
		}
		keys = append(keys, k)
	}
	tasks := make([]LoggedTask, len(keys))

	err := datastore.GetMulti(c, keys, tasks)
	if err != nil {
		panic(err)
	}

	now := time.Now().UTC()
	for i := range tasks {
		tasks[i].Paid = true
		tasks[i].PaidTime = now
	}

	_, err = datastore.PutMulti(c, keys, tasks)
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/admin/", 303)
}

func serveAdmin(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)

	c.Infof("Got admin request from %v", u)

	templates.ExecuteTemplate(w, "admin.html", u)
}
