package typesense

import (
	"fmt"
	"io"
	"net/http"
)

func (c *Client) GetSchema(collection string) ([]byte, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/collections/%s", c.BaseURL, collection),
		nil,
	)
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch schema")
	}

	return io.ReadAll(resp.Body)
}
