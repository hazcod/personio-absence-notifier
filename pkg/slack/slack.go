package slack

import (
	"errors"
	"github.com/sirupsen/logrus"
)

type Slack struct {
	logger     *logrus.Logger
	WebhookURL string
}

func New(l *logrus.Logger, webhookURL string) (*Slack, error) {
	if l == nil {
		return nil, errors.New("nil logrus.Logger")
	}
	if webhookURL == "" {
		return nil, errors.New("missing slack webhook URL")
	}

	s := Slack{WebhookURL: webhookURL, logger: l}

	return &s, nil
}
