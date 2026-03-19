package pontomais

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func TestGetAccessToken(t *testing.T) {
	oldClient := httpClient
	httpClient = newTestClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/auth/sign_in" {
			t.Fatalf("path = %s, want /api/auth/sign_in", r.URL.Path)
		}
		if got := r.Header.Get("Api-Version"); got != "2" {
			t.Fatalf("Api-Version = %s, want 2", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %s, want application/json", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var req LoginRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		if req.Email != "user@example.com" || req.Password != "secret" {
			t.Fatalf("unexpected login payload: %+v", req)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"token":"abc","client_id":"client-1","data":{"login":"uid-1"}}`)),
		}, nil
	})
	defer func() {
		httpClient = oldClient
	}()

	response, err := getAccessTokenWithBaseURL(PontoMaisConfig{
		Email:    "user@example.com",
		Password: "secret",
	}, "https://example.test")
	if err != nil {
		t.Fatalf("GetAccessToken() error = %v", err)
	}

	if response.Token != "abc" {
		t.Fatalf("token = %s, want abc", response.Token)
	}

	if response.ClientID != "client-1" {
		t.Fatalf("client_id = %s, want client-1", response.ClientID)
	}

	if response.Data.Login != "uid-1" {
		t.Fatalf("login = %s, want uid-1", response.Data.Login)
	}
}

func TestGetAccessTokenInvalidJSON(t *testing.T) {
	oldClient := httpClient
	httpClient = newTestClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{invalid`)),
		}, nil
	})
	defer func() {
		httpClient = oldClient
	}()

	_, err := getAccessTokenWithBaseURL(PontoMaisConfig{
		Email:    "user@example.com",
		Password: "secret",
	}, "https://example.test")
	if err == nil || !strings.Contains(err.Error(), "erro ao parsear o JSON") {
		t.Fatalf("expected JSON parse error, got %v", err)
	}
}

func TestGetAccessTokenInvalidCredentials(t *testing.T) {
	oldClient := httpClient
	httpClient = newTestClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Status:     "401 Unauthorized",
			Body:       io.NopCloser(strings.NewReader(`{"errors":["Invalid login credentials. Please try again."]}`)),
		}, nil
	})
	defer func() {
		httpClient = oldClient
	}()

	_, err := getAccessTokenWithBaseURL(PontoMaisConfig{
		Email:    "user@example.com",
		Password: "wrong",
	}, "https://example.test")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestGetWorkDays(t *testing.T) {
	oldClient := httpClient
	httpClient = newTestClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/time_card_control/current/work_days" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("start_date"); got != "2026-03-01" {
			t.Fatalf("start_date = %s, want 2026-03-01", got)
		}
		if got := r.URL.Query().Get("end_date"); got != "2026-03-31" {
			t.Fatalf("end_date = %s, want 2026-03-31", got)
		}
		if got := r.URL.Query().Get("sort_direction"); got != "desc" {
			t.Fatalf("sort_direction = %s, want desc", got)
		}
		if got := r.Header.Get("Access-Token"); got != "access" {
			t.Fatalf("Access-Token = %s, want access", got)
		}
		if got := r.Header.Get("Token"); got != "token" {
			t.Fatalf("Token = %s, want token", got)
		}
		if got := r.Header.Get("Uid"); got != "uid" {
			t.Fatalf("Uid = %s, want uid", got)
		}
		if got := r.Header.Get("Client"); got != "client" {
			t.Fatalf("Client = %s, want client", got)
		}
		if got := r.Header.Get("Uuid"); got != "uuid-1" {
			t.Fatalf("Uuid = %s, want uuid-1", got)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"work_days":[{"id":7,"date":"2026-03-10","status":{"id":1,"name":"Falta"}}]}`)),
		}, nil
	})
	defer func() {
		httpClient = oldClient
	}()

	start := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	workDays, err := getWorkDaysWithBaseURL(PontoMaisConfig{
		AccessToken: "access",
		Token:       "token",
		Uid:         "uid",
		Client:      "client",
		Uuid:        "uuid-1",
	}, start, end, "https://example.test")
	if err != nil {
		t.Fatalf("GetWorkDays() error = %v", err)
	}

	if len(workDays) != 1 || workDays[0].ID != 7 {
		t.Fatalf("workDays = %+v, want one workday with ID 7", workDays)
	}
}

func TestGetWorkDaysInvalidJSON(t *testing.T) {
	oldClient := httpClient
	httpClient = newTestClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`not-json`)),
		}, nil
	})
	defer func() {
		httpClient = oldClient
	}()

	_, err := getWorkDaysWithBaseURL(PontoMaisConfig{}, time.Now(), time.Now(), "https://example.test")
	if err == nil || !strings.Contains(err.Error(), "erro ao parsear o JSON") {
		t.Fatalf("expected JSON parse error, got %v", err)
	}
}

func TestAjustarPonto(t *testing.T) {
	oldClient := httpClient
	httpClient = newTestClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/time_cards/proposals" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Origin"); got != "https://app2.pontomais.com.br" {
			t.Fatalf("Origin = %s", got)
		}
		if got := r.Header.Get("Referer"); got != "https://app2.pontomais.com.br/" {
			t.Fatalf("Referer = %s", got)
		}
		if got := r.Header.Get("Uuid"); got != "764c0af2-a116-4075-9d7a-12c3675f840e" {
			t.Fatalf("Uuid = %s", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var req AjustePontoRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		if req.Proposal.Date != "2026-03-19" {
			t.Fatalf("proposal date = %s", req.Proposal.Date)
		}

		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})
	defer func() {
		httpClient = oldClient
	}()

	err := ajustarPontoWithBaseURL(PontoMaisConfig{
		AccessToken: "access",
		Token:       "token",
		Uid:         "uid",
		Client:      "client",
	}, AjustePontoRequest{
		Proposal: Proposal{Date: "2026-03-19"},
	}, "https://example.test")
	if err != nil {
		t.Fatalf("AjustarPonto() error = %v", err)
	}
}

func TestAjustarPontoNonCreatedStatus(t *testing.T) {
	oldClient := httpClient
	httpClient = newTestClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(strings.NewReader(`{"error":"bad request"}`)),
		}, nil
	})
	defer func() {
		httpClient = oldClient
	}()

	err := ajustarPontoWithBaseURL(PontoMaisConfig{}, AjustePontoRequest{}, "https://example.test")
	if err == nil || !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("expected HTTP status error, got %v", err)
	}
}
