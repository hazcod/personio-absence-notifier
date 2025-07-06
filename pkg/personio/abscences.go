package personio

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	timeOffURL = "https://api.personio.de/v1/company/time-offs"
	absenceURL = "https://api.personio.de/v1/company/absence-periods"
	queryLimit = 200

	OffMorning = iota
	OffAfternoon
	OffFullday
)

type Absentee struct {
	FullName string
	Type     uint16
}

type employee struct {
	Type       string `json:"type"`
	Attributes struct {
		FirstName struct {
			Label       string `json:"label"`
			Value       string `json:"value"`
			Type        string `json:"type"`
			UniversalID string `json:"universal_id"`
		} `json:"first_name"`
		LastName struct {
			Label       string `json:"label"`
			Value       string `json:"value"`
			Type        string `json:"type"`
			UniversalID string `json:"universal_id"`
		} `json:"last_name"`
		Email struct {
			Label       string `json:"label"`
			Value       string `json:"value"`
			Type        string `json:"type"`
			UniversalID string `json:"universal_id"`
		} `json:"email"`
	} `json:"attributes"`
}

type apiEmployee struct {
	Type       string `json:"type"`
	Attributes struct {
		Status       string  `json:"status"`
		StartDate    string  `json:"start_date"`
		EndDate      string  `json:"end_date"`
		DaysCount    float32 `json:"days_count"`
		HalfDayStart int     `json:"half_day_start"`
		HalfDayEnd   int     `json:"half_day_end"`
		TimeOffType  struct {
			Type       string `json:"type"`
			Attributes struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Category string `json:"category"`
			} `json:"attributes"`
		} `json:"time_off_type"`
		Employee    employee `json:"employee"`
		Certificate struct {
			Status string `json:"status"`
		} `json:"certificate"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

type timeOffResponse struct {
	Success  bool `json:"success"`
	Metadata struct {
		TotalElements int `json:"total_elements"`
		CurrentPage   int `json:"current_page"`
		TotalPages    int `json:"total_pages"`
	} `json:"metadata"`
	Data   []apiEmployee `json:"data"`
	Offset int           `json:"offset"`
	Limit  int           `json:"limit"`
}

func isInslice(item string, list []string) bool {
	for _, sliceItem := range list {
		if strings.EqualFold(item, sliceItem) {
			return true
		}
	}

	return false
}

func uniqueSlice(s []string) []string {
	inResult := make(map[string]bool)
	var result []string
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}

// https://developer.personio.de/v1.0/reference/get_company-time-offs

func (p *Personio) GetTimeOffs() ([]Absentee, error) {
	now := time.Now()
	nowZero := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// timeoff endpoint
	morningCheckpoint := nowZero.Add(time.Hour * 9)
	p.logger.WithField("check", morningCheckpoint).Debugf("retrieving morning absences")
	morningOff, err := p.retrieveTimeOffs(morningCheckpoint)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve morning abscences: %v", err)
	}

	afternoonCheckpoint := nowZero.Add(time.Hour * 16)
	var afternoonOff []string
	if afternoonCheckpoint != morningCheckpoint {
		p.logger.WithField("check", afternoonCheckpoint).Debugf("retrieving afternoon absences")
		afternoonOff, err = p.retrieveTimeOffs(afternoonCheckpoint)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve afternoon abscences: %v", err)
		}
	}

	// abdsence endpoint
	morningOff2, err := p.retrieveAbsences(morningCheckpoint)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve morning absences: %v", err)
	}
	morningOff = append(morningOff, morningOff2...)

	if afternoonCheckpoint != morningCheckpoint {
		afternoonOff2, err := p.retrieveAbsences(afternoonCheckpoint)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve afternoon absences: %v", err)
		}
		afternoonOff = append(afternoonOff, afternoonOff2...)
	}

	var absentees []Absentee
	for _, absentee := range uniqueSlice(append(morningOff, afternoonOff...)) {
		isMorningOff := isInslice(absentee, morningOff)
		isAfternoonOff := isInslice(absentee, afternoonOff)

		// person is off for the whole day
		if isMorningOff && isAfternoonOff {
			absentees = append(absentees, Absentee{
				FullName: absentee,
				Type:     OffFullday,
			})
			continue
		}

		// person is off in the morning only
		if isMorningOff && !isAfternoonOff {
			absentees = append(absentees, Absentee{
				FullName: absentee,
				Type:     OffMorning,
			})
			continue
		}

		// person is off in the afternoon
		if !isMorningOff && isAfternoonOff {
			absentees = append(absentees, Absentee{
				FullName: absentee,
				Type:     OffFullday,
			})
			continue
		}

		p.logger.Fatalf("unreachable statement in getAbsences")
	}

	return absentees, nil
}

func capitalize(str string) string {
	if len(str) == 1 {
		return strings.ToUpper(str)
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

func tryGetGivenName(email, firstName, lastName string) (string, error) {
	emailParts := strings.SplitN(
		strings.SplitN(email, "@", 2)[0],
		".", 2,
	)

	if len(emailParts) != 2 {
		return fmt.Sprintf("%s %s", capitalize(firstName), capitalize(lastName)), nil
	}

	return fmt.Sprintf("%s %s", capitalize(emailParts[0]), capitalize(emailParts[1])), nil
}

type absenceResponse struct {
	Success  bool `json:"success"`
	Metadata struct {
		TotalElements int `json:"total_elements"`
		CurrentPage   int `json:"current_page"`
		TotalPages    int `json:"total_pages"`
	} `json:"metadata"`
	Data []struct {
		Type       string `json:"type"`
		Attributes struct {
			ID                string `json:"id"`
			MeasurementUnit   string `json:"measurement_unit"`
			EffectiveDuration int    `json:"effective_duration"`
			Employee          struct {
				Type       string `json:"type"`
				Attributes struct {
					ID struct {
						Label       string `json:"label"`
						Value       int    `json:"value"`
						Type        string `json:"type"`
						UniversalID string `json:"universal_id"`
					} `json:"id"`
					FirstName struct {
						Label       string `json:"label"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						UniversalID string `json:"universal_id"`
					} `json:"first_name"`
					LastName struct {
						Label       string `json:"label"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						UniversalID string `json:"universal_id"`
					} `json:"last_name"`
					Email struct {
						Label       string `json:"label"`
						Value       string `json:"value"`
						Type        string `json:"type"`
						UniversalID string `json:"universal_id"`
					} `json:"email"`
				} `json:"attributes"`
			} `json:"employee"`
			AbsenceType struct {
				Type       string `json:"type"`
				Attributes struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					TimeOffTypeID int    `json:"time_off_type_id"`
				} `json:"attributes"`
			} `json:"absence_type"`
			Certificate struct {
				Status string `json:"status"`
			} `json:"certificate"`
			Start        time.Time `json:"start"`
			End          time.Time `json:"end"`
			HalfDayStart bool      `json:"half_day_start"`
			HalfDayEnd   bool      `json:"half_day_end"`
			Comment      string    `json:"comment"`
			Origin       string    `json:"origin"`
			Status       string    `json:"status"`
			CreatedBy    int       `json:"created_by"`
			CreatedAt    time.Time `json:"created_at"`
			UpdatedAt    time.Time `json:"updated_at"`
			ApprovedAt   time.Time `json:"approved_at"`
			Breakdowns   []struct {
				Date              string `json:"date"`
				EffectiveDuration int    `json:"effective_duration"`
			} `json:"breakdowns"`
		} `json:"attributes"`
	} `json:"data"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func (p *Personio) retrieveAbsences(checkDate time.Time) ([]string, error) {
	token, err := p.getToken()
	if err != nil {
		return nil, fmt.Errorf("could not get auth value: %w", err)
	}

	checkDateFormatted := checkDate.Format("2006-01-02")

	var absentees []string

	page := 0
	pages := 1

	for {
		params := url.Values{}
		params.Add("limit", fmt.Sprintf("%d", queryLimit))
		params.Add("offset", fmt.Sprintf("%d", page))
		params.Add("start_date", checkDateFormatted)
		params.Add("end_date", checkDateFormatted)

		fullURL := fmt.Sprintf("%s?%s", absenceURL, params.Encode())

		p.logger.WithField("url", fullURL).Debug("getting absentees")

		req, err := http.NewRequest(http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("could not create request: %w", err)
		}

		req.Header.Add("accept", "application/json")
		req.Header.Add("authorization", "Bearer "+token)

		p.logger.WithField("page", page).WithField("total_pages", pages).WithField("url", fullURL).
			Debug("fetching abscences")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("could not get abscences: %w", err)
		}

		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read abscences: %w", err)
		}

		if p.logger.IsLevelEnabled(logrus.DebugLevel) {
			p.logger.Println(string(body))
		}

		var response absenceResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("could not parse abscences: %w", err)
		}

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("could not get abscences: status code %d", res.StatusCode)
		}

		p.logger.WithField("page", page).WithField("total_pages", pages).WithField("url", fullURL).
			WithField("returned", len(response.Data)).
			Debug("received abscences")

		for _, data := range response.Data {
			fullName, err := tryGetGivenName(
				data.Attributes.Employee.Attributes.Email.Value,
				data.Attributes.Employee.Attributes.FirstName.Value,
				data.Attributes.Employee.Attributes.LastName.Value,
			)
			if err != nil {
				return nil, fmt.Errorf("could not get full name for %s: %w",
					data.Attributes.Employee.Attributes.Email.Value, err)
			}

			absentees = append(absentees, fullName)
		}

		// Determine if there are more pages to fetch
		p.logger.Tracef("set total pages to %d", response.Metadata.TotalPages)

		if page+1 >= response.Metadata.TotalPages {
			break
		}

		pages = response.Metadata.TotalPages
		page += 1
	}

	p.logger.WithField("total", len(absentees)).Debug("retrieved abscences")

	return absentees, nil
}

func (p *Personio) retrieveTimeOffs(checkDate time.Time) ([]string, error) {
	token, err := p.getToken()
	if err != nil {
		return nil, fmt.Errorf("could not get auth value: %w", err)
	}

	checkDateFormatted := checkDate.Format("2006-01-02")

	var absentees []string

	page := 0
	pages := 1

	for {
		params := url.Values{}
		params.Add("limit", fmt.Sprintf("%d", queryLimit))
		params.Add("offset", fmt.Sprintf("%d", page))
		params.Add("start_date", checkDateFormatted)
		params.Add("end_date", checkDateFormatted)

		fullURL := fmt.Sprintf("%s?%s", timeOffURL, params.Encode())

		p.logger.WithField("url", fullURL).Debug("getting timeoffs")

		req, err := http.NewRequest(http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("could not create request: %w", err)
		}

		req.Header.Add("accept", "application/json")
		req.Header.Add("authorization", "Bearer "+token)

		p.logger.WithField("page", page).WithField("total_pages", pages).WithField("url", fullURL).
			Debug("fetching timeoffs")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("could not get timeoffs: %w", err)
		}

		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read timeoffs: %w", err)
		}

		if p.logger.IsLevelEnabled(logrus.DebugLevel) {
			p.logger.Println(string(body))
		}

		var response timeOffResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("could not parse timeoffs: %w", err)
		}

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("could not get timeoffs: status code %d", res.StatusCode)
		}

		p.logger.WithField("page", page).WithField("total_pages", pages).WithField("url", fullURL).
			WithField("returned", len(response.Data)).
			Debug("received timeoffs")

		for _, data := range response.Data {
			fullName, err := tryGetGivenName(
				data.Attributes.Employee.Attributes.Email.Value,
				data.Attributes.Employee.Attributes.FirstName.Value,
				data.Attributes.Employee.Attributes.LastName.Value,
			)
			if err != nil {
				return nil, fmt.Errorf("could not get full name for %s: %w",
					data.Attributes.Employee.Attributes.Email.Value, err)
			}

			absentees = append(absentees, fullName)
		}

		// Determine if there are more pages to fetch
		p.logger.Tracef("set total pages to %d", response.Metadata.TotalPages)

		if page+1 >= response.Metadata.TotalPages {
			break
		}

		pages = response.Metadata.TotalPages
		page += 1
	}

	p.logger.WithField("total", len(absentees)).Debug("retrieved timeoffs")

	return absentees, nil
}
