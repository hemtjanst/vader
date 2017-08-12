package vader

import (
	"fmt"
	wunder "github.com/greyfocus/go-wunderground-api"
)

// GetWeather returns the current conditions for the requested location
func GetWeather(token string, location string) (*wunder.Conditions, error) {
	client := wunder.JsonClient{ApiKey: token}
	request := wunder.Request{Features: []string{"conditions"}, Location: location}
	resp, err := client.Execute(&request)
	if err != nil {
		return nil, fmt.Errorf("Could not fetch weather: %s", err)
	}

	if resp.CurrentConditions == nil {
		return nil, fmt.Errorf("Current conditions are missing, check you have a valid API key")
	}

	return resp.CurrentConditions, nil
}
