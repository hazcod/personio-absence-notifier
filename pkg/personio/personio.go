package personio

import "fmt"
import "github.com/sirupsen/logrus"

type Personio struct {
	logger   *logrus.Logger
	clientID string
	secret   string
	token    *tokenFactory
}

func New(l *logrus.Logger, clientID, secret string) (*Personio, error) {
	if l == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	if clientID == "" || secret == "" {
		return nil, fmt.Errorf("clientID or secret is empty")
	}

	p := Personio{logger: l, clientID: clientID, secret: secret, token: &tokenFactory{}}

	return &p, nil
}
