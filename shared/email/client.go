package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Pass1   string
	Pass2   string
	HTTP    *http.Client
}

func NewClient(baseURL, pass1, pass2 string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		BaseURL: baseURL,
		Pass1:   pass1,
		Pass2:   pass2,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) Send(ctx context.Context, to, subject, message string) error {
	if c == nil || c.BaseURL == "" {
		return nil
	}

	payload := map[string]string{
		"email":   to,
		"subject": subject,
		"message": message,
		"pass1":   c.Pass1,
		"pass2":   c.Pass2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("email API returned status %d", resp.StatusCode)
	}
	return nil
}
