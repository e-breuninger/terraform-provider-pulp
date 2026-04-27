// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type PulpClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
}

const taskPollingInterval = 2 * time.Second

func NewPulpClient(baseURL, username, password string) *PulpClient {
	return &PulpClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Username: username,
		Password: password,
	}
}

func (c *PulpClient) doRequest(ctx context.Context, method, url string, body map[string]any) (map[string]any, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	// No content (e.g., 204 on delete)
	if len(respBody) == 0 {
		return nil, resp.StatusCode, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to unmarshal response: %w (body: %s)", err, string(respBody))
	}

	return result, resp.StatusCode, nil
}

// BuildResourcePath constructs paths like "remotes/rpm/rpm" or "remotes/deb/apt".
func BuildResourcePath(resourceType, contentType, pluginName string) string {
	if pluginName == "" {
		pluginName = contentType
	}
	return fmt.Sprintf("%s/%s/%s", resourceType, contentType, pluginName)
}

func (c *PulpClient) Create(ctx context.Context, resourcePath string, body map[string]any) (map[string]any, error) {
	url := fmt.Sprintf("%s/pulp/api/v3/%s/", c.BaseURL, resourcePath)
	result, statusCode, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	// Handle async task response, e.g., Distributions
	if taskHref, ok := result["task"].(string); ok {
		if err := c.WaitForTask(ctx, taskHref); err != nil {
			return nil, err
		}
		// Re-fetch created resource from task's created_resources
		task, _, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("%s%s", c.BaseURL, taskHref), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch task result: %w", err)
		}
		if created, ok := task["created_resources"].([]interface{}); ok && len(created) > 0 {
			href, ok := created[0].(string)
			if !ok {
				return nil, fmt.Errorf("expected href to be a string, got %T", created[0])
			}
			return c.ReadByHref(ctx, href)
		}
	}

	if statusCode != http.StatusCreated && statusCode != http.StatusOK {
		return nil, fmt.Errorf("create failed with status %d: %v", statusCode, result)
	}

	return result, nil
}

func (c *PulpClient) ReadByHref(ctx context.Context, pulpHref string) (map[string]any, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, pulpHref)
	result, statusCode, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil // signal that resource is gone
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("read failed with status %d: %v", statusCode, result)
	}
	return result, nil
}

func (c *PulpClient) Update(ctx context.Context, pulpHref string, body map[string]any) (map[string]any, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, pulpHref)
	result, statusCode, err := c.doRequest(ctx, http.MethodPatch, url, body)
	if err != nil {
		return nil, err
	}

	// Handle async task
	if taskHref, ok := result["task"].(string); ok {
		if err := c.WaitForTask(ctx, taskHref); err != nil {
			return nil, err
		}
		return c.ReadByHref(ctx, pulpHref)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("update failed with status %d: %v", statusCode, result)
	}
	return result, nil
}

func (c *PulpClient) Delete(ctx context.Context, pulpHref string) error {
	url := fmt.Sprintf("%s%s", c.BaseURL, pulpHref)
	result, statusCode, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	// Handle async task
	if result != nil {
		if taskHref, ok := result["task"].(string); ok {
			return c.WaitForTask(ctx, taskHref)
		}
	}

	if statusCode == http.StatusNotFound {
		return nil
	}

	if statusCode != http.StatusNoContent {
		return fmt.Errorf("delete failed with status %d: %v", statusCode, result)
	}
	return nil
}

func (c *PulpClient) CallHrefAction(ctx context.Context, href, action string, body map[string]any) (map[string]any, int, error) {
	resourcePath := strings.TrimRight(href, "/") + "/" + strings.Trim(action, "/") + "/"
	url := fmt.Sprintf("%s%s", c.BaseURL, resourcePath)
	return c.doRequest(ctx, http.MethodPost, url, body)
}

// ListHrefAction GETs {href}{action}/ and returns the decoded body.
func (c *PulpClient) ListHrefAction(ctx context.Context, href, action string) (map[string]any, int, error) {
	resourcePath := strings.TrimRight(href, "/") + "/" + strings.Trim(action, "/") + "/"
	url := fmt.Sprintf("%s%s", c.BaseURL, resourcePath)
	return c.doRequest(ctx, http.MethodGet, url, nil)
}

func (c *PulpClient) WaitForTask(ctx context.Context, taskHref string) error {
	url := fmt.Sprintf("%s%s", c.BaseURL, taskHref)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, _, err := c.doRequest(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to poll task: %w", err)
		}

		state, _ := result["state"].(string)
		switch state {
		case "completed":
			return nil
		case "failed":
			return fmt.Errorf("task %s failed: %v", taskHref, result["error"])
		case "canceled", "cancelled":
			return fmt.Errorf("task %s was canceled", taskHref)
		}

		time.Sleep(taskPollingInterval)
	}
}
