package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "github.com/ahmetb/coffeelog/coffeelog"
	"github.com/ahmetb/coffeelog/version"
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

type server struct {
	cfg         *oauth2.Config
	userSvc     pb.UserDirectoryClient
	roasterSvc  pb.RoasterDirectoryClient
	activitySvc pb.ActivityDirectoryClient
}

var (
	projectID              = flag.String("google-project-id", "", "google cloud project id")
	addr                   = flag.String("addr", ":8000", "[host]:port to listen")
	oauthConfig            = flag.String("google-oauth2-config", "", "path to oauth2 config json")
	userDirectoryBackend   = flag.String("user-directory-addr", "", "address of user directory backend")
	coffeeDirectoryBackend = flag.String("coffee-directory-addr", "", "address of coffee directory backend")

	hashKey  = []byte("very-secret")      // TODO extract to env
	blockKey = []byte("a-lot-secret-key") // TODO extract to env
	sc       = securecookie.New(hashKey, blockKey)
)

var log *logrus.Entry

func main() {
	flag.Parse()
	host, err := os.Hostname()
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot get hostname"))
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	log = logrus.WithFields(logrus.Fields{
		"service": "web",
		"host":    host,
		"v":       version.Version(),
	})
	sc.SetSerializer(securecookie.JSONEncoder{})

	if *userDirectoryBackend == "" {
		log.Fatal("user directory address flag not specified")
	}
	if *coffeeDirectoryBackend == "" {
		log.Fatal("user directory address flag not specified")
	}
	if *oauthConfig == "" {
		log.Fatal("google oauth2 config flag not specified")
	}

	b, err := ioutil.ReadFile(*oauthConfig)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to parse config file"))
	}
	authConf, err := google.ConfigFromJSON(b)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to parse config file"))
	}

	userSvcConn, err := grpc.Dial(*userDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot connect user service"))
	}
	defer func() {
		log.Info("closing connection to user directory")
		userSvcConn.Close()
	}()
	coffeeSvcConn, err := grpc.Dial(*coffeeDirectoryBackend, grpc.WithInsecure())
	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot connect coffee service"))
	}
	defer func() {
		log.Info("closing connection to user directory")
		coffeeSvcConn.Close()
	}()

	s := &server{
		cfg:         authConf,
		userSvc:     pb.NewUserDirectoryClient(userSvcConn),
		activitySvc: pb.NewActivityDirectoryClient(coffeeSvcConn),
		roasterSvc:  pb.NewRoasterDirectoryClient(coffeeSvcConn),
	}

	// set up server
	r := mux.NewRouter()
	r.PathPrefix("/static/").HandlerFunc(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)
	r.HandleFunc("/", logHandler(s.home)).Methods(http.MethodGet)
	r.HandleFunc("/login", logHandler(s.login)).Methods(http.MethodGet)
	r.HandleFunc("/logout", logHandler(s.logout)).Methods(http.MethodGet)
	r.HandleFunc("/oauth2callback", logHandler(s.oauth2Callback)).Methods(http.MethodGet)
	r.HandleFunc("/coffee", logHandler(s.logCoffee)).Methods(http.MethodPost)
	r.HandleFunc("/a/{id:[0-9]+}", logHandler(s.activity)).Methods(http.MethodGet)
	r.HandleFunc("/u/{id:[0-9]+}", logHandler(s.userProfile)).Methods(http.MethodGet)
	r.HandleFunc("/autocomplete/roaster", logHandler(s.autocompleteRoaster)).Methods(http.MethodGet)

	srv := http.Server{
		Addr:    *addr, // TODO make configurable
		Handler: r}
	log.WithField("addr", *addr).Info("starting to listen on http")
	log.Fatal(errors.Wrap(srv.ListenAndServe(), "failed to listen/serve"))
}

// logHandler wraps the HTTP handler with logging.
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

func (s *server) getUser(id string) (*pb.UserResponse, error) {
	userResp, err := s.userSvc.GetUser(context.TODO(),
		&pb.UserRequest{ID: id})
	return userResp, err
}

func (s *server) authUser(r *http.Request) (user *pb.User, errFunc httpErrorWriter, err error) {
	c, err := r.Cookie("user")
	if err == http.ErrNoCookie {
		return nil, nil, nil
	}
	log.Debug("auth cookie found")
	var userID string
	if err := sc.Decode("user", c.Value, &userID); err != nil {
		return nil, badRequest, errors.Wrap(err, "failed to decode cookie")
	}

	userResp, err := s.getUser(userID)
	if err != nil {
		return nil, serverError, errors.Wrap(err, "failed to look up the user")
	} else if !userResp.GetFound() {
		return nil, badRequest, errors.New("unrecognized user")
	}
	return userResp.GetUser(), nil, nil
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	user, errF, err := s.authUser(r)
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
		"methods":         methodsList,
		"authenticated":   user != nil,
		"originCountries": originCountries}); err != nil {
		log.Fatal(err)
	}
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	s.cfg.RedirectURL = "http://" + r.Host + "/oauth2callback" // TODO this is hacky
	s.cfg.Scopes = []string{"profile", "email"}
	url := s.cfg.AuthCodeURL("todo_rand_state",
		oauth2.SetAuthURLParam("access_type", "offline"))
	log.Debug("redirecting user to oauth2 consent page")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	log.Debug("logout requested")
	for _, c := range r.Cookies() {
		c.Expires = time.Unix(1, 0)
		http.SetCookie(w, c)
		log.WithField("key", c.Name).Debug("cleared cookie")
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func (s *server) oauth2Callback(w http.ResponseWriter, r *http.Request) {
	if state := r.URL.Query().Get("state"); state != "todo_rand_state" {
		badRequest(w, errors.New("wrong oauth2 state"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		badRequest(w, errors.New("missing oauth2 grant code"))
		return
	}

	tok, err := s.cfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		serverError(w, errors.Wrap(err, "oauth2 token exchange failed"))
		return
	}

	svc, err := plus.New(oauth2.NewClient(oauth2.NoContext, s.cfg.TokenSource(oauth2.NoContext, tok)))
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

	user, err := s.userSvc.AuthorizeGoogle(context.TODO(),
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

func (s *server) logCoffee(w http.ResponseWriter, r *http.Request) {
	user, errF, err := s.authUser(r)
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
		method        = r.FormValue("brew-method")
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

	var pFile *pb.PostActivityRequest_File
	if len(picture) > 0 {
		pFile = &pb.PostActivityRequest_File{
			Data:        picture,
			ContentType: h.Header.Get("Content-Type"),
			Filename:    h.Filename,
		}
	}

	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		serverError(w, errors.Wrap(err, "cannot convert timestamp to proto"))
	}
	resp, err := s.activitySvc.PostActivity(context.TODO(), &pb.PostActivityRequest{
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
		Picture:     pFile,
		Notes:       notes,
	})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to save activity"))
		return
	}
	log.WithField("id", resp.GetID()).Info("activity posted")

	w.Header().Set("Location", fmt.Sprintf("/u/%s", user.GetID()))
	w.WriteHeader(http.StatusFound)
}

func (s *server) autocompleteRoaster(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("data")
	if len(q) > 100 {
		badRequest(w, errors.New("request too long"))
		return
	}

	type result struct {
		Value string `json:"value"`
	}

	resp, err := s.roasterSvc.ListRoasters(context.TODO(), new(pb.RoastersRequest))
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
	log.WithFields(logrus.Fields{
		"q":       q,
		"matches": len(v)}).Debug("autocomplete response")
}

func (s *server) activity(w http.ResponseWriter, r *http.Request) {
	user, ef, err := s.authUser(r)
	if err != nil {
		ef(w, err)
		return
	}

	idS := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		badRequest(w, errors.Wrap(err, "bad activity id"))
		return
	}

	e := log.WithField("id", id)
	e.Debug("activity requested")

	ar, err := s.activitySvc.GetActivity(context.TODO(), &pb.ActivityRequest{ID: id})
	if err != nil {
		serverError(w, errors.Wrap(err, "cannot get activity"))
		return
	}
	e.WithField("user.id", ar.GetUser().GetID()).Debug("retrieved activity")

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "activity.html")))

	if err := tmpl.Execute(w, map[string]interface{}{
		"activity": ar,
		"me":       user}); err != nil {
		log.Fatal(err)
	}
}

func (s *server) userProfile(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	me, ef, err := s.authUser(r)
	if err != nil {
		ef(w, err)
		return
	}

	userResp, err := s.getUser(userID)
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to look up the user"))
		return
	} else if !userResp.GetFound() {
		errorCode(w, http.StatusNotFound, "not found", errors.New("user not found"))
		return
	}

	ar, err := s.activitySvc.GetUserActivities(context.TODO(),
		&pb.UserActivitiesRequest{UserID: userID})
	if err != nil {
		serverError(w, errors.Wrap(err, "failed to query activities"))
		return
	}

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("static", "template", "layout.html"),
		filepath.Join("static", "template", "profile.html")))
	if err := tmpl.Execute(w, map[string]interface{}{
		"me":         me,
		"user":       userResp.GetUser(),
		"activities": ar.GetActivities(),
		"methods":    methodIcons,
		"drinks":     drinks}); err != nil {
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
		"Latte":          true,
		"Mocha":          true,
		"Breve":          true,
		"Espresso":       true,
		"Macchiato":      true,
		"Cortado":        true,
		"Americano":      true,
		"Cappuccino":     true,
		"Flat white":     true,
		"Café Cubano":    true,
		"Affogato":       true,
		"Ristretto":      true,
		"Corretto":       true,
		"Turkish coffee": true,
		// non-espresso based:
		"Coffee":       false,
		"Cold brew":    false,
		"Iced coffee":  false,
		"Decaf coffee": false,
		"Café au lait": false, // ?
	}

	// TODO fix these with proper attribution to designers.
	methodIcons = map[string]string{
		"Espresso":       "espresso-machine.png",
		"Chemex":         "chemex.png",
		"Aeropress":      "aeropress.png",
		"Hario V60":      "v60.png",
		"French press":   "french-press.png",
		"Dripper":        "dripper.png",
		"Kyoto Dripper":  "kyoto.png",
		"Moka Pot":       "moka.png",
		"Turkish coffee": "turkish.png",
	}
	methodsList = []struct{ Name, Icon string }{
		{"Espresso", methodIcons["Espresso"]},
		{"Chemex", methodIcons["Chemex"]},
		{"Aeropress", methodIcons["Aeropress"]},
		{"Hario V60", methodIcons["Hario V60"]},
		{"French press", methodIcons["French press"]},
		{"Dripper", methodIcons["Dripper"]},
		{"Kyoto Dripper", methodIcons["Kyoto Dripper"]},
		{"Moka Pot", methodIcons["Moka Pot"]},
		{"Turkish coffee", methodIcons["Turkish coffee"]},
	}
)

func unauthorized(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusUnauthorized, "unauthorized", err)
}

func badRequest(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusBadRequest, "bad request", err)
}

func serverError(w http.ResponseWriter, err error) {
	errorCode(w, http.StatusInternalServerError, "server error", err)
}
