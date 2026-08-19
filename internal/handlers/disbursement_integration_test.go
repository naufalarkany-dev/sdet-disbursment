package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/disbursement/internal/middleware"
	"example.com/disbursement/internal/models"
	"example.com/disbursement/internal/repository/memory"
	"example.com/disbursement/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const integrationJWTSecret = "test-secret-key"

type integrationEnvironment struct {
	router *gin.Engine
	repo   *memory.DisbursementRepository
}

type apiResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    json.RawMessage        `json:"data"`
	Meta    map[string]json.Number `json:"meta"`
}

func newIntegrationEnvironment(t *testing.T) integrationEnvironment {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := memory.NewDisbursementRepository()
	svc := services.NewDisbursementService(repo)
	handler := NewHandler(svc)
	router := gin.New()
	router.Use(gin.Recovery())
	auth := router.Group("/")
	auth.Use(middleware.JWTAuth(integrationJWTSecret))
	auth.POST("/disbursements", handler.CreateDisbursement)
	auth.PATCH("/disbursements/:id/status", handler.UpdateDisbursementStatus)
	auth.GET("/disbursements", handler.ListDisbursements)
	return integrationEnvironment{router: router, repo: repo}
}

func authToken(t *testing.T, username, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  username,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(integrationJWTSecret))
	require.NoError(t, err)
	return signed
}

func performJSONRequest(t *testing.T, router http.Handler, method, path, body, role string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken(t, role+"-user", role))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var response apiResponse
	decoder := json.NewDecoder(recorder.Body)
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&response), "response body: %s", recorder.Body.String())
	return response
}

func TestCreateDisbursementHTTP(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "valid_request",
			body:        `{"recipient_name":"Budi","account_number":"123","bank_code":"BCA","amount":5000000}`,
			wantStatus:  http.StatusCreated,
			wantMessage: "berhasil dibuat",
		},
		{
			name:        "missing_recipient_is_informative",
			body:        `{"account_number":"123","bank_code":"BCA","amount":5000000}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "recipient_name",
		},
		{
			name:        "amount_below_minimum",
			body:        `{"recipient_name":"Budi","account_number":"123","bank_code":"BCA","amount":9999}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantMessage: "amount must be a positive integer >= 10000",
		},
		{
			name:        "malformed_json",
			body:        `{"recipient_name":`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "request body tidak valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newIntegrationEnvironment(t)
			recorder := performJSONRequest(t, env.router, http.MethodPost, "/disbursements", tt.body, "operator")

			require.Equal(t, tt.wantStatus, recorder.Code)
			response := decodeResponse(t, recorder)
			assert.Contains(t, response.Message, tt.wantMessage)
			if tt.wantStatus >= 400 {
				assert.False(t, response.Success)
			}
		})
	}
}

func TestUpdateDisbursementStatusHTTP(t *testing.T) {
	t.Run("pending_to_approved", func(t *testing.T) {
		env := newIntegrationEnvironment(t)
		require.NoError(t, env.repo.Create(&models.Disbursement{ID: "DSB-1", Status: models.StatusPending}))

		recorder := performJSONRequest(t, env.router, http.MethodPatch, "/disbursements/DSB-1/status", `{"status":"APPROVED"}`, "admin")

		assert.Equal(t, http.StatusOK, recorder.Code)
		response := decodeResponse(t, recorder)
		var got models.Disbursement
		require.NoError(t, json.Unmarshal(response.Data, &got))
		assert.Equal(t, models.StatusApproved, got.Status)
		assert.Equal(t, "admin-user", got.ApprovedBy)
	})

	t.Run("missing_id", func(t *testing.T) {
		env := newIntegrationEnvironment(t)
		recorder := performJSONRequest(t, env.router, http.MethodPatch, "/disbursements/missing/status", `{"status":"APPROVED"}`, "admin")

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Contains(t, decodeResponse(t, recorder).Message, "not found")
	})

	t.Run("repeated_approval", func(t *testing.T) {
		env := newIntegrationEnvironment(t)
		require.NoError(t, env.repo.Create(&models.Disbursement{ID: "DSB-1", Status: models.StatusPending}))
		first := performJSONRequest(t, env.router, http.MethodPatch, "/disbursements/DSB-1/status", `{"status":"APPROVED"}`, "admin")
		require.Equal(t, http.StatusOK, first.Code)

		second := performJSONRequest(t, env.router, http.MethodPatch, "/disbursements/DSB-1/status", `{"status":"APPROVED"}`, "admin")

		assert.Equal(t, http.StatusUnprocessableEntity, second.Code)
		assert.Contains(t, decodeResponse(t, second).Message, "already in final state")
	})

	t.Run("non_admin", func(t *testing.T) {
		env := newIntegrationEnvironment(t)
		require.NoError(t, env.repo.Create(&models.Disbursement{ID: "DSB-1", Status: models.StatusPending}))
		recorder := performJSONRequest(t, env.router, http.MethodPatch, "/disbursements/DSB-1/status", `{"status":"APPROVED"}`, "operator")

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, decodeResponse(t, recorder).Message, "hanya admin")
	})
}

func seedListData(t *testing.T, repo *memory.DisbursementRepository) {
	t.Helper()
	items := []*models.Disbursement{
		{ID: "DSB-1", RecipientName: "Budi Santoso", Status: models.StatusPending},
		{ID: "DSB-2", RecipientName: "Siti Aminah", Status: models.StatusApproved},
		{ID: "DSB-3", RecipientName: "Budi Hartono", Status: models.StatusRejected},
	}
	for _, item := range items {
		require.NoError(t, repo.Create(item))
	}
}

func TestListDisbursementsHTTP(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCount  int
		wantTotal  int64
		wantPage   int64
		wantLimit  int64
		wantPages  int64
		wantError  string
	}{
		{name: "default_pagination", wantStatus: http.StatusOK, wantCount: 3, wantTotal: 3, wantPage: 1, wantLimit: 10, wantPages: 1},
		{name: "pending_status_filter", query: "?status=PENDING", wantStatus: http.StatusOK, wantCount: 1, wantTotal: 1, wantPage: 1, wantLimit: 10, wantPages: 1},
		{name: "recipient_search", query: "?search=Budi", wantStatus: http.StatusOK, wantCount: 2, wantTotal: 2, wantPage: 1, wantLimit: 10, wantPages: 1},
		{name: "total_pages_rounds_up", query: "?limit=2", wantStatus: http.StatusOK, wantCount: 2, wantTotal: 3, wantPage: 1, wantLimit: 2, wantPages: 2},
		{name: "zero_limit", query: "?limit=0", wantStatus: http.StatusUnprocessableEntity, wantError: "limit must be between 1 and 100"},
		{name: "negative_limit", query: "?limit=-1", wantStatus: http.StatusUnprocessableEntity, wantError: "limit must be between 1 and 100"},
		{name: "page_far_beyond_results", query: "?page=999999", wantStatus: http.StatusOK, wantCount: 0, wantTotal: 3, wantPage: 999999, wantLimit: 10, wantPages: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newIntegrationEnvironment(t)
			seedListData(t, env.repo)
			recorder := performJSONRequest(t, env.router, http.MethodGet, "/disbursements"+tt.query, "", "operator")

			require.Equal(t, tt.wantStatus, recorder.Code)
			response := decodeResponse(t, recorder)
			if tt.wantError != "" {
				assert.Contains(t, response.Message, tt.wantError)
				return
			}

			var data []models.Disbursement
			require.NoError(t, json.Unmarshal(response.Data, &data))
			assert.Len(t, data, tt.wantCount)
			assert.Equal(t, fmt.Sprint(tt.wantTotal), response.Meta["total"].String())
			assert.Equal(t, fmt.Sprint(tt.wantPage), response.Meta["page"].String())
			assert.Equal(t, fmt.Sprint(tt.wantLimit), response.Meta["limit"].String())
			assert.Equal(t, fmt.Sprint(tt.wantPages), response.Meta["total_pages"].String())
		})
	}
}
