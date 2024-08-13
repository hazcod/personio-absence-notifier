package main

import (
	"flag"
	"fmt"
	"github.com/hazcod/personio-abscences/config"
	"github.com/hazcod/personio-abscences/pkg/personio"
	"github.com/hazcod/personio-abscences/pkg/slack"
	"github.com/sirupsen/logrus"
	"os"
	"sort"
)

func main() {
	// ctx := context.Background()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	confFile := flag.String("config", "config.yml", "The YAML configuration file.")
	flag.Parse()

	conf := config.Config{}
	if err := conf.Load(*confFile); err != nil {
		logger.WithError(err).WithField("config", *confFile).Fatal("failed to load configuration")
	}

	if err := conf.Validate(); err != nil {
		logger.WithError(err).WithField("config", *confFile).Fatal("invalid configuration")
	}

	logrusLevel, err := logrus.ParseLevel(conf.Log.Level)
	if err != nil {
		logger.WithError(err).Error("invalid log level provided")
		logrusLevel = logrus.InfoLevel
	}
	logger.SetLevel(logrusLevel)

	// ---

	pers, err := personio.New(logger, conf.Personio.ClientID, conf.Personio.Secret)
	if err != nil {
		logger.WithError(err).Fatal("failed to create personio client")
	}

	absentees, err := pers.GetAbscences()
	if err != nil {
		logger.WithError(err).Fatal("failed to get abscences")
	}

	if len(absentees) == 0 {
		logger.Info("no abscences found for today")
		os.Exit(0)
	}

	sort.Strings(absentees)

	message := fmt.Sprintf(":x: *Out today* (%d):\n", len(absentees))
	for _, absentee := range absentees {
		message += fmt.Sprintf("\n- %s", absentee)
	}

	slacker, err := slack.New(logger, conf.Slack.WebhookURL)
	if err != nil {
		logger.WithError(err).Fatal("failed to create slack client")
	}

	if err := slacker.Send(message); err != nil {
		logger.WithError(err).Fatal("failed to send Slack message")
	}

	logger.Info("sent absentee messages")
}
