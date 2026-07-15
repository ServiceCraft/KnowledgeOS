// Package yclients is a thin HTTP client for the YCLIENTS online-booking API
// (https://developers.yclients.com/). It covers the online-booking flow used by
// the bot to register clients: listing bookable services and staff, fetching
// free time slots and creating a booking via book_record.
package yclients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the public YCLIENTS REST API root.
	DefaultBaseURL = "https://api.yclients.com/api/v1"
	// acceptHeader pins the API to the documented response format version.
	acceptHeader         = "application/vnd.api.v2+json"
	defaultTimeout       = 15 * time.Second
	maxResponseBodyBytes = 4 << 20 // 4 MiB safety cap when buffering responses
)

// Config configures a Client. Only PartnerToken is required; BaseURL and Timeout
// fall back to sane defaults.
type Config struct {
	PartnerToken string
	BaseURL      string
	Timeout      time.Duration
}

// Client talks to the YCLIENTS API on behalf of a single company using a partner
// (bearer) token. The online-booking endpoints do not require a per-user token.
type Client struct {
	http         *http.Client
	baseURL      string
	partnerToken string
}

// NewClient builds a Client. A nil httpClient gets a default one with the
// configured (or default) timeout.
func NewClient(httpClient *http.Client, cfg Config) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{http: httpClient, baseURL: baseURL, partnerToken: strings.TrimSpace(cfg.PartnerToken)}
}

// Service is a bookable service offered by a company.
type Service struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	PriceMin     float64 `json:"price_min"`
	PriceMax     float64 `json:"price_max"`
	SeanceLength int     `json:"seance_length"` // duration in seconds
}

// Staff is a bookable specialist.
type Staff struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Specialization string `json:"specialization"`
	Bookable       bool   `json:"bookable"`
}

// Slot is a single free time slot for a staff member on a given day.
type Slot struct {
	Time     string `json:"time"`     // "10:00"
	Datetime string `json:"datetime"` // ISO8601 with timezone
}

// Appointment is one line of a booking request: which services with which staff
// at which time. ID is an arbitrary client-side reference echoed back in errors.
type Appointment struct {
	ID       int    `json:"id"`
	Services []int  `json:"services"`
	StaffID  int    `json:"staff_id"`
	Datetime string `json:"datetime"`
}

// BookRequest is the body of a book_record call.
type BookRequest struct {
	Phone        string        `json:"phone"`
	Fullname     string        `json:"fullname"`
	Email        string        `json:"email,omitempty"`
	Comment      string        `json:"comment,omitempty"`
	APIID        string        `json:"api_id,omitempty"`
	Appointments []Appointment `json:"appointments"`
}

// BookResult identifies a created record.
type BookResult struct {
	RecordID   int    `json:"record_id"`
	RecordHash string `json:"record_hash"`
}

// envelope is the standard YCLIENTS response wrapper.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Meta    json.RawMessage `json:"meta"`
}

// GetServices lists services bookable online for the company.
func (c *Client) GetServices(ctx context.Context, companyID string) ([]Service, error) {
	data, err := c.get(ctx, fmt.Sprintf("book_services/%s", url.PathEscape(companyID)), nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Services []Service `json:"services"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode services: %w", err)
	}
	return payload.Services, nil
}

// GetStaff lists specialists bookable online, optionally filtered by services.
func (c *Client) GetStaff(ctx context.Context, companyID string, serviceIDs []int) ([]Staff, error) {
	q := url.Values{}
	for _, id := range serviceIDs {
		q.Add("service_ids[]", strconv.Itoa(id))
	}
	data, err := c.get(ctx, fmt.Sprintf("book_staff/%s", url.PathEscape(companyID)), q)
	if err != nil {
		return nil, err
	}
	var staff []Staff
	if err := json.Unmarshal(data, &staff); err != nil {
		return nil, fmt.Errorf("decode staff: %w", err)
	}
	return staff, nil
}

// GetBookTimes returns free slots for a staff member on the given day (YYYY-MM-DD).
func (c *Client) GetBookTimes(ctx context.Context, companyID string, staffID int, date string) ([]Slot, error) {
	path := fmt.Sprintf("book_times/%s/%d/%s", url.PathEscape(companyID), staffID, url.PathEscape(date))
	data, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	var slots []Slot
	if err := json.Unmarshal(data, &slots); err != nil {
		return nil, fmt.Errorf("decode times: %w", err)
	}
	return slots, nil
}

// CreateBookRecord creates an online booking and returns the created records.
func (c *Client) CreateBookRecord(ctx context.Context, companyID string, req BookRequest) ([]BookResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode book request: %w", err)
	}
	data, err := c.post(ctx, fmt.Sprintf("book_record/%s", url.PathEscape(companyID)), body)
	if err != nil {
		return nil, err
	}
	var results []BookResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("decode book result: %w", err)
	}
	return results, nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	u := c.baseURL + "/" + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) post(ctx context.Context, path string, body []byte) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) (json.RawMessage, error) {
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Authorization", "Bearer "+c.partnerToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return nil, err
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("yclients: unexpected response (status %d)", resp.StatusCode)
	}
	if resp.StatusCode >= http.StatusBadRequest || !env.Success {
		return nil, fmt.Errorf("yclients: %s", apiErrorMessage(env, resp.StatusCode))
	}
	return env.Data, nil
}

// apiErrorMessage extracts a human-readable message from the response meta.
func apiErrorMessage(env envelope, status int) string {
	var meta struct {
		Message string `json:"message"`
	}
	if len(env.Meta) > 0 {
		_ = json.Unmarshal(env.Meta, &meta)
	}
	if strings.TrimSpace(meta.Message) != "" {
		return meta.Message
	}
	return fmt.Sprintf("request failed (status %d)", status)
}

// NormalizePhone reduces a phone number to bare digits in the 7XXXXXXXXXX form
// expected by YCLIENTS. A leading 8 on an 11-digit number is rewritten to 7.
func NormalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if len(digits) == 11 && digits[0] == '8' {
		return "7" + digits[1:]
	}
	return digits
}
