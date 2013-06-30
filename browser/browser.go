// Copyright 2013 Ryan Brown (ryan@ryanleebrown.com).

// Package browser provides access to coursera.org videos for courses
// in which you're enrolled.
package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// A Browser makes requests to coursera, scraping for course
// information and videos for the courses in which you are enrolled.
type Browser struct {
	Client     *http.Client
	CsrfToken  string
	User       User
	SigninTime time.Time
}

// A User represents the person who is logged-in.
type User struct {
	FullName string `json:"full_name"`
	Id       int
}

// A Topic is what we would think of as a "course" that may or may not
// be scheduled (e.g. "Machine Learning").
type Topic struct {
	Name      string
	ShortName string `json:"short_name"`
	Courses   []*Course
}

// A Course is a scheduled instance of a Topic (e.g. "A 10 week
// 'Machine Learning' course starting 4/22/2013")
type Course struct {
	Name     string
	Active   bool
	HomeLink string `json:"home_link"`
}

const (
	userAgent           = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/28.0.1500.63 Safari/537.36"
	signinPageUrl       = "https://www.coursera.org/account/signin"
	signinUrl           = "https://www.coursera.org/maestro/api/user/login"
	topicListUrlFormat  = "https://www.coursera.org/maestro/api/topic/list_my?user_id=%d"
	courseAuthUrlFormat = "%sauth/auth_redirector?type=login&subtype=normal"
	courseUrlFormat     = "%slecture/index"
	videoUrlFormat      = "%slecture/download.mp4?lecture_id="
)

var (
	// Controls the length of the pause before each request (to simulate
	// human-like behavior).
	PauseDuration = 3 * time.Second
	// Returned from anything other than SignIn() if the session is expired.
	ErrSessionExpired = errors.New("session expired")
)

// Create a new coursera browser.
func New() *Browser {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &Browser{
		Client: &http.Client{
			Jar: jar,
		},
	}
}

// Sign into coursera. Must be called before anything else.
func (b *Browser) SignIn(email, password string) error {
	err := b.getSigninPage()
	b.SigninTime = time.Now()
	if err != nil {
		return err
	}
	err = b.signIn(email, password)
	if err != nil {
		return err
	}
	return nil
}

func (b *Browser) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	// Simulate human-like behavior.
	time.Sleep(PauseDuration)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if b.CsrfToken != "" {
		req.AddCookie(&http.Cookie{
			Name:  "csrftoken",
			Value: b.CsrfToken,
		})
	}
	req.Header.Add("User-Agent", userAgent)
	return req, nil
}

// Fetches the signin page. Good for obtaining any cookies needed to
// sign in.
func (b *Browser) getSigninPage() error {
	req, err := b.newRequest("GET", signinPageUrl, nil)
	if err != nil {
		return err
	}
	resp, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	return err
}

// Signs into coursera. Must be called *after* `getSigninPage` so that
// the cookie jar has the right cookies.
func (b *Browser) signIn(email, password string) error {
	b.CsrfToken = makeCsrfToken()
	formData := url.Values{
		"email_address": {email},
		"password":      {password},
	}
	req, err := b.newRequest("POST", signinUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Origin", "https://www.coursera.org")
	req.Header.Add("Referer", signinPageUrl)
	req.Header.Add("X-CSRFToken", b.CsrfToken)
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	resp, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&b.User)
	return err
}

func makeCsrfToken() string {
	charset := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYX"
	token := ""
	for i := 0; i < 24; i++ {
		j := rand.Intn(len(charset))
		token += charset[j : j+1]
	}
	return token
}

// Retrieves all of the topics in which the current user is enrolled.
func (b *Browser) GetTopicList() ([]*Topic, error) {
	if b.SessionExpired() {
		return nil, ErrSessionExpired
	}
	req, err := b.newRequest("GET", fmt.Sprintf(topicListUrlFormat, b.User.Id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	topics := make([]*Topic, 0)
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&topics)
	return topics, err
}

// Obtains cookies needed to access a course.
func (b *Browser) AuthCourse(course *Course) error {
	if b.SessionExpired() {
		return ErrSessionExpired
	}
	req, err := b.newRequest("GET", fmt.Sprintf(courseAuthUrlFormat, course.HomeLink), nil)
	if err != nil {
		return err
	}
	resp, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	return err
}

// Retrieves all of the video urls for the course.
func (b *Browser) GetVideoUrlList(course *Course) ([]string, error) {
	if b.SessionExpired() {
		return nil, ErrSessionExpired
	}
	// Get the HTML that has the video urls embeded.
	req, err := b.newRequest("GET", fmt.Sprintf(courseUrlFormat, course.HomeLink), nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Search the HTML for video urls.
	re, err := regexp.Compile(regexp.QuoteMeta(fmt.Sprintf(videoUrlFormat, course.HomeLink)) + `\d*`)
	if err != nil {
		return nil, err
	}
	return re.FindAllString(string(bytes), -1), nil
}

// Downloads a video if a file with the same name doesn't already
// exist in the directory. Creates the directory if it doesn't already
// exist. Returns os.ErrExist if the file already exists.
func (b *Browser) SyncVideo(dirname, videoUrl string) error {
	if b.SessionExpired() {
		return ErrSessionExpired
	}
	// Create the directory if it doesn't already exist.
	err := os.Mkdir(dirname, 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}
	// Start the download now in order to obtain the filename.
	r, err := b.StartVideoDownload(videoUrl)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	// Parse filename.
	disp, ok := r.Header[http.CanonicalHeaderKey("Content-Disposition")]
	if !ok || len(disp) != 1 {
		return errors.New("error parsing filename from response")
	}
	start := strings.Index(disp[0], `"`)
	end := strings.LastIndex(disp[0], `"`)
	if start == -1 || end == -1 {
		return errors.New("error parsing filename from response")
	}
	basename := disp[0][start+1 : end]
	// Check if file already exists.
	filename := fmt.Sprintf("%s/%s", dirname, basename)
	_, err = os.Stat(filename)
	if err == nil {
		return os.ErrExist
	}
	if !os.IsNotExist(err) {
		return err
	}
	// File doesn't exist, continue with download.
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	io.Copy(f, r.Body)
	return nil
}

// Sends a request for the video and returns the *http.Response.
func (b *Browser) StartVideoDownload(videoUrl string) (*http.Response, error) {
	if b.SessionExpired() {
		return nil, ErrSessionExpired
	}
	req, err := b.newRequest("GET", videoUrl, nil)
	if err != nil {
		return nil, err
	}
	return b.Client.Do(req)
}

func (b *Browser) SessionExpired() bool {
	return b.SigninTime.Add(15 * time.Minute).Before(time.Now())
}
