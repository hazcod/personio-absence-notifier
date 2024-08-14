package personio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	timeOffURL = "https://api.personio.de/v1/company/time-offs"
	queryLimit = 200
)

type Employee struct {
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
}

type abscenceResponse struct {
	Success  bool `json:"success"`
	Metadata struct {
		TotalElements int `json:"total_elements"`
		CurrentPage   int `json:"current_page"`
		TotalPages    int `json:"total_pages"`
	} `json:"metadata"`
	Data []struct {
		Type       string `json:"type"`
		Attributes struct {
			ID           int     `json:"id"`
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
			Employee    Employee `json:"employee"`
			Certificate struct {
				Status string `json:"status"`
			} `json:"certificate"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		} `json:"attributes"`
	} `json:"data"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func (p *Personio) GetAbscences() ([]string, error) {
	token, err := p.getToken()
	if err != nil {
		return nil, fmt.Errorf("could not get auth value: %w", err)
	}

	today := time.Now().Format("2006-01-02")

	var absentees []string

	page := 0
	pages := 1

	for {
		params := url.Values{}
		params.Add("limit", fmt.Sprintf("%d", queryLimit))
		// weird bug where page=0 and page=1 return same results from personio API. so just immediately fetch page=1
		params.Add("offset", fmt.Sprintf("%d", page+1))
		params.Add("start_date", today)
		params.Add("end_date", today)

		fullURL := fmt.Sprintf("%s?%s", timeOffURL, params.Encode())

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

		var response abscenceResponse
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
			absentees = append(absentees,
				data.Attributes.Employee.Attributes.FirstName.Value+" "+
					data.Attributes.Employee.Attributes.LastName.Value,
			)
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
