package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"io/ioutil"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var cfg *oauth2.Config

func main() {
	// read oauth2 config
	env := "GOOGLE_OAUTH2_CONFIG"
	if os.Getenv(env) == "" {
		panic(env + " is not set") // TODO fix
	}
	b, err := ioutil.ReadFile(os.Getenv(env))
	if err != nil {
		panic(err) // TODO fix
	}
	authConf, err := google.ConfigFromJSON(b)
	if err != nil {
		panic(err) // TODO fix
	}
	cfg = authConf

	// set up server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "We log coffee!")
	})
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/oauth2callback", oauth2Callback)
	srv := http.Server{
		Addr:    "127.0.0.1:8000",
		Handler: mux}
	log.Printf("listening at %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func login(w http.ResponseWriter, r *http.Request) {
	url := cfg.AuthCodeURL("todo_rand_state",
		// oauth2.SetAuthURLParam("redirect_uri", google.RedirectURL),
		oauth2.SetAuthURLParam("access_type", "offline"),
		oauth2.SetAuthURLParam("scope", "profile"))
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
}

func oauth2Callback(w http.ResponseWriter, r *http.Request) {
	if state := r.URL.Query().Get("state"); state != "todo_rand_state" {
		log.Fatal("wrong state") // TODO fix
	}

	code := r.URL.Query().Get("code") // TODO check
	fmt.Println(code)

	tok, err := cfg.Exchange(nil, code)
	if err != nil {
		log.Fatal(err) // TODO fix
	}
	fmt.Printf("%v\n", tok)
}
