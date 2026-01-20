package typesense

import (
	"bytes"
	"fmt"
	"net/http"
)

func (c *Client) CreateCollection(schema []byte) error {
	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/collections", c.BaseURL),
		bytes.NewReader(schema),
	)
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("collection creation failed")
	}

	return nil
}
