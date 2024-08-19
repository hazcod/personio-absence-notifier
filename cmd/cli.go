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
		logger.WithError(err).Fatal("failed to create Personio client")
	}

	absentees, err := pers.GetAbsences()
	if err != nil {
		logger.WithError(err).Fatal("failed to get absences")
	}

	if len(absentees) == 0 {
		logger.Info("no absences found for today")
		os.Exit(0)
	}

	sort.Slice(absentees, func(i, j int) bool {
		return absentees[i].FullName < absentees[j].FullName
	})

	message := fmt.Sprintf(":x: *Out today* (%d):\n", len(absentees))

	for _, absentee := range absentees {
		switch absentee.Type {
		case personio.OffFullday:
			message += fmt.Sprintf("\n- %s", absentee.FullName)
		case personio.OffMorning:
			message += fmt.Sprintf("\n- %s _(morning)_", absentee.FullName)
		case personio.OffAfternoon:
			message += fmt.Sprintf("\n- %s _(afternoon)_", absentee.FullName)
		default:
			logger.WithField("type", absentee.Type).Fatal("unknown type")
		}
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
