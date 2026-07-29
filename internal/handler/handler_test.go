package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/venexene/wbl3-delayed-notifier/internal/storage"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockStore struct {
	createFn  func(ctx context.Context, n storage.Notification) error
	getByIDFn func(ctx context.Context, id string) (*storage.Notification, error)
	cancelFn  func(ctx context.Context, id string) (bool, error)
}

func (m *mockStore) Create(ctx context.Context, n storage.Notification) error {
	return m.createFn(ctx, n)
}

func (m *mockStore) GetByID(ctx context.Context, id string) (*storage.Notification, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockStore) Cancel(ctx context.Context, id string) (bool, error) {
	return m.cancelFn(ctx, id)
}

type mockPublisher struct {
	publishFn func(n storage.Notification) error
}

func (m *mockPublisher) Publish(n storage.Notification) error {
	return m.publishFn(n)
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func makeJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return b
}

func TestHealthCheck(t *testing.T) {
	router := newTestRouter()
	router.GET("/test_server", HealthCheck)

	req := httptest.NewRequest(http.MethodGet, "/test_server", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if body := w.Body.String(); body == "" {
		t.Error("expected non-empty response body")
	}
}

func TestCreateNotification_Success(t *testing.T) {
	store := &mockStore{
		createFn: func(ctx context.Context, n storage.Notification) error {
			if n.ID == "" {
				t.Error("expected non-empty ID")
			}
			if n.Status != "pending" {
				t.Errorf("expected status 'pending', got %q", n.Status)
			}
			return nil
		},
	}
	pub := &mockPublisher{
		publishFn: func(n storage.Notification) error {
			return nil
		},
	}

	router := newTestRouter()
	router.POST("/notify", CreateNotification(store, pub))

	body := makeJSON(t, map[string]string{
		"target":  "user@test.com",
		"message": "Test notification",
		"send_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})

	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["id"] == "" {
		t.Error("expected non-empty id in response")
	}
}

func TestCreateNotification_BadJSON(t *testing.T) {
	store := &mockStore{}
	pub := &mockPublisher{}

	router := newTestRouter()
	router.POST("/notify", CreateNotification(store, pub))

	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateNotification_DBError(t *testing.T) {
	store := &mockStore{
		createFn: func(ctx context.Context, n storage.Notification) error {
			return errors.New("db error")
		},
	}
	pub := &mockPublisher{}

	router := newTestRouter()
	router.POST("/notify", CreateNotification(store, pub))

	body := makeJSON(t, map[string]string{
		"target":  "user@test.com",
		"message": "Test",
		"send_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})

	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestCreateNotification_QueueError(t *testing.T) {
	store := &mockStore{
		createFn: func(ctx context.Context, n storage.Notification) error {
			return nil
		},
	}
	pub := &mockPublisher{
		publishFn: func(n storage.Notification) error {
			return errors.New("queue error")
		},
	}

	router := newTestRouter()
	router.POST("/notify", CreateNotification(store, pub))

	body := makeJSON(t, map[string]string{
		"target":  "user@test.com",
		"message": "Test",
		"send_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})

	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestGetNotificationStatus_Found(t *testing.T) {
	expected := &storage.Notification{
		ID:      "abc-123",
		Target:  "user@test.com",
		Message: "Hello",
		Status:  "pending",
	}

	store := &mockStore{
		getByIDFn: func(ctx context.Context, id string) (*storage.Notification, error) {
			if id != "abc-123" {
				t.Errorf("expected id 'abc-123', got %q", id)
			}
			return expected, nil
		},
	}

	router := newTestRouter()
	router.GET("/notify/:id", GetNotificationStatus(store))

	req := httptest.NewRequest(http.MethodGet, "/notify/abc-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var got storage.Notification
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected ID %q, got %q", expected.ID, got.ID)
	}
	if got.Status != expected.Status {
		t.Errorf("expected status %q, got %q", expected.Status, got.Status)
	}
}

func TestGetNotificationStatus_NotFound(t *testing.T) {
	store := &mockStore{
		getByIDFn: func(ctx context.Context, id string) (*storage.Notification, error) {
			return nil, errors.New("not found")
		},
	}

	router := newTestRouter()
	router.GET("/notify/:id", GetNotificationStatus(store))

	req := httptest.NewRequest(http.MethodGet, "/notify/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestCancelNotification_Success(t *testing.T) {
	store := &mockStore{
		cancelFn: func(ctx context.Context, id string) (bool, error) {
			if id != "abc-123" {
				t.Errorf("expected id 'abc-123', got %q", id)
			}
			return true, nil
		},
	}

	router := newTestRouter()
	router.DELETE("/notify/:id", CancelNotification(store))

	req := httptest.NewRequest(http.MethodDelete, "/notify/abc-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["status"] != "canceled" {
		t.Errorf("expected status 'canceled', got %q", resp["status"])
	}
}

func TestCancelNotification_CannotCancel(t *testing.T) {
	store := &mockStore{
		cancelFn: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
	}

	router := newTestRouter()
	router.DELETE("/notify/:id", CancelNotification(store))

	req := httptest.NewRequest(http.MethodDelete, "/notify/already-sent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCancelNotification_DBError(t *testing.T) {
	store := &mockStore{
		cancelFn: func(ctx context.Context, id string) (bool, error) {
			return false, errors.New("db error")
		},
	}

	router := newTestRouter()
	router.DELETE("/notify/:id", CancelNotification(store))

	req := httptest.NewRequest(http.MethodDelete, "/notify/abc-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
