package personio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (p *Personio) getToken() (string, error) {
	url := "https://api.personio.de/v1/auth"

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

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(payloadBytes)))
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

	p.logger.Debug("successfully retrieved token")

	return response.Data.Token, nil
}
