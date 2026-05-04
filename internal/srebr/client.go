// Package srebr provides a Jira HTTP client for SREBR ticket operations.
package srebr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/symphony/srebr-bg/internal/config"
)

// Client is a minimal Jira REST API v2 client.
type Client struct {
	baseURL string
	auth    string // "Basic <base64>"
	http    *http.Client
}

// NewClient creates a Client from the provided credentials.
func NewClient(creds config.Creds) *Client {
	raw := creds.User + ":" + creds.Token
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
	return &Client{
		baseURL: creds.BaseURL,
		auth:    auth,
		http:    http.DefaultClient,
	}
}

// IssueFields is a lightweight representation of a Jira issue's fields.
type IssueFields struct {
	Summary     string `json:"summary"`
	Description string `json:"description"`
	IssueType   struct {
		Name string `json:"name"`
	} `json:"issuetype"`
	Status struct {
		Name string `json:"name"`
	} `json:"status"`
	Assignee *struct {
		DisplayName string `json:"displayName"`
	} `json:"assignee"`
}

// Issue is a single Jira issue.
type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

// Comment is a single Jira issue comment.
type Comment struct {
	ID      string `json:"id"`
	Created string `json:"created"`
	Author  struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Body string `json:"body"`
}

// GetComments returns all comments for an issue.
func (c *Client) GetComments(key string) ([]Comment, error) {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s/comment?maxResults=100&orderBy=created", c.baseURL, key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JIRA API error (%d): %s", resp.StatusCode, body)
	}
	var r struct {
		Comments []Comment `json:"comments"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse comments: %w", err)
	}
	return r.Comments, nil
}

// GetIssue fetches an issue by key.
func (c *Client) GetIssue(key string) (*Issue, error) {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=summary,description,issuetype,status,assignee", c.baseURL, key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JIRA API error (%d): %s", resp.StatusCode, body)
	}
	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parse issue: %w", err)
	}
	return &issue, nil
}

// CreateIssue creates a new issue and returns its key.
func (c *Client) CreateIssue(fields map[string]any) (string, error) {
	payload := map[string]any{"fields": fields}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf("%s/rest/api/2/issue", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("JIRA API error (%d): %s", resp.StatusCode, body)
	}
	var r struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return r.Key, nil
}

// LinkIssues creates a link of the given type between two issues.
func (c *Client) LinkIssues(linkType, inwardKey, outwardKey string) error {
	payload := map[string]any{
		"type":         map[string]any{"name": linkType},
		"inwardIssue":  map[string]any{"key": inwardKey},
		"outwardIssue": map[string]any{"key": outwardKey},
	}
	data, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/rest/api/2/issueLink", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("JIRA API error (%d): %s", resp.StatusCode, body)
	}
	return nil
}

// Transition moves an issue to the named transition (e.g. "Approve").
func (c *Client) Transition(key, transitionName string) error {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s/transitions", c.baseURL, key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JIRA API error (%d): %s", resp.StatusCode, body)
	}

	var tr struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"transitions"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return fmt.Errorf("parse transitions: %w", err)
	}

	var id string
	for _, t := range tr.Transitions {
		if equalFold(t.Name, transitionName) {
			id = t.ID
			break
		}
	}
	if id == "" {
		return fmt.Errorf("transition %q not found", transitionName)
	}

	payload := map[string]any{"transition": map[string]any{"id": id}}
	data, _ := json.Marshal(payload)
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("JIRA API error (%d): %s", resp.StatusCode, body)
	}
	return nil
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
