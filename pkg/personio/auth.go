package personio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	authURL = "https://api.personio.de/v1/auth"
)

type authResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
		Scope     string `json:"scope"`
	} `json:"data"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type tokenFactory struct {
	expires time.Time
	value   string
}

func (p *Personio) getToken() (string, error) {
	if p.token == nil {
		p.token = &tokenFactory{}
	}

	if p.token.expires.After(time.Now().Add(time.Second * 5)) {
		return p.token.value, nil
	}

	payload := struct {
		ClientID string `json:"client_id"`
		Secret   string `json:"client_secret"`
	}{
		ClientID: p.clientID,
		Secret:   p.secret,
	}

	payloadBytes, err := json.Marshal(&payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %w", err)
	}

	p.logger.WithField("client_id", p.clientID).Debug("authenticating to personio")

	req, err := http.NewRequest(http.MethodPost, authURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var response authResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return "", fmt.Errorf("failed with status code %d: %s", res.StatusCode, response.Error.Message)
	}

	if response.Data.Token == "" {
		return "", fmt.Errorf("received empty token from personio")
	}

	p.token.value = response.Data.Token
	p.token.expires = time.Now().Add(time.Duration(response.Data.ExpiresIn) * time.Second)

	p.logger.WithField("expires", p.token.expires.Format(time.RFC822)).
		Debug("successfully retrieved new token")

	return p.token.value, nil
}
