package main

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"

	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	plus "github.com/google/google-api-go-client/plus/v1"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	cfg      *oauth2.Config
	hashKey  = []byte("very-secret")      // TODO extract to env
	blockKey = []byte("a-lot-secret-key") // TODO extract to env
	sc       = securecookie.New(hashKey, blockKey)

	userDirectoryBackend = "127.0.0.1:8001" // TODO use service discovery
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
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
	r.HandleFunc("/", logHandler(home)).Methods(http.MethodGet)
	r.HandleFunc("/login", logHandler(login)).Methods(http.MethodGet)
	r.HandleFunc("/logout", logHandler(logout)).Methods(http.MethodGet)
	r.HandleFunc("/oauth2callback", logHandler(oauth2Callback)).Methods(http.MethodGet)
	r.PathPrefix("/static/").Handler(logHandler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP))
	srv := http.Server{
		Addr:    "127.0.0.1:8000", // TODO make configurable
		Handler: r}
	log.WithField("addr", srv.Addr).Info("starting to listen on http")
	log.Fatal(errors.Wrap(srv.ListenAndServe(), "failed to listen/serve"))
}

func logHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e := log.WithFields(log.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		e.Debug("request accepted")
		start := time.Now()
		defer func() {
			e.WithFields(log.Fields{
				"elapsed": time.Now().Sub(start),
			}).Debug("request completed")
		}()
		h(w, r)
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	var user *pb.User
	if c, err := r.Cookie("user"); err == http.ErrNoCookie {
		log.Debug("no auth cookie found")
	} else {
		log.Debug("auth cookie found")
		var userID string
		if err := sc.Decode("user", c.Value, &userID); err != nil {
			badRequest(w, errors.Wrap(err, "failed to decode cookie"))
			return
		}

		// reach out to the user directory to save/retrieve user
		cc, err := grpc.Dial(userDirectoryBackend, grpc.WithInsecure())
		if err != nil {
			serverError(w, errors.Wrap(err, "failed to communicate the backend"))
			return
		}
		defer cc.Close()
		userResp, err := pb.NewUserDirectoryClient(cc).GetUser(context.TODO(),
			&pb.UserRequest{ID: userID})
		if err != nil {
			serverError(w, errors.Wrap(err, "failed to retrieve the user"))
			return
		} else if !userResp.Found {
			badRequest(w, errors.New("unrecognized user"))
			return
		}
		user = userResp.User
	}

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "home.html")))

	if err := tmpl.Execute(w, map[string]interface{}{"user": user}); err != nil {
		log.Fatal(err)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	url := cfg.AuthCodeURL("todo_rand_state",
		oauth2.SetAuthURLParam("access_type", "offline"),
		oauth2.SetAuthURLParam("scope", "profile"))
	log.Debug("redirecting user to oauth2 consent page")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
}

func logout(w http.ResponseWriter, r *http.Request) {
	log.Debug("logout requested")
	for _, c := range r.Cookies() {
		c.Expires = time.Unix(1, 0)
		http.SetCookie(w, c)
		log.WithField("key", c.Name).Debug("cleared cookie")
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
		serverError(w, errors.Wrap(err, "oauth2 token exchange failed"))
		return
	}

	svc, err := plus.New(oauth2.NewClient(oauth2.NoContext, cfg.TokenSource(oauth2.NoContext, tok)))
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to construct g+ client"))
		return
	}
	me, err := plus.NewPeopleService(svc).Get("me").Do()
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to query user g+ profile"))
		return
	}
	log.WithField("google.id", me.Id).Debug("retrieved google user")

	// reach out to the user directory to save/retrieve user
	cc, err := grpc.Dial(userDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to communicate the backend"))
		return
	}
	defer cc.Close()
	user, err := pb.NewUserDirectoryClient(cc).AuthorizeGoogle(context.TODO(),
		&pb.GoogleUser{
			ID: me.Id,
			// Email:       me.Emails[0].Value,
			DisplayName: me.DisplayName,
			PictureURL:  me.Image.Url,
		})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to log in the user"))
		return
	}
	log.WithField("id", user.ID).Info("authenticated user with google")

	// save the user id to cookies
	// TODO implement as sessions
	co, err := sc.Encode("user", user.ID)
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to encode the token"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "user",
		Path:  "/",
		Value: co,
	})

	log.WithField("user.id", me.Id).Info("authenticated user")
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func badRequest(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusBadRequest, "bad request", err)
}

func serverError(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusInternalServerError, "server error", err)
}

func errorCode(w http.ResponseWriter, code int, msg string, err error) {
	log.WithField("http.status", code).WithField("error", err).Warn(msg)
	w.WriteHeader(code)
	fmt.Fprint(w, errors.Wrap(err, msg))
}
