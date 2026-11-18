package adapters

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type FCM struct {
	client   *http.Client
	endpoint string
}

func NewFCM(ctx context.Context, encoded string) (*FCM, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	var key struct {
		Project string `json:"project_id"`
		Email   string `json:"client_email"`
		Key     string `json:"private_key"`
	}
	if err != nil || json.Unmarshal(raw, &key) != nil || key.Project == "" || key.Email == "" || key.Key == "" {
		return nil, errors.New("invalid FCM_SERVICE_ACCOUNT_B64")
	}
	auth := jwt.Config{Email: key.Email, PrivateKey: []byte(key.Key), TokenURL: "https://oauth2.googleapis.com/token", Scopes: []string{"https://www.googleapis.com/auth/firebase.messaging"}}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: 4 * time.Second})
	client := auth.Client(ctx)
	client.Timeout = 7 * time.Second
	return &FCM{client: client, endpoint: "https://fcm.googleapis.com/v1/projects/" + url.PathEscape(key.Project) + "/messages:send"}, nil
}
func (f *FCM) Send(ctx context.Context, fid, id string) (bool, error) {
	data, _ := json.Marshal(map[string]any{"message": map[string]any{"fid": fid, "notification": map[string]string{"title": "SAYA", "body": "Доступен новый опрос"}, "data": map[string]string{"route": "weekly_survey"}, "android": map[string]any{"collapse_key": "weekly_survey", "ttl": "3600s", "notification": map[string]string{"tag": "weekly_survey"}}}})
	req, err := http.NewRequestWithContext(ctx, "POST", f.endpoint, bytes.NewReader(data))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return false, errors.New("FCM transport unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return false, nil
	}
	var failure struct {
		Error struct {
			Details []struct {
				Code string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&failure) == nil {
		for _, d := range failure.Error.Details {
			if d.Code == "UNREGISTERED" {
				return true, nil
			}
		}
	}
	return false, errors.New("FCM send rejected")
}
