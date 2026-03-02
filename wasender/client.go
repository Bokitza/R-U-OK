package wasender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://www.wasenderapi.com"

type Client struct {
	apiKey string
	http   *http.Client
}

type GroupMetadata struct {
	JID          string                `json:"jid"`
	Subject      string                `json:"subject"`
	Participants []MetadataParticipant `json:"participants"`
}

type MetadataParticipant struct {
	JID          string `json:"jid"`
	IsAdmin      bool   `json:"isAdmin"`
	IsSuperAdmin bool   `json:"isSuperAdmin"`
}

type GroupParticipant struct {
	ID    string `json:"id"`
	Admin string `json:"admin,omitempty"`
}

type SendMessageRequest struct {
	To       string   `json:"to"`
	Text     string   `json:"text"`
	Mentions []string `json:"mentions,omitempty"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) GetGroupMetadata(ctx context.Context, groupJID string) (*GroupMetadata, error) {
	var result struct {
		Success bool          `json:"success"`
		Data    GroupMetadata `json:"data"`
	}
	if err := c.doGet(ctx, fmt.Sprintf("%s/api/groups/%s/metadata", baseURL, url.PathEscape(groupJID)), &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *Client) GetGroupParticipants(ctx context.Context, groupJID string) ([]GroupParticipant, error) {
	var result struct {
		Success bool               `json:"success"`
		Data    []GroupParticipant `json:"data"`
	}
	if err := c.doGet(ctx, fmt.Sprintf("%s/api/groups/%s/participants", baseURL, url.PathEscape(groupJID)), &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) ResolvePhoneFromLID(ctx context.Context, lid string) (string, error) {
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			PN string `json:"pn"`
		} `json:"data"`
	}
	if err := c.doGet(ctx, fmt.Sprintf("%s/api/pn-from-lid/%s", baseURL, url.PathEscape(lid)), &result); err != nil {
		return "", err
	}
	return result.Data.PN, nil
}

func (c *Client) SendMessage(ctx context.Context, req SendMessageRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/send-message", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send message failed (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *Client) doGet(ctx context.Context, url string, result any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
