package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/4n4nd/cap-connectors-spotify/internal/store"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not found")
	ErrRateLimited  = errors.New("rate limited")
	ErrUpstream     = errors.New("upstream failure")
)

const (
	baseURL           = "https://api.spotify.com/v1"
	maxAttempts       = 2
	defaultRetryAfter = 2 * time.Second
)

type Client interface {
	ListDevices(ctx context.Context, userID string) ([]Device, error)
	TransferPlayback(ctx context.Context, userID, deviceID string, playNow bool) error
	AddToQueue(ctx context.Context, userID, deviceID, trackURI string) error
	SearchTracks(ctx context.Context, q string, limit int, market string) ([]Track, error)
	ResolveByISRC(ctx context.Context, isrc string) (*Track, error)
}

type client struct {
	http   *http.Client
	tokens store.TokenStore
}

func NewClient(tokens store.TokenStore) Client {
	return &client{
		http:   &http.Client{Timeout: 10 * time.Second},
		tokens: tokens,
	}
}

// --- Models ---

type Device struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsActive     bool   `json:"is_active"`
	IsRestricted bool   `json:"is_restricted"`
	Type         string `json:"type"`
}

type Track struct {
	ID         string   `json:"id"`
	URI        string   `json:"uri"`
	ISRC       string   `json:"isrc,omitempty"`
	Title      string   `json:"title"`
	Artists    []string `json:"artists"`
	DurationMS int      `json:"duration_ms"`
}

// --- API methods ---

func (c *client) ListDevices(ctx context.Context, userID string) ([]Device, error) {
	resp, err := c.do(ctx, requestConfig{
		method: http.MethodGet,
		url:    baseURL + "/me/player/devices",
		userID: userID,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	}
	if resp.StatusCode/100 != 2 {
		return nil, ErrUpstream
	}

	var payload struct {
		Devices []Device `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Devices, nil
}

func (c *client) TransferPlayback(ctx context.Context, userID, deviceID string, playNow bool) error {
	body := fmt.Sprintf(`{"device_ids":["%s"],"play":%v}`, deviceID, playNow)

	resp, err := c.do(ctx, requestConfig{
		method: http.MethodPut,
		url:    baseURL + "/me/player",
		body:   []byte(body),
		headers: map[string]string{
			"Content-Type": "application/json",
		},
		userID: userID,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if resp.StatusCode/100 == 2 {
			return nil
		}
		return ErrUpstream
	}
}

func (c *client) AddToQueue(ctx context.Context, userID, deviceID, trackURI string) error {
	u := baseURL + "/me/player/queue?uri=" + url.QueryEscape(trackURI)
	if deviceID != "" {
		q := url.Values{}
		q.Set("uri", trackURI)
		q.Set("device_id", deviceID)
		u = baseURL + "/me/player/queue?" + q.Encode()
	}

	resp, err := c.do(ctx, requestConfig{
		method: http.MethodPost,
		url:    u,
		userID: userID,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if resp.StatusCode/100 == 2 {
			return nil
		}
		return ErrUpstream
	}
}

func (c *client) SearchTracks(ctx context.Context, q string, limit int, market string) ([]Track, error) {
	v := url.Values{}
	v.Set("q", q)
	v.Set("type", "track")
	v.Set("limit", fmt.Sprintf("%d", limit))
	if market != "" {
		v.Set("market", market)
	}

	resp, err := c.do(ctx, requestConfig{
		method: http.MethodGet,
		url:    baseURL + "/search?" + v.Encode(),
		userID: "",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	}
	if resp.StatusCode/100 != 2 {
		return nil, ErrUpstream
	}

	var payload struct {
		Tracks struct {
			Items []struct {
				ID         string `json:"id"`
				URI        string `json:"uri"`
				DurationMS int    `json:"duration_ms"`
				Name       string `json:"name"`
				Artists    []struct {
					Name string `json:"name"`
				} `json:"artists"`
				ExternalIDs struct {
					ISRC string `json:"isrc"`
				} `json:"external_ids"`
			} `json:"items"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]Track, 0, len(payload.Tracks.Items))
	for _, it := range payload.Tracks.Items {
		names := make([]string, 0, len(it.Artists))
		for _, a := range it.Artists {
			names = append(names, a.Name)
		}
		out = append(out, Track{
			ID:        it.ID,
			URI:       it.URI,
			ISRC:      it.ExternalIDs.ISRC,
			Title:     it.Name,
			Artists:   names,
			DurationMS: it.DurationMS,
		})
	}
	return out, nil
}

func (c *client) ResolveByISRC(ctx context.Context, isrc string) (*Track, error) {
	q := "isrc:" + isrc
	items, err := c.SearchTracks(ctx, q, 1, "")
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &items[0], nil
}

func (c *client) attachAuth(req *http.Request, userID string) error {
	token := c.tokens.AccessToken(userID)
	if token == "" {
		return ErrUnauthorized
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

type requestConfig struct {
	method  string
	url     string
	body    []byte
	headers map[string]string
	userID  string
}

func (c *client) do(ctx context.Context, cfg requestConfig) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		var bodyReader io.Reader
		if cfg.body != nil {
			bodyReader = bytes.NewReader(cfg.body)
		}

		req, err := http.NewRequestWithContext(ctx, cfg.method, cfg.url, bodyReader)
		if err != nil {
			return nil, err
		}
		for k, v := range cfg.headers {
			req.Header.Set(k, v)
		}

		if err := c.attachAuth(req, cfg.userID); err != nil {
			return nil, err
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if attempt+1 == maxAttempts {
				break
			}
			if waitErr := wait(ctx, backoffDelay(attempt)); waitErr != nil {
				return nil, waitErr
			}
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt+1 < maxAttempts {
			delay := retryAfter(resp.Header.Get("Retry-After"))
			resp.Body.Close()
			if err := wait(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode >= 500 && resp.StatusCode <= 599 && attempt+1 < maxAttempts {
			resp.Body.Close()
			if err := wait(ctx, jitterDelay()); err != nil {
				return nil, err
			}
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func backoffDelay(attempt int) time.Duration {
	base := 500 * time.Millisecond
	return base * time.Duration(attempt+1)
}

func jitterDelay() time.Duration {
	min := 500 * time.Millisecond
	max := 2 * time.Second
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

func retryAfter(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return defaultRetryAfter
	}
	if secs, err := strconv.ParseFloat(header, 64); err == nil {
		if secs <= 0 {
			return defaultRetryAfter
		}
		return time.Duration(secs * float64(time.Second))
	}
	if ts, err := time.Parse(time.RFC1123, header); err == nil {
		d := time.Until(ts)
		if d > 0 {
			return d
		}
	}
	return defaultRetryAfter
}
