package tibberapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func GetTibberAPIPayload(tibberApiKey string, graphqlQuery string) ([]byte, error) {
	// Define the GraphQL query payload
	payload := map[string]string{
		"query": graphqlQuery,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "https://api.tibber.com/v1-beta/gql", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the required headers
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", "Bearer "+tibberApiKey)
	req.Header.Set("content-type", "application/json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
