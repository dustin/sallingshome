package sallingshome

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"appengine/user"
)

func init() {
	http.HandleFunc("/admin/users/new", adminNewUser)

	http.HandleFunc("/admin/topay/update/", adminMarkPaid)

	http.HandleFunc("/api/admin/topay/", adminListUnpaid)
	http.HandleFunc("/admin/cron/topay/", adminMailUnpaid)

	http.HandleFunc("/admin/tasks/new", adminNewTask)
	http.HandleFunc("/api/admin/tasks/update/", adminUpdateTask)
	http.HandleFunc("/api/admin/tasks/makeAvailable/", adminTaskMakeAvailable)
	http.HandleFunc("/api/admin/tasks/makeUnavailable/", adminTaskMakeUnavailable)
	http.HandleFunc("/api/admin/tasks/markFor/", adminMarkTaskFor)
	http.HandleFunc("/api/admin/tasks/delete/", adminDeleteTask)
	http.HandleFunc("/api/admin/tasks/", adminListTasks)

	http.HandleFunc("/api/admin/users/", adminListUsers)

	http.HandleFunc("/admin/", serveStaticAdmin)
}

func serveStaticAdmin(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "admin.html", nil)
	if err != nil {
		panic(err)
	}
}

func asInt(s string) int {
	x, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return x
}

func adminMarkTaskFor(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	uu := &user.User{Email: r.FormValue("email")}
	u, err := getUser(c, uu)
	if err != nil {
		panic(err)
	}

	k, err := datastore.DecodeKey(r.FormValue("taskKey"))
	if err != nil {
		panic(err)
	}

	task := &Task{}
	if err := datastore.Get(c, k, task); err != nil {
		panic(err)
	}

	task.Prev = time.Now()
	task.updateTime()

	if _, err := datastore.Put(c, k, task); err != nil {
		panic(err)
	}

	// log done
	lk := datastore.NewIncompleteKey(c, "LoggedTask", nil)
	_, err = datastore.Put(c, lk, &LoggedTask{
		Task: k, User: u.Key, Completed: time.Now(), Who: u.Name,
		Name: task.Name, Amount: task.Value,
	})
	if err != nil {
		panic(err)
	}

	c.Infof("Administratively logged task %q for %q", task.Name, u.Name)

	mustEncode(w, map[string]interface{}{"next": task.Next})
}

func adminUpdateTask(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	tid := r.FormValue("taskKey")

	k, err := datastore.DecodeKey(tid)
	if err != nil {
		panic(err)
	}

	task := &Task{}
	if err := datastore.Get(c, k, task); err != nil {
		panic(err)
	}

	task.Name = r.FormValue("name")
	task.Description = r.FormValue("description")
	task.Value = asInt(r.FormValue("value"))
	task.Period = asInt(r.FormValue("period"))
	task.Disabled = r.FormValue("disabled") == "true"
	task.Assignee = r.FormValue("assignee")

	if _, err := datastore.Put(c, k, task); err != nil {
		panic(err)
	}
}

func adminTaskMakeAvailable(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	tid := r.FormValue("taskKey")

	k, err := datastore.DecodeKey(tid)
	if err != nil {
		panic(err)
	}

	task := &Task{}
	if err := datastore.Get(c, k, task); err != nil {
		panic(err)
	}

	task.Next = time.Now()

	if _, err := datastore.Put(c, k, task); err != nil {
		panic(err)
	}

	mustEncode(w, map[string]interface{}{"next": task.Next})
}

func adminTaskMakeUnavailable(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	tid := r.FormValue("taskKey")

	k, err := datastore.DecodeKey(tid)
	if err != nil {
		panic(err)
	}

	task := &Task{}
	if err := datastore.Get(c, k, task); err != nil {
		panic(err)
	}

	task.updateTime()

	if _, err := datastore.Put(c, k, task); err != nil {
		panic(err)
	}

	mustEncode(w, map[string]interface{}{"next": task.Next})
}

func adminDeleteTask(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	tid := r.FormValue("taskKey")

	k, err := datastore.DecodeKey(tid)
	if err != nil {
		panic(err)
	}

	if err := datastore.Delete(c, k); err != nil {
		panic(err)
	}

	w.WriteHeader(204)
}

func adminListTasks(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("Task").Order("Disabled").Order("Name")

	results := []Task{}
	for t := q.Run(c); ; {
		var x Task
		k, err := t.Next(&x)
		if err != nil {
			break
		}
		x.Key = k
		results = append(results, x)
	}
	mustEncode(w, results)
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
	c := appengine.NewContext(r)

	q := datastore.NewQuery("User").Order("Name")

	results := []User{}
	for t := q.Run(c); ; {
		var x User
		k, err := t.Next(&x)
		if err != nil {
			break
		}
		x.Key = k
		results = append(results, x)
	}

	mustEncode(w, results)
}

func adminListUnpaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	results := []LoggedTask{}

	q := datastore.NewQuery("LoggedTask").
		Filter("Paid = ", false).
		Order("Completed")

	for t := q.Run(c); ; {
		var x LoggedTask
		k, err := t.Next(&x)
		if err != nil {
			break
		}
		x.Key = k
		results = append(results, x)
	}

	mustEncode(w, results)
}

func adminMarkPaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	action := r.FormValue("action")

	r.ParseForm()
	keys := make([]*datastore.Key, 0, len(r.Form["pay"]))
	for _, s := range r.Form["pay"] {
		k, err := datastore.DecodeKey(s)
		if err != nil {
			panic(err)
		}
		keys = append(keys, k)
	}

	if action == "Mark Paid" {
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
	} else if action == "Delete" {
		err := datastore.DeleteMulti(c, keys)
		if err != nil {
			panic(err)
		}
	} else {
		panic("Unhandled action: " + action)
	}

	http.Redirect(w, r, "/admin/", 303)
}

func serveAdmin(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)

	c.Infof("Got admin request from %v", u)

	templates.ExecuteTemplate(w, "admin.html", u)
}

type mailTask struct {
	Amount int
	Tasks  []LoggedTask
}

func adminMailUnpaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	total := 0
	people := map[string]mailTask{}

	q := datastore.NewQuery("LoggedTask").
		Filter("Paid = ", false).
		Order("Completed")

	for t := q.Run(c); ; {
		var x LoggedTask
		_, err := t.Next(&x)
		if err != nil {
			break
		}
		total += x.Amount
		p := people[x.Who]
		p.Amount += x.Amount
		p.Tasks = append(p.Tasks, x)
		people[x.Who] = p
	}

	buf := &bytes.Buffer{}
	templates.ExecuteTemplate(buf, "mail.txt",
		struct {
			Total  int
			People map[string]mailTask
		}{total, people})

	msg := &mail.Message{
		Sender:  "Dustin Sallings <dsallings@gmail.com>",
		To:      []string{"dustin@sallings.org"},
		Subject: "Payment Report",
		Body:    string(buf.Bytes()),
	}
	if err := mail.Send(c, msg); err != nil {
		c.Errorf("Couldn't send email: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(204)
}
