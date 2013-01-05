package sallingshome

import (
	"fmt"
	"html/template"
	"net/http"

	"appengine"
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
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)

	c.Infof("Got a request from %v", u)

	templates.ExecuteTemplate(w, "index.html", nil)
}
