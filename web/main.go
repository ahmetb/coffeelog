package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	plus "github.com/google/google-api-go-client/plus/v1"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	cfg      *oauth2.Config
	hashKey  = []byte("very-secret")      // TODO extract to env
	blockKey = []byte("a-lot-secret-key") // TODO extract to env
	sc       = securecookie.New(hashKey, blockKey)
)

func main() {
	sc.SetSerializer(securecookie.JSONEncoder{})

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
	r := mux.NewRouter()
	r.HandleFunc("/", home).Methods(http.MethodGet)
	r.HandleFunc("/login", login).Methods(http.MethodGet)
	r.HandleFunc("/logout", logout).Methods(http.MethodGet)
	r.HandleFunc("/oauth2callback", oauth2Callback).Methods(http.MethodGet)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	srv := http.Server{
		Addr:    "127.0.0.1:8000",
		Handler: r,
	}
	log.Printf("listening at %s", srv.Addr)
	log.Fatal(errors.Wrap(srv.ListenAndServe(), "failed to listen/serve"))
}

func home(w http.ResponseWriter, r *http.Request) {
	type user struct {
		Name    string
		ID      string
		Picture string
	}

	var userData *user

	c, err := r.Cookie("oauth2_token")
	if err != http.ErrNoCookie {
		var token *oauth2.Token
		if err := sc.Decode("oauth2_token", c.Value, &token); err != nil {
			badRequest(w, errors.Wrap(err, "failed to decode cookie"))
			return
		}

		tokenSource := cfg.TokenSource(oauth2.NoContext, token)
		svc, err := plus.New(oauth2.NewClient(oauth2.NoContext, tokenSource))
		if err != nil {
			badRequest(w, errors.Wrap(err, "failed to construct g+ client"))
			return
		}
		me, err := plus.NewPeopleService(svc).Get("me").Do()
		if err != nil {
			badRequest(w, errors.Wrap(err, "failed to query user g+ profile"))
			return
		}
		userData = &user{ID: me.Id,
			Name:    me.DisplayName,
			Picture: me.Image.Url}
	}

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "home.html")))

	if err := tmpl.Execute(w, map[string]interface{}{"user": userData}); err != nil {
		log.Fatal(err)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	url := cfg.AuthCodeURL("todo_rand_state",
		oauth2.SetAuthURLParam("access_type", "offline"),
		oauth2.SetAuthURLParam("scope", "profile"))
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
}

func logout(w http.ResponseWriter, r *http.Request) {
	for _, c := range r.Cookies() {
		log.Printf("clearing cookie: %q", c.Name)
		c.Expires = time.Unix(1, 0)
		http.SetCookie(w, c)
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func oauth2Callback(w http.ResponseWriter, r *http.Request) {
	if state := r.URL.Query().Get("state"); state != "todo_rand_state" {
		badRequest(w, errors.New("wrong oauth2 state"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		badRequest(w, errors.New("missing oauth2 grant code"))
		return
	}

	tok, err := cfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		badRequest(w, errors.Wrap(err, "oauth2 token exchange failed"))
		return
	}

	tokenSource := cfg.TokenSource(oauth2.NoContext, tok)
	svc, err := plus.New(oauth2.NewClient(oauth2.NoContext, tokenSource))
	if err != nil {
		badRequest(w, errors.Wrap(err, "failed to construct g+ client"))
		return
	}
	me, err := plus.NewPeopleService(svc).Get("me").Do()
	if err != nil {
		badRequest(w, errors.Wrap(err, "failed to query user g+ profile"))
		return
	}

	// encrypt the cookie
	newToken, err := tokenSource.Token()
	if err != nil {
		badRequest(w, errors.Wrap(err, "failed to extract token from tokensource"))
		return
	}
	tokEncoded, err := sc.Encode("oauth2_token", newToken)
	if err != nil {
		badRequest(w, errors.Wrap(err, "failed to encode the token"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "oauth2_token",
		Path:  "/",
		Value: tokEncoded,
	})

	log.Printf("Authenticated as user id: %s", me.Id)
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func badRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, errors.Wrap(err, "bad request"))
}
