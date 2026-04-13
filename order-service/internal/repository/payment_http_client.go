package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type PaymentHTTPClient struct {
	client  *http.Client
	baseURL string
}

func NewPaymentHTTPClient(client *http.Client, baseURL string) *PaymentHTTPClient {
	return &PaymentHTTPClient{client: client, baseURL: baseURL}
}

type authorizeRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type authorizeResponse struct {
	Status string `json:"status"`
}

func (httpClient *PaymentHTTPClient) Authorize(ctx context.Context, orderID string, amount int64) (string, error) {
	body, _ := json.Marshal(authorizeRequest{OrderID: orderID, Amount: amount})

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, httpClient.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("payment service unreachable: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("payment service returned status %d", response.StatusCode)
	}

	var result authorizeResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode payment response: %w", err)
	}
	return result.Status, nil
}
