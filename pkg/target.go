package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

type RequestMethod string
type Target struct {
	URL                  string
	RequestMethod        RequestMethod
	RequestHeaders       string `gorm:"type:json"`
	HealthyMinimumStatus int64
	HealthyMaximumStatus int64
	ResponseType         string
	HealthyContent       string
	Timeout              time.Duration
	gorm.Model
}

var (
	InvalidUrl   = errors.New("An invalid url was provided")
	RequestError = errors.New("There was an error making the HTTP request")
)

func (t *Target) MakeRequest() (*http.Response, error) {
	client := http.Client{}
	request := http.Request{
		Method: string(t.RequestMethod),
	}
	url, err := url.Parse(t.URL)
	if err != nil {
		return nil, InvalidUrl
	}
	request.URL = url

	if t.RequestHeaders != "" {

		var headers map[string]string

		reader := strings.NewReader(t.RequestHeaders)

		json.NewDecoder(reader).Decode(&headers)

		request.Header = make(map[string][]string)
		for key, value := range headers {
			request.Header.Add(key, string(value))
		}
	}

	resp, err := client.Do(&request)

	if err != nil {
		return nil, RequestError
	}
	return resp, nil

}
func (t *Target) ParseContent(resp *http.Response) string {

	type GenericJson map[string]interface{}
	var res GenericJson

	json.NewDecoder(resp.Body).Decode(&res)

	return stringFromJson(res)

}

type Checker struct {
	StatusCode int
	Content    string
}

func (t *Target) IsHealthyCheck(check Checker) (bool, error) {

	var err error
	statusIsHealthy := check.StatusCode >= int(t.HealthyMinimumStatus) && check.StatusCode <= int(t.HealthyMaximumStatus)

	if !statusIsHealthy {
		err = fmt.Errorf("THE RESPONSE HAS AN UNHEALTHY STATUSCODE OF %d", check.StatusCode)
	}
	var contentIsCorrect bool = true

	if t.HealthyContent != "" {
		contentIsCorrect = strings.Contains(check.Content, t.HealthyContent)
	}
	// TODO make it have both not just the first or last
	if !contentIsCorrect {
		err = fmt.Errorf("THE RESPONSE DID NOT CONTAIN %s", t.HealthyContent)
	}

	return statusIsHealthy && contentIsCorrect, err
}

type TargetRepository struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *TargetRepository {
	return &TargetRepository{db: db}
}

func (r *TargetRepository) FetchAll() []*Target {
	var targets []*Target
	r.db.Find(&targets).Limit(10)
	fmt.Println(len(targets))
	return targets
}

func stringFromJson(m map[string]interface{}) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
	}
	return b.String()
}
