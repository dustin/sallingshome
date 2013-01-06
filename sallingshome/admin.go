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
	http.HandleFunc("/admin/tasks/", adminListTasks)
	http.HandleFunc("/admin/", serveAdmin)
}

func iterateTasks(c appengine.Context) chan Task {
	ch := make(chan Task)

	q := datastore.NewQuery("Task").Order("Name")

	go func() {
		defer close(ch)
		for t := q.Run(c); ; {
			var x Task
			k, err := t.Next(&x)
			if err == datastore.Done {
				break
			}
			x.Key = k
			ch <- x
		}
	}()

	return ch
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
		Name:     r.FormValue("name"),
		Assignee: r.FormValue("assignee"),
		RType:    r.FormValue("rtype"),
		Period:   asInt(r.FormValue("period")),
		Value:    asInt(r.FormValue("value")),
		Next:     time.Now().UTC(),
	}

	k, err := datastore.Put(c,
		datastore.NewIncompleteKey(c, "Task", nil), &task)
	if err != nil {
		c.Warningf("Error storing stats item:  %v", err)
		panic(err)
	}
	c.Infof("Stored new thing with key %v", k)

	http.Redirect(w, r, "/admin/tasks/", 307)
}

func adminListTasks(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "admin_tasks.html",
		map[string]interface{}{
			"results":   iterateTasks(appengine.NewContext(r)),
			"assignees": []string{"Jennalynn", "Sidney"},
		})
}

func serveAdmin(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)

	c.Infof("Got admin request from %v", u)

	templates.ExecuteTemplate(w, "admin.html", u)
}
