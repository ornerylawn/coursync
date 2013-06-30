// Package browser provides access to coursera.org videos for courses
// in which you're enrolled.
package browser

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Browser struct {
	Client *http.Client
	CsrfToken string
}

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/28.0.1500.63 Safari/537.36"
	signinPageUrl = "https://www.coursera.org/account/signin"
	signinUrl = "https://www.coursera.org/maestro/api/user/login"
	courseAuthUrlFormat = "https://class.coursera.org/%s/auth/auth_redirector?type=login&subtype=normal"
	videoUrlFormat = "https://class.coursera.org/%s/lecture/download.mp4?lecture_id=%s"
	PauseDuration = 5 * time.Second
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
	if err != nil {
		return err
	}
	time.Sleep(PauseDuration)
	err = b.signIn(email, password)
	if err != nil {
		return err
	}
	return nil
}

func (b *Browser) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if b.CsrfToken != "" {
		req.AddCookie(&http.Cookie{
			Name: "csrftoken",
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
	fmt.Println(req)
	fmt.Println(b.Client.Jar)
	resp, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(b.Client.Jar)
	fmt.Println(string(bytes))
	return nil
}

// Signs into coursera. Must be called *after* `getSigninPage` so that
// the cookie jar has the right cookies.
func (b *Browser) signIn(email, password string) error {
	b.CsrfToken = makeCsrfToken()
	formData := url.Values{
		"email_address": {email},
		"password": {password},
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
	fmt.Println(req)
	fmt.Println(b.Client.Jar)
	resp, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(b.Client.Jar)
	fmt.Println(string(bytes))
	return nil
}

func makeCsrfToken() string {
	charset := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYX"
	token := "";
	for i := 0; i < 24; i++ {
		j := rand.Intn(len(charset))
		token += charset[j:j+1]
	}
	return token
}

func (b *Browser) GetCourseList() ([]string, error) {
	return []string{"audiomusicengpart1-001"}, nil
}

// Obtains cookies needed to access a course.
func (b *Browser) AuthCourse(course string) error {
	req, err := b.newRequest("GET", fmt.Sprintf(courseAuthUrlFormat, course), nil)
	if err != nil {
		return err
	}
	fmt.Println(req)
	fmt.Println(b.Client.Jar)
	resp, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(b.Client.Jar)
	fmt.Println(string(bytes))
	return nil
}

func (b *Browser) GetVideoList(course string) ([]string, error) {
	return []string{"25"}, nil
}

func (b *Browser) DownloadVideo(course, video string) (io.Reader, error) {
	req, err := b.newRequest("GET", fmt.Sprintf(videoUrlFormat, course, video), nil)
	if err != nil {
		return nil, err
	}
	fmt.Println(req)
	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
