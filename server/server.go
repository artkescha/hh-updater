package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/artkescha/hh-updater/crypto"
	"net/http"
	"regexp"
	"time"

	"github.com/artkescha/hh-updater/config"
	"github.com/artkescha/hh-updater/hhclient"
	"github.com/boltdb/bolt"
	"github.com/gorilla/context"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const UserCtxKey = "ctxUser"

// Endpoint is HH's OAuth 2.0 endpoint.
var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://hh.ru/oauth/authorize",
	TokenURL: "https://hh.ru/oauth/token",
}

var (
	ErrEmptyResumeList = errors.New("Empty resume list")

	UsersBucket     = []byte("usersv1")
	UsersKey        = []byte("list")
	MailLoginRegExp = regexp.MustCompile(`^([^@]*)`)
)

type Server struct {
	c               *config.Config
	userList        map[string]*User
	userListChanged bool
	oAuthConf       *oauth2.Config
	db              *bolt.DB
}

type User struct {
	ID          string        `json:"id"`
	Email       string        `json:"email"`
	Token       *oauth2.Token `json:"token"`
	UpdatedAt   time.Time     `json:"updated_at"`
	UpdateCount int           `json:"update_count"`
}

type SafeUser struct {
	ID string `json:"id"`
}

func NewServer(config *config.Config) *Server {
	return &Server{
		c:        config,
		userList: map[string]*User{},
		oAuthConf: &oauth2.Config{
			Endpoint:     Endpoint,
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
		},
	}
}

func (s *Server) Init() error {
	db, err := bolt.Open(s.c.DatabasePath, 0600, nil)
	if err != nil {
		return err
	}
	s.db = db
	return s.db.Update(func(tx *bolt.Tx) error {
		// Always create Users bucket.
		if _, err := tx.CreateBucketIfNotExists(UsersBucket); err != nil {
			return err
		}
		return nil
	})
}

func (u *User) SafeMail() string {
	return MailLoginRegExp.ReplaceAllString(u.Email, "***")
}

func (u *User) ToSafeUser() *User {
	return &User{
		ID:          u.ID,
		Email:       u.SafeMail(),
		Token:       nil,
		UpdatedAt:   u.UpdatedAt,
		UpdateCount: u.UpdateCount,
	}
}

func (s *Server) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	url := s.oAuthConf.AuthCodeURL(s.c.StateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) (*User, error) {
	q := r.URL.Query()
	if q.Get("state") != s.c.StateString {
		return nil, errors.New("Invalid oAuth2 state")
	}
	token, err := s.oAuthConf.Exchange(oauth2.NoContext, q.Get("code"))
	if err != nil {
		return nil, err
	}
	client := hhclient.NewClient(token)
	me, err := client.Me.GetMe()
	if err != nil {
		return nil, err
	}
	u := &User{
		ID:    me.ID,
		Email: me.Email,
		Token: token,
	}
	return u, nil
}

func (s *Server) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	user, err := s.userHandler(w, r)
	if err != nil {
		logrus.Error(err)
		http.Redirect(w, r, "/error.html", http.StatusFound)
		return
	}
	if _, ok := s.userList[user.ID]; !ok {
		s.userListChanged = true
		s.userList[user.ID] = user
		logrus.Infof("User %s added", user.Email)
	} else {
		logrus.Debugf("User %s logged", user.Email)
	}
	encodedCookie, err := s.Encrypt(user.ToSafeUser())
	if err != nil {
		logrus.Error(err)
		http.Redirect(w, r, "/error.html", http.StatusFound)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    s.c.CookieName,
		Value:   encodedCookie,
		Path:    "/",
		Domain:  s.c.CookieHostname,
		Expires: time.Now().AddDate(1, 0, 0),
		Secure:  s.c.CookieSecure,
		// Disallow access from JavaScript
		HttpOnly: true,
	})
	http.Redirect(w, r, "/logged.html", http.StatusFound)
}

func (s *Server) upAndPublishUserResumes(user *User) (int, error) {
	var updateCount int
	client := hhclient.NewClient(user.Token)
	if _, err := client.Me.GetMe(); err != nil {
		return 0, fmt.Errorf("Error getting information of user %s: %v", user.Email, err)
	}
	logrus.Debugf("Getting resumes for user: %s", user.Email)
	resumeList, err := client.Resume.ResumeMine()
	if err != nil {
		return 0, fmt.Errorf("Error getting resume for user %s: %v", user.Email, err)
	}
	if len(resumeList) == 0 {
		return 0, ErrEmptyResumeList
	}
	for _, r := range resumeList {
		logrus.Debugf("Requesting resume status: '%s'", r.Title)
		status, err := client.Resume.ResumesStatus(r)
		if err != nil {
			return 0, fmt.Errorf("Error getting resume status '%s': %v", r.Title, err)
		}
		if !status.CanPublishOrUpdate {
			logrus.Debugf("Skipping publish resume: '%s'", r.Title)
			continue
		}
		if err := client.Resume.ResumePublish(r); err != nil {
			return 0, fmt.Errorf("error publishing resume '%s': %s", r.Title, err)
		}
		if err := s.updateResume(client, r.ID, upExperience); err != nil {
			logrus.Errorf("error update resume '%s': fail %s", r.Title, err)
		}
		updateCount++
		logrus.Infof("Resume updated: '%s'", r.Title)
	}
	return updateCount, nil
}

func (s *Server) updateResume(client *hhclient.Client, resumeId string,
	upFunc func(companies []hhclient.Company, prefix string)) error {
	if len(s.c.ExperienceDescSuffix) == 0 {
		return nil
	}
	resume, err := client.Resume.ReadResume(resumeId)
	if err != nil {
		return fmt.Errorf("error read resume fail %s", err)
	}
	upFunc(resume.Experience, s.c.ExperienceDescSuffix)

	if err := client.Resume.EditResume(resume); err != nil {
		return fmt.Errorf("error editing resume fail %s", err)
	}
	return nil
}

func (s *Server) RestoreUserList() error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(UsersBucket)
		v := b.Get(UsersKey)
		if len(v) == 0 {
			logrus.Warn("No entries in database")
			return nil
		}
		if err := json.Unmarshal(v, &s.userList); err != nil {
			return err
		}
		return nil
	})
}

func (s *Server) SaveUserList() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(UsersBucket)
		encoded, err := json.Marshal(s.userList)
		if err != nil {
			return err
		}
		return b.Put(UsersKey, encoded)
	})
}

func (s *Server) Stop() error {
	return s.SaveUserList()
}

func (s *Server) Encrypt(body interface{}) (string, error) {
	return crypto.EncryptObj(body, s.c.CookieEncryptionCipher)
}

func (s *Server) Decrypt(encrypted string, body interface{}) error {
	return crypto.DecryptObj(encrypted, s.c.CookieEncryptionCipher, body)
}

func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   s.c.CookieName,
		Domain: s.c.CookieHostname,
		MaxAge: 0,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		http.Error(w, "Empty user data", http.StatusInternalServerError)
		return
	}
	delete(s.userList, user.ID)
	logrus.Infof("User %s deleted", user.Email)
}

func (s *Server) MeHandler(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		http.Error(w, "Empty user data", http.StatusInternalServerError)
		return
	}
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(user.ToSafeUser()); err != nil {
		http.Error(w, fmt.Sprintf("Cannot encode response data: %v", err), http.StatusInternalServerError)
		return
	}
}

func GetUserFromContext(r *http.Request) *User {
	if value := context.Get(r, UserCtxKey); value != nil {
		return value.(*User)
	}
	return nil
}

func SetUserToContext(r *http.Request, user *User) {
	context.Set(r, UserCtxKey, user)
}

func (s *Server) Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(s.c.CookieName)
		if err != nil || len(cookie.Value) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var safeUser *SafeUser
		if err := s.Decrypt(cookie.Value, &safeUser); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, ok := s.userList[safeUser.ID]
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		SetUserToContext(r, user)
		next.ServeHTTP(w, r)
	}
}

func (s *Server) UpdateLoop() {
	for {
		for _, user := range s.userList {
			logrus.Debugf("Getting information of user: %s", user.Email)
			tokenSource := s.oAuthConf.TokenSource(oauth2.NoContext, user.Token)
			newToken, err := tokenSource.Token()
			if err != nil {
				logrus.Errorf("Error getting token for user %s: %v", user.Email, err)
				continue
			}
			if user.Token.AccessToken != newToken.AccessToken {
				logrus.Infof("Updating token for user %s", user.Email)
				user.Token = newToken
				s.userListChanged = true
				logrus.Infof("New expiry date for user %s token: %s", user.Email, user.Token.Expiry.String())
			}
			updates, err := s.upAndPublishUserResumes(user)
			if err != nil {
				if err == ErrEmptyResumeList {
					logrus.Infof("Deleting user with empty resume list: %s", user.Email)
					delete(s.userList, user.ID)
					s.userListChanged = true
				}
				continue
			}
			if updates == 0 {
				// Skipping user update if nothing changed
				continue
			}
			user.UpdateCount = user.UpdateCount + updates
			user.UpdatedAt = time.Now().UTC()
			s.userListChanged = true
		}
		time.Sleep(s.c.UpdateInterval)
	}
}

func (s *Server) DumpLoop() {
	for {
		if s.userListChanged {
			logrus.Debug("Saving to disk...")
			if err := s.SaveUserList(); err != nil {
				logrus.Errorf("Error saving to disk: %v", err)
			} else {
				logrus.Debug("Saved to disk")
				s.userListChanged = false
			}
		}
		time.Sleep(s.c.DumpInterval)
	}
}

func (s *Server) Start() error {
	if err := s.RestoreUserList(); err != nil {
		return err
	}

	http.HandleFunc("/authorize", s.AuthorizeHandler)
	http.HandleFunc("/callback", s.CallbackHandler)

	http.HandleFunc("/logout", s.Auth(http.HandlerFunc(s.LogoutHandler)))
	http.HandleFunc("/delete", s.Auth(http.HandlerFunc(s.DeleteHandler)))
	http.HandleFunc("/me", s.Auth(http.HandlerFunc(s.MeHandler)))

	http.Handle("/", http.FileServer(http.Dir("./public")))

	go s.UpdateLoop()
	go s.DumpLoop()

	logrus.Infof("Started running on %s", s.c.ListenAddress)
	return http.ListenAndServe(s.c.ListenAddress, nil)
}
