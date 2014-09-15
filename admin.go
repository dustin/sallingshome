package sallingshome

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"reflect"

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
	http.HandleFunc("/admin/cron/auto/", adminAutoPay)

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
	execTemplate(appengine.NewContext(r), w, "admin.html", nil)
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
	task.Disabled = mightParseBool(r.FormValue("disabled"))
	task.Automatic = mightParseBool(r.FormValue("automatic"))
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

func fillKeyQuery(c appengine.Context, q *datastore.Query, results interface{}) error {
	keys, err := q.GetAll(c, results)
	if err == nil {
		rslice := reflect.ValueOf(results).Elem()
		for i := range keys {
			if k, ok := rslice.Index(i).Interface().(Keyable); ok {
				k.setKey(keys[i])
			} else if k, ok := rslice.Index(i).Addr().Interface().(Keyable); ok {
				k.setKey(keys[i])
			} else {
				c.Infof("Warning: %v is not Keyable", rslice.Index(i).Interface())
			}
		}
	} else {
		c.Errorf("Error executing query: %v", err)
	}
	return err
}

func adminListTasks(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("Task").Order("Disabled").Order("Name")

	results := []Task{}
	fillKeyQuery(c, q, &results)
	mustEncode(w, results)
}

func mightParseBool(s string) bool {
	switch strings.ToLower(s) {
	case "on", "true", "1", "y", "t", "yes":
		return true
	}
	return false
}

func adminNewTask(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

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
		Automatic:   mightParseBool(r.FormValue("automatic")),
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
	fillKeyQuery(c, q, &results)
	mustEncode(w, results)
}

func adminListUnpaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q := datastore.NewQuery("LoggedTask").
		Filter("Paid = ", false).
		Order("Completed")

	results := []LoggedTask{}
	fillKeyQuery(c, q, &results)
	mustEncode(w, results)
}

func adminMarkPaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	action := r.FormValue("action")

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

	execTemplate(c, w, "admin.html", u)
}

type mailUserTask struct {
	Task     LoggedTask
	Quantity int
	Subtotal int
}

type mailTask struct {
	Amount int
	Tasks  map[string]*mailUserTask
}

func adminMailUnpaid(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	total := 0
	people := map[string]mailTask{}

	q := datastore.NewQuery("LoggedTask").
		Filter("Paid = ", false).
		Order("Name")

	for t := q.Run(c); ; {
		var x LoggedTask
		_, err := t.Next(&x)
		if err != nil {
			break
		}
		total += x.Amount
		p := people[x.Who]
		p.Amount += x.Amount

		mut, ok := p.Tasks[x.Name]
		if !ok {
			mut = &mailUserTask{Task: x}
			if p.Tasks == nil {
				p.Tasks = map[string]*mailUserTask{}
			}
			p.Tasks[x.Name] = mut
		}

		mut.Quantity++
		mut.Subtotal += x.Amount
		people[x.Who] = p
	}

	buf := &bytes.Buffer{}
	tw := tabwriter.NewWriter(buf, 0, 2, 1, ' ', 0)
	err := execTemplate(c, tw, "mail.txt",
		struct {
			Total  int
			People map[string]mailTask
		}{total, people})
	if err != nil {
		c.Errorf("Template error: %v", err)
		return
	}
	tw.Flush()

	msg := &mail.Message{
		Sender:  "Dustin Sallings <dsallings@gmail.com>",
		To:      []string{"dustin@sallings.org"},
		Subject: "Payment Report",
		Body:    string(buf.Bytes()),
	}
	c.Infof("Sending:\n%s\n", msg.Body)
	if err := mail.Send(c, msg); err != nil {
		c.Errorf("Couldn't send email: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(204)
}

func adminAutoPay(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	c := appengine.NewContext(r)
	q := datastore.NewQuery("Task").
		Filter("Disabled =", false).
		Filter("Automatic = ", true).
		Filter("Next < ", now)

	tasks := []*Task{}
	if err := fillKeyQuery(c, q, &tasks); err != nil {
		c.Warningf("Error finding automatic things: %v", err)
		w.WriteHeader(500)
		return
	}

	if len(tasks) == 0 {
		c.Infof("No automatic tasks.")
		w.WriteHeader(204)
		return
	}

	storeKeys := make([]*datastore.Key, 0, 2*len(tasks))
	vals := []interface{}{}

	for i := range tasks {
		c.Infof("Recording automatic task %q for %v at %s", tasks[i].Name,
			tasks[i].Assignee, moneyFmt(tasks[i].Value))

		su, err := getUserByEmail(c, tasks[i].Assignee)
		if err != nil {
			c.Warningf("Failed to look up user %v: %v", tasks[i].Assignee, err)
			w.WriteHeader(500)
			return
		}

		tasks[i].updateTime()
		tasks[i].Prev = now
		storeKeys = append(storeKeys, tasks[i].Key)
		vals = append(vals, tasks[i])

		storeKeys = append(storeKeys,
			datastore.NewIncompleteKey(c, "LoggedTask", nil))
		vals = append(vals, &LoggedTask{
			Task:      tasks[i].Key,
			User:      su.Key,
			Completed: now,
			Who:       su.Name,
			Name:      tasks[i].Name,
			Amount:    tasks[i].Value,
		})
	}

	if _, err := datastore.PutMulti(c, storeKeys, vals); err != nil {
		panic(err)
	}

	w.WriteHeader(204)
}
