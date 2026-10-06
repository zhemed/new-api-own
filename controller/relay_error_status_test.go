package controller

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestRelayValidationErrorStatus guards the client-visible contract that a
// request-validation failure is answered with 400 (the status published for
// these endpoints in docs/openapi/relay.json), not with the 500 that
// types.NewError defaults to.
func TestRelayValidationErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newMultipartContext := func(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("prompt", "a cat"))
		require.NoError(t, writer.Close())

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return c, recorder
	}

	tests := []struct {
		name       string
		format     types.RelayFormat
		newContext func(*testing.T) (*gin.Context, *httptest.ResponseRecorder)
		wantMsg    string
	}{
		{
			name:   "chat completion without model",
			format: types.RelayFormatOpenAI,
			newContext: func(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
					strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
				c.Request.Header.Set("Content-Type", "application/json")
				return c, recorder
			},
			wantMsg: "model is required",
		},
		{
			name:       "multipart image edit without model",
			format:     types.RelayFormatOpenAIImage,
			newContext: newMultipartContext,
			wantMsg:    "model is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := tt.newContext(t)

			Relay(c, tt.format)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), tt.wantMsg)
		})
	}
}

// TestRelaySensitiveWordsStatus guards that a content-policy rejection caused by
// the client's own prompt is answered with 400 - the same class the gateway uses
// for upstream policy blocks (relay/channel/gemini/relay-gemini.go:334) and the
// status class published for relay endpoints in docs/openapi/relay.json. A 5xx
// would make relays and client SDKs retry a request that can never succeed.
func TestRelaySensitiveWordsStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalWords := setting.SensitiveWords
	originalCheckEnabled := setting.CheckSensitiveEnabled
	originalPromptEnabled := setting.CheckSensitiveOnPromptEnabled
	setting.SensitiveWords = []string{"blocked-phrase"}
	setting.CheckSensitiveEnabled = true
	setting.CheckSensitiveOnPromptEnabled = true
	t.Cleanup(func() {
		setting.SensitiveWords = originalWords
		setting.CheckSensitiveEnabled = originalCheckEnabled
		setting.CheckSensitiveOnPromptEnabled = originalPromptEnabled
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-test","messages":[{"role":"user","content":"please repeat blocked-phrase"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	Relay(c, types.RelayFormatOpenAI)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), string(types.ErrorCodeSensitiveWordsDetected))
}
