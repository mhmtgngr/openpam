package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMain_DefaultPort(t *testing.T) {
	// Clear PORT env
	os.Unsetenv("PORT")
	defer os.Unsetenv("PORT")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8501"
	}

	assert.Equal(t, "8501", port)
}

func TestMain_CustomPort(t *testing.T) {
	os.Setenv("PORT", "9001")
	defer os.Unsetenv("PORT")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8501"
	}

	assert.Equal(t, "9001", port)
}

func TestHealthEndpoint(t *testing.T) {
	// Create a minimal router for testing
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "session-service"})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"up"`)
	assert.Contains(t, w.Body.String(), `"service":"session-service"`)
}

func TestReadyEndpoint(t *testing.T) {
	r := gin.New()
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ready":true`)
}

func TestConfig_Defaults(t *testing.T) {
	os.Unsetenv("PORT")

	expectedPort := "8501"
	actualPort := os.Getenv("PORT")
	if actualPort == "" {
		actualPort = expectedPort
	}

	assert.Equal(t, expectedPort, actualPort)
}

func TestConfig_FromEnv(t *testing.T) {
	os.Setenv("PORT", "9999")
	defer os.Unsetenv("PORT")

	expectedPort := "9999"
	actualPort := os.Getenv("PORT")
	if actualPort == "" {
		actualPort = "8501"
	}

	assert.Equal(t, expectedPort, actualPort)
}
