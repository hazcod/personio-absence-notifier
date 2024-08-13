package personio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
		return nil, fmt.Errorf("could not get auth token: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("token was empty")
	}

	today := time.Now().Format("2006-01-02")

	url := "https://api.personio.de/v1/company/time-offs?limit=200&offset=0&start_date=" + today + "&end_date=" + today

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Bearer "+token)

	p.logger.Debugf("retrieving abscences from personio")

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

	absentees := make([]string, len(response.Data))

	for i, data := range response.Data {
		absentees[i] = data.Attributes.Employee.Attributes.FirstName.Value + " " + data.Attributes.Employee.Attributes.LastName.Value
	}

	p.logger.WithField("total", len(absentees)).Debug("retrieved abscenes")

	return absentees, nil
}
