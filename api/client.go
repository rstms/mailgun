package api

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type Client struct {
	client  http.Client
	url     string
	apiKey  string
	verbose bool
}

type Domain struct {
    Id	string
    Name string
    Type string

}
type Domains struct {
    Count int // `json:"total_count"`
    Items []Domain
}


func NewClient() (*Client, error) {

	viper.SetDefault("url", "https://api.mailgun.net")
	viper.SetDefault("verbose", false)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := Client{
		client: http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				IdleConnTimeout: 5 * time.Second,
			},
		},
		url:     viper.GetString("url"),
		apiKey:  viper.GetString("api_key"),
		verbose: viper.GetBool("verbose"),
	}
	return &c, nil
}

func (c *Client) req(method, path string, params *map[string]any, body io.Reader, response *map[string]interface{}) (int, string, error) {

	requestUrl := c.url + path

	if params != nil && len(*params) > 0 {
		urlParams := url.Values{}
		for name, value := range *params {
			urlParams.Add(name, fmt.Sprintf("%v", value))
		}
		requestUrl += "?" + urlParams.Encode()
	}

	req, err := http.NewRequest(method, requestUrl, body)
	if err != nil {
		return 0, "", fmt.Errorf("failed creating request: %v", err)
	}
	req.SetBasicAuth("api", c.apiKey)

	if c.verbose {
		log.Printf("request: %+v\n", req)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed making request: %v", err)
	}
	defer resp.Body.Close()

	if response != nil {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, "", fmt.Errorf("failed reading response body: %v", err)
		}
		err = json.Unmarshal(data, response)
		if err != nil {
			return 0, "", fmt.Errorf("failed decoding response body: %v", err)
		}
	}
	if c.verbose {
		log.Printf("response: %s\n", resp.Status)
	}
	return resp.StatusCode, resp.Status, nil
}

func (c *Client) get(path string, params *map[string]any) (int, string, *map[string]interface{}, error) {
	response := make(map[string]interface{})
	code, status, err := c.req("GET", path, params, nil, &response)
	return code, status, &response, err
}

func (c *Client) Domains() ([]string, error) {
	code, status, domains, err := c.get("/v4/domains", nil)
	if err != nil {
		return []string{}, err
	}
	if code < 200 || code >= 300 {
		return []string{}, fmt.Errorf("request failed: %s", status)
	}
	items := (*domains)["items"]
	fmt.Printf("%v\n", items)
	/*
	ret := make([]string, len(items))
	for i, item := range items {
		ret[i] = item["name"]
	}
	*/
	return ret, nil
}

func dumpJSON(object *map[string]interface{}) error {
	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		fmt.Errorf("failed decoding JSON: %v", err)
	}
	log.Println(string(data))
	return nil
}
