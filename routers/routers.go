package routers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// MakeAPICall faz uma requisição GET com contexto e decodifica o JSON para target.
// Retorna o status HTTP da resposta (quando disponível) e um erro descritivo.
func MakeAPICall(ctx context.Context, client *http.Client, url string, target interface{}) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("erro ao criar request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("erro na requisição GET: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("erro ao ler a resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("erro na API. Status: %s. Resposta: %s", resp.Status, string(bodyBytes))
	}

	if err := json.Unmarshal(bodyBytes, target); err != nil {
		return resp.StatusCode, fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	return resp.StatusCode, nil
}

func statusOr500(code int) int {
	if code >= 100 && code <= 599 {
		return code
	}
	return http.StatusInternalServerError
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/fetch", fetchHandler)
}

func fetchHandler(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query 'url' obrigatória"})
		return
	}

	timeout := 10 * time.Second
	if t := c.Query("timeout"); t != "" {
		if parsed, err := time.ParseDuration(t); err == nil {
			timeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	client := &http.Client{Timeout: timeout}

	var result interface{}
	status, err := MakeAPICall(ctx, client, url, &result)
	if err != nil {
		c.JSON(statusOr500(status), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}


