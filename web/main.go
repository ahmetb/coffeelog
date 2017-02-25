package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "github.com/ahmetalpbalkan/coffeelog/coffeelog"
	"github.com/golang/protobuf/ptypes"
	plus "github.com/google/google-api-go-client/plus/v1"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	logrus "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
)

var (
	cfg      *oauth2.Config
	hashKey  = []byte("very-secret")      // TODO extract to env
	blockKey = []byte("a-lot-secret-key") // TODO extract to env
	sc       = securecookie.New(hashKey, blockKey)

	userDirectoryBackend   = "127.0.0.1:8001" // TODO use service discovery
	coffeeDirectoryBackend = "127.0.0.1:8002" // TODO use service discovery
)

var log *logrus.Entry

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log = logrus.WithField("service", "web")
	sc.SetSerializer(securecookie.JSONEncoder{})

	// read oauth2 config
	env := "GOOGLE_OAUTH2_CONFIG"
	if os.Getenv(env) == "" {
		log.Fatalf("%s is not set", env)
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
	r.PathPrefix("/static/").HandlerFunc(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)
	r.HandleFunc("/", logHandler(home)).Methods(http.MethodGet)
	r.HandleFunc("/login", logHandler(login)).Methods(http.MethodGet)
	r.HandleFunc("/logout", logHandler(logout)).Methods(http.MethodGet)
	r.HandleFunc("/oauth2callback", logHandler(oauth2Callback)).Methods(http.MethodGet)
	r.HandleFunc("/coffee", logHandler(logCoffee)).Methods(http.MethodPost)
	r.HandleFunc("/a/{id:[0-9]+}", logHandler(activity)).Methods(http.MethodGet)
	r.HandleFunc("/u/{id:[0-9]+}", logHandler(userProfile)).Methods(http.MethodGet)
	r.HandleFunc("/autocomplete/roaster", logHandler(autocompleteRoaster)).Methods(http.MethodGet)
	srv := http.Server{
		Addr:    "127.0.0.1:8000", // TODO make configurable
		Handler: r}
	log.WithField("addr", srv.Addr).Info("starting to listen on http")
	log.Fatal(errors.Wrap(srv.ListenAndServe(), "failed to listen/serve"))
}

func logHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e := log.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		e.Debug("request accepted")
		start := time.Now()
		defer func() {
			e.WithFields(logrus.Fields{
				"elapsed": time.Now().Sub(start),
			}).Debug("request completed")
		}()
		h(w, r)
	}
}

type httpErrorWriter func(http.ResponseWriter, error)

func getUser(id string) (*pb.UserResponse, error) {
	cc, err := grpc.Dial(userDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "failed to communicate the backend")
	}
	defer cc.Close()

	userResp, err := pb.NewUserDirectoryClient(cc).GetUser(context.TODO(),
		&pb.UserRequest{ID: id})
	return userResp, err
}

func authUser(r *http.Request) (user *pb.User, errFunc httpErrorWriter, err error) {
	c, err := r.Cookie("user")
	if err == http.ErrNoCookie {
		return nil, nil, nil
	}
	log.Debug("auth cookie found")
	var userID string
	if err := sc.Decode("user", c.Value, &userID); err != nil {
		return nil, badRequest, errors.Wrap(err, "failed to decode cookie")
	}

	userResp, err := getUser(userID)
	if err != nil {
		return nil, serverError, errors.Wrap(err, "failed to look up the user")
	} else if !userResp.GetFound() {
		return nil, badRequest, errors.New("unrecognized user")
	}
	return userResp.GetUser(), nil, nil
}

func home(w http.ResponseWriter, r *http.Request) {
	user, errF, err := authUser(r)
	if err != nil {
		errF(w, err)
		return
	}

	log.WithField("logged_in", user != nil).Debug("serving home page")
	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "home.html")))

	if err := tmpl.Execute(w, map[string]interface{}{
		"me":              user,
		"drinks":          drinks,
		"methods":         methods,
		"authenticated":   user != nil,
		"originCountries": originCountries}); err != nil {
		log.Fatal(err)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	cfg.Scopes = []string{"profile", "email"}
	url := cfg.AuthCodeURL("todo_rand_state",
		oauth2.SetAuthURLParam("access_type", "offline"))
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
			ID:          me.Id,
			Email:       me.Emails[0].Value,
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

func unauthorized(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusUnauthorized, "unauthorized", err)
}

func badRequest(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusBadRequest, "bad request", err)
}

func serverError(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusInternalServerError, "server error", err)
}

func logCoffee(w http.ResponseWriter, r *http.Request) {
	user, errF, err := authUser(r)
	if err != nil {
		errF(w, err)
		return
	}
	if user == nil {
		badRequest(w, errors.New("required user to log in to post activity"))
		return
	}

	if err := r.ParseMultipartForm(16 * 1024 * 1024); err != nil { // max 16 mb memory
		badRequest(w, errors.Wrap(err, "failed to parse request"))
		return
	}

	// parse picture
	var picture []byte

	f, h, err := r.FormFile("picture")
	if err == http.ErrMissingFile {
		log.Debug("no file was uploaded")
	} else if err != nil {
		badRequest(w, errors.Wrap(err, "failed to parse form file"))
		return
	} else {
		defer f.Close()

		ct := h.Header.Get("Content-Type")
		entry := log.WithField("content-type", ct).WithField("name", h.Filename)
		entry.Debug("upload received")
		if !strings.HasPrefix(ct, "image/") {
			badRequest(w, errors.New("uploaded file is not a photo"))
			return
		}

		picture, err = ioutil.ReadAll(f)
		if err != nil {
			serverError(w, errors.Wrap(err, "failed to read file"))
			return
		}
		entry.WithField("size", len(picture)).Debug("uploaded file is read")
	}

	var (
		drink         = r.FormValue("drink")
		homebrew      = r.FormValue("homebrew") == "on"
		amount        = r.FormValue("amount")
		amountUnitStr = r.FormValue("amount_unit")
		roasterName   = r.FormValue("roaster")
		origin        = r.FormValue("origin")
		method        = r.FormValue("method")
		notes         = r.FormValue("notes")
	)

	amountN, _ := strconv.ParseInt(amount, 10, 32)
	var amountU pb.Activity_DrinkAmount_CaffeineUnit
	switch amountUnitStr {
	case "oz":
		amountU = pb.Activity_DrinkAmount_OUNCES
	case "shots":
		amountU = pb.Activity_DrinkAmount_SHOTS
	default:
		amountU = pb.Activity_DrinkAmount_UNSPECIFIED
	}

	log.WithFields(logrus.Fields{
		"user":          user.GetID(),
		"drink":         drink,
		"homebrew":      homebrew,
		"roasterName":   roasterName,
		"origin":        origin,
		"method":        method,
		"picture_bytes": len(picture),
		"amount":        fmt.Sprintf("%d %s", amountN, amountU),
		"notes":         notes,
	}).Info("received form")

	cc, err := grpc.Dial(coffeeDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to communicate the backend"))
		return
	}
	defer cc.Close()

	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		serverError(w, errors.Wrap(err, "cannot convert timestamp to proto"))
	}
	resp, err := pb.NewActivityDirectoryClient(cc).PostActivity(context.TODO(), &pb.PostActivityRequest{
		UserID: user.GetID(),
		Date:   ts,
		Amount: &pb.Activity_DrinkAmount{
			N:    int32(amountN),
			Unit: amountU,
		},
		Drink:       drink,
		Origin:      origin,
		RoasterName: roasterName,
		Homebrew:    homebrew,
		Method:      method,
		Picture: &pb.PostActivityRequest_File{
			Data:        picture,
			ContentType: h.Header.Get("Content-Type"),
			Filename:    h.Filename,
		},
		Notes: notes,
	})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to save activity"))
		return
	}
	log.WithField("id", resp.GetID()).Info("activity posted")

	w.Header().Set("Location", fmt.Sprintf("/a/%d", resp.GetID()))
	w.WriteHeader(http.StatusFound)
}

func autocompleteRoaster(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("data")
	if len(q) > 100 {
		badRequest(w, errors.New("request too long"))
		return
	}

	type result struct {
		Value string `json:"value"`
	}

	cc, err := grpc.Dial(coffeeDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to communicate the backend"))
		return
	}
	defer cc.Close()

	resp, err := pb.NewRoasterDirectoryClient(cc).ListRoasters(context.TODO(), new(pb.RoastersRequest))
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to query the roasters"))
		return
	}

	var v []result
	q = strings.ToLower(q)
	for _, r := range resp.GetResults() {
		if strings.Contains(strings.ToLower(r.GetName()), q) {
			v = append(v, result{r.GetName()})
		}
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		serverError(w, errors.Wrap(err, "failed to encode the response"))
		return
	}
	logrus.WithFields(logrus.Fields{
		"q":       q,
		"matches": len(v)}).Debug("autocomplete response")
}

func activity(w http.ResponseWriter, r *http.Request) {
	idS := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		badRequest(w, errors.Wrap(err, "bad activity id"))
		return
	}

	e := log.WithField("id", id)
	e.Debug("activity requested")

	// TODO eliminate duplication
	cc, err := grpc.Dial(coffeeDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to communicate the backend"))
		return
	}
	defer cc.Close()
	ar, err := pb.NewActivityDirectoryClient(cc).GetActivity(context.TODO(), &pb.ActivityRequest{ID: id})
	if err != nil {
		serverError(w, errors.Wrap(err, "cannot get activity"))
		return
	}
	e.WithField("user.id", ar.GetUser().GetID()).Debug("retrieved activity")

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "activity.html")))

	if err := tmpl.Execute(w, map[string]interface{}{
		"activity": ar}); err != nil {
		log.Fatal(err)
	}
}

func userProfile(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	userResp, err := getUser(userID)
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to look up the user"))
		return
	} else if !userResp.GetFound() {
		errorCode(w, http.StatusNotFound, "not found", errors.New("user not found"))
		return
	}

	// TODO eliminate duplicatio
	cc, err := grpc.Dial(coffeeDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to communicate the backend"))
		return
	}
	defer cc.Close()

	ar, err := pb.NewActivityDirectoryClient(cc).GetUserActivities(context.TODO(),
		&pb.UserActivitiesRequest{UserID: userID})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to query activities"))
		return
	}

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "profile.html")))
	if err := tmpl.Execute(w, map[string]interface{}{
		"user":       userResp.GetUser(),
		"activities": ar.GetActivities()}); err != nil {
		log.Fatal(err)
	}
}

func errorCode(w http.ResponseWriter, code int, msg string, err error) {
	log.WithField("http.status", code).WithField("error", err).Warn(msg)
	w.WriteHeader(code)
	fmt.Fprint(w, errors.Wrap(err, msg))
}

var (
	originCountries = map[string][]string{
		"Africa":   {"Kenya", "Ethiophia", "Nigeria", "Burundi", "Rwanda"},
		"Americas": {"Colombia", "Venezuela", "Brazil", "Peru", "Cuba", "Ecuador", "Honduras", "Mexico", "Costa Rica"},
		"Asia":     {"Indonesia", "India", "Vietnam"},
	}

	drinks = map[string]bool{
		// espresso-based:
		"Latte":       true,
		"Mocha":       true,
		"Breve":       true,
		"Espresso":    true,
		"Macchiato":   true,
		"Cortado":     true,
		"Americano":   true,
		"Cappuccino":  true,
		"Flat white":  true,
		"Café Cubano": true,
		"Affogato":    true,
		"Ristretto":   true,
		"Corretto":    true,
		// non-espresso based:
		"Brewed coffee": false,
		"Cold brew":     false,
		"Iced coffee":   false,
		"Decaf coffee":  false,
		"Café au lait":  false, // ?
	}

	// TODO fix these with proper attribution to designers.
	methods = []struct{ Name, Icon string }{
		{"Espresso", "espresso-machine.png"},
		{"Chemex", "chemex.png"},
		{"Aeropress", "aeropress.png"},
		{"Hario V60", "v60.png"},
		{"French press", "french-press.png"},
		{"Dripper", "dripper.png"},
		{"Kyoto Dripper", "kyoto.png"},
		{"Moka Pot", "moka.png"},
		{"Turkish coffee", "turkish.png"},
	}
)
