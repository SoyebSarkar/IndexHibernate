package typesense

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) ImportDocuments(collection string, r io.Reader) error {
	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/collections/%s/documents/import?action=upsert", c.BaseURL, collection),
		r,
	)
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("import failed: %s", buf.String())
	}

	return nil
}
