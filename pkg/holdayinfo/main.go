package holdayinfo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HolydayInfo struct {
	Today        string   `json:"today"`
	Celebrations []string `json:"celebrations,omitempty"`
	Message      string   `json:"message,omitempty"`
}

func fetchHolidays() ([]map[string]any, error) {
	url := "https://date.nager.at/Api/v3/NextPublicHolidaysWorldwide"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch calendar data, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var holidays []map[string]any
	if err := json.Unmarshal(body, &holidays); err != nil {
		return nil, err
	}

	return holidays, nil
}

// GetHolydayInfo returns today's celebrations
func GetHolydayInfo() (*HolydayInfo, error) {
	holidays, err := fetchHolidays()
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("2006-01-02")
	todayCelebrations := []string{}

	for _, h := range holidays {
		if date, ok := h["date"].(string); ok && date == today {
			if name, ok := h["name"].(string); ok {
				todayCelebrations = append(todayCelebrations, name)
			}
		}
	}

	info := &HolydayInfo{Today: today}
	if len(todayCelebrations) == 0 {
		info.Message = "Nothing is celebrated today. Have a nice day!"
	} else {
		info.Celebrations = todayCelebrations
	}

	return info, nil
}

func GetHolydayInfoJSON() ([]byte, error) {
	info, err := GetHolydayInfo()
	if err != nil {
		return nil, errors.New("failed to get calendar info: " + err.Error())
	}

	return json.MarshalIndent(info, "", "  ")
}
