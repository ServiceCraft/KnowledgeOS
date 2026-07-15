package yclients

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetServices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/book_services/123" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer partner-token" {
			t.Errorf("unexpected auth header %q", got)
		}
		if got := r.Header.Get("Accept"); got != acceptHeader {
			t.Errorf("unexpected accept header %q", got)
		}
		_, _ = io.WriteString(w, `{"success":true,"data":{"services":[{"id":1,"title":"Стрижка","price_min":500,"price_max":1000,"seance_length":3600}]}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{PartnerToken: "partner-token", BaseURL: srv.URL})
	services, err := c.GetServices(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetServices: %v", err)
	}
	if len(services) != 1 || services[0].ID != 1 || services[0].Title != "Стрижка" || services[0].SeanceLength != 3600 {
		t.Fatalf("unexpected services: %+v", services)
	}
}

func TestCreateBookRecord(t *testing.T) {
	var body BookRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/book_record/55" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = io.WriteString(w, `{"success":true,"data":[{"record_id":42,"record_hash":"abc"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{PartnerToken: "t", BaseURL: srv.URL})
	res, err := c.CreateBookRecord(context.Background(), "55", BookRequest{
		Phone:        "79991234567",
		Fullname:     "Иван",
		Appointments: []Appointment{{ID: 1, Services: []int{7}, StaffID: 3, Datetime: "2026-07-10T10:00:00+03:00"}},
	})
	if err != nil {
		t.Fatalf("CreateBookRecord: %v", err)
	}
	if len(res) != 1 || res[0].RecordID != 42 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if body.Phone != "79991234567" || len(body.Appointments) != 1 || body.Appointments[0].StaffID != 3 {
		t.Fatalf("server received unexpected body: %+v", body)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"success":false,"meta":{"message":"Время уже занято"}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), Config{PartnerToken: "t", BaseURL: srv.URL})
	_, err := c.GetBookTimes(context.Background(), "1", 2, "2026-07-10")
	if err == nil || !strings.Contains(err.Error(), "Время уже занято") {
		t.Fatalf("expected api error message, got %v", err)
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"+7 (999) 123-45-67": "79991234567",
		"8 999 123 45 67":    "79991234567",
		"79991234567":        "79991234567",
		"":                   "",
	}
	for in, want := range cases {
		if got := NormalizePhone(in); got != want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}
