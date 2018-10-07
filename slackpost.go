package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/appengine/urlfetch"

	"google.golang.org/appengine"
)

type slackClient struct {
	httpClient *http.Client
	token      string
}

type slackResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}
type slackUser struct {
	ID      string `json:"id"`
	Profile struct {
		DisplayName string `json:"display_name"`
	} `json:"profile"`
}
type slackUserList struct {
	Members          []slackUser           `json:"members"`
	ResponseMetadata slackResponseMetadata `json:"response_metadata"`
}
type slackPostResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// Call makes a Slack API call, sending arguments and parsing into output.
func (s *slackClient) callAndUnmarshal(method string, arguments url.Values, output interface{}) error {
	httpMethod := http.MethodGet
	var body io.Reader
	if len(arguments) > 0 {
		httpMethod = http.MethodPost
		body = strings.NewReader(arguments.Encode())
	}
	req, err := http.NewRequest(httpMethod, "https://slack.com/api/"+method, body)
	if err != nil {
		return err
	}
	if httpMethod == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	out, err := ioutil.ReadAll(resp.Body)
	err2 := resp.Body.Close()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
		return fmt.Errorf("failed request: %d %s", resp.StatusCode, resp.Status)
	}

	return json.Unmarshal(out, output)
}

func findUserID(client *slackClient, displayName string) (string, error) {
	if len(displayName) > 0 && displayName[0] == '@' {
		displayName = displayName[1:]
	}

	userList := &slackUserList{}
	err := client.callAndUnmarshal("users.list", url.Values{"limit": []string{"100"}}, userList)
	if err != nil {
		return "", err
	}
	if userList.ResponseMetadata.NextCursor != "" {
		return "", fmt.Errorf("requires paginated response")
	}

	for _, user := range userList.Members {
		if user.Profile.DisplayName == displayName {
			return user.ID, nil
		}
	}
	return "", fmt.Errorf("display_name '%s' not found", displayName)
}

type clientRequest struct {
	Token       string `json:"token"`
	DisplayName string `json:"display_name"`
	Text        string `json:"text"`
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/_ah/start" {
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}

	reader := json.NewDecoder(r.Body)
	parsedRequest := &clientRequest{}
	err := reader.Decode(parsedRequest)
	if err != nil {
		panic(err)
	}

	ctx := appengine.NewContext(r)
	urlfetchClient := urlfetch.Client(ctx)
	client := &slackClient{urlfetchClient, parsedRequest.Token}

	userID, err := findUserID(client, parsedRequest.DisplayName)
	if err != nil {
		panic(err)
	}

	v := url.Values{}
	v.Add("channel", userID)
	v.Add("text", parsedRequest.Text)
	v.Add("as_user", "true")
	postResponse := &slackPostResponse{}
	// TODO: Don't decode this post response? Just proxy it?
	err = client.callAndUnmarshal("chat.postMessage", v, postResponse)
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(w)
	err = encoder.Encode(postResponse)
	if err != nil {
		panic(err)
	}
}

func main() {

	http.HandleFunc("/", handle)
	appengine.Main()

}
