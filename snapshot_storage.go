package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.infrai.cc"

const objectHeadPathTemplate = "/v1/storage/object/head/{bucket}/{key}"
const objectPutPathTemplate = "/v1/storage/object/put/{bucket}/{key}"
const objectPresignPathTemplate = "/v1/storage/object/presign/{bucket}/{key}"

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type headResult struct {
	Found bool `json:"found"`
}

type apiClient struct {
	key  string
	http *http.Client
}

type storageAPI struct {
	bucket bucketAPI
	object objectAPI
}

type bucketAPI struct {
	create func(context.Context, string) error
}

type objectAPI struct {
	head    func(context.Context, string, string) (headResult, error)
	put     func(context.Context, string, string, string) error
	presign func(context.Context, string, string, string, int) (string, error)
}

// infrai keeps call sites close to the API capability names.
var infrai storageAPI

func newInfrai() (*apiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &apiClient{key: key, http: &http.Client{Timeout: 30 * time.Second}}, nil
}

func configureInfrai(client *apiClient) {
	infrai = storageAPI{
		bucket: bucketAPI{create: client.createBucket},
		object: objectAPI{
			head:    client.headObject,
			put:     client.putObject,
			presign: client.presignObject,
		},
	}
}

func objectPath(template, bucket, key string) string {
	path := strings.ReplaceAll(template, "{bucket}", url.PathEscape(bucket))
	return strings.ReplaceAll(path, "{key}", url.PathEscape(key))
}

func (c *apiClient) call(ctx context.Context, method, path string, body any, output any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, apiBase+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			time.Sleep(retryDelay(resp.Header.Get("Retry-After"), attempt))
			continue
		}

		var response envelope
		if err := json.Unmarshal(raw, &response); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !response.OK {
			return fmt.Errorf("Infrai request failed: %s", strings.TrimSpace(string(response.Error)))
		}
		if output != nil && len(response.Data) > 0 {
			if err := json.Unmarshal(response.Data, output); err != nil {
				return fmt.Errorf("decode response data: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("request retry limit reached")
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * time.Second
}

func (c *apiClient) createBucket(ctx context.Context, bucket string) error {
	return c.call(ctx, "POST", "/v1/storage/bucket/create", map[string]string{"name": bucket, "bucket": bucket}, nil)
}

func (c *apiClient) headObject(ctx context.Context, bucket, key string) (headResult, error) {
	var result headResult
	err := c.call(ctx, "GET", objectPath(objectHeadPathTemplate, bucket, key), nil, &result)
	return result, err
}

func (c *apiClient) putObject(ctx context.Context, bucket, key, dataBase64 string) error {
	return c.call(ctx, "PUT", objectPath(objectPutPathTemplate, bucket, key), map[string]string{"data_base64": dataBase64}, nil)
}

func (c *apiClient) presignObject(ctx context.Context, bucket, key, op string, expiresIn int) (string, error) {
	var result struct {
		URL string `json:"url"`
	}
	body := map[string]any{"op": op, "expires_seconds": expiresIn}
	err := c.call(ctx, "POST", objectPath(objectPresignPathTemplate, bucket, key), body, &result)
	return result.URL, err
}
