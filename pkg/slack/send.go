package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Slack) Send(message string) error {
	payload := struct {
		Text string `json:"text"`
	}{
		Text: message,
	}

	b, err := json.Marshal(&payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	resp, err := http.Post(s.WebhookURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("could not send Slack message: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not send Slack message: %s", resp.Status)
	}

	return nil
}
