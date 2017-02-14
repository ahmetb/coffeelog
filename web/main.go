package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	plus "github.com/google/google-api-go-client/plus/v1"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var cfg *oauth2.Config

func main() {
	// read oauth2 config
	env := "GOOGLE_OAUTH2_CONFIG"
	if os.Getenv(env) == "" {
		panic(errors.New(env + " is not set"))
	}
	b, err := ioutil.ReadFile(os.Getenv(env))
	if err != nil {
		panic(errors.Wrap(err, "failed to parse config file"))
	}
	authConf, err := google.ConfigFromJSON(b)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse config file"))
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
		Handler: mux,
	}
	log.Printf("listening at %s", srv.Addr)
	log.Fatal(errors.Wrap(srv.ListenAndServe(), "failed to listen/serve"))
}

func login(w http.ResponseWriter, r *http.Request) {
	url := cfg.AuthCodeURL("todo_rand_state",
		oauth2.SetAuthURLParam("access_type", "offline"),
		oauth2.SetAuthURLParam("scope", "profile"))
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
}

func oauth2Callback(w http.ResponseWriter, r *http.Request) {
	if state := r.URL.Query().Get("state"); state != "todo_rand_state" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "wrong state")
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "missing oauth2 grant code")
		return
	}

	tok, err := cfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, errors.Wrap(err, "oauth2 token exchange failed"))
		return
	}
	svc, err := plus.New(cfg.Client(oauth2.NoContext, tok))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, errors.Wrap(err, "failed to construct g+ client"))
		return
	}
	me, err := plus.NewPeopleService(svc).Get("me").Do()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, errors.Wrap(err, "failed to query user g+ profile"))
		return
	}
	log.Printf("Logged in user id: %s", me.Id)
	fmt.Fprintf(w, "Hello "+me.DisplayName)
}
