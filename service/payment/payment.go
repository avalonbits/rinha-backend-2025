package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/avalonbits/rinha-backend-2025/service"
	"github.com/avalonbits/rinha-backend-2025/storage/datastore"
	"github.com/avalonbits/rinha-backend-2025/storage/sharded"
)

type Service struct {
	main   paymentClient
	backup paymentClient
	store  *sharded.Store
}

func New(mainURL, backupURL string, store *sharded.Store) *Service {
	return &Service{
		main:   createClient("default", mainURL, 0),
		backup: createClient("fallback", backupURL, 2500*time.Millisecond),
		store:  store,
	}
}

type processPaymentRequest struct {
	CorrleationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
	RequestedAt   string  `json:"requestedAt"`
}

const ErrAlreadyProcessed = service.Error("payment was already processed. check correlationId")

func (s *Service) ProcessPayment(
	ctx context.Context, correlationID string, amount float64, requestedAt string,
) (string, error) {
	req := processPaymentRequest{
		CorrleationID: correlationID,
		Amount:        amount,
		RequestedAt:   requestedAt,
	}

	client := s.preferDefault()
	status, err := client.post(ctx, "/payments", req, nil)
	if err != nil {
		if status == http.StatusUnprocessableEntity {
			return "", ErrAlreadyProcessed
		}
		return "", err
	}

	return client.String(), nil
}

type SummaryResponse struct {
	Default  paymentProcessor `json:"default"`
	Fallback paymentProcessor `json:"fallback"`
}

type paymentProcessor struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

func (s *Service) Summary(ctx context.Context, from, to string) (SummaryResponse, error) {
	partialRes := make([]SummaryResponse, s.store.ShardCount())
	err := s.store.ReadAll(ctx, func(shard int, queries *datastore.Queries) error {
		res := &partialRes[shard]
		summary, err := queries.PaymentSummary(ctx, datastore.PaymentSummaryParams{
			FromRequestedAt: from,
			ToRequestedAt:   to,
		})
		if err != nil {
			return err
		}

		for _, row := range summary {
			if row.Processor == "default" {
				res.Default.TotalRequests += int(row.Total)
				res.Default.TotalAmount += row.Amount.Float64
			} else if row.Processor == "fallback" {
				res.Fallback.TotalRequests += int(row.Total)
				res.Fallback.TotalAmount += row.Amount.Float64
			}
		}
		return nil
	})

	if err != nil {
		return SummaryResponse{}, err
	}

	res := SummaryResponse{}
	for _, partial := range partialRes {
		res.Default.TotalAmount += partial.Default.TotalAmount
		res.Default.TotalRequests += partial.Default.TotalRequests
		res.Fallback.TotalAmount += partial.Fallback.TotalAmount
		res.Fallback.TotalRequests += partial.Fallback.TotalRequests
	}

	return res, nil
}

func (s *Service) Log(
	ctx context.Context, correlationID string, amount float64, requestedAt string, processor string,
) error {
	return s.store.Write(ctx, correlationID, func(queries *datastore.Queries) error {
		return queries.LogPayment(ctx, datastore.LogPaymentParams{
			CorrelationID: correlationID,
			RequestedAt:   requestedAt,
			Amount:        amount,
			Processor:     processor,
		})
	})
}

func (s *Service) Expunge(ctx context.Context, correlationID string) error {
	return s.store.Write(ctx, correlationID, func(queries *datastore.Queries) error {
		return queries.ExpungePayment(ctx, correlationID)
	})
}

func (s *Service) preferDefault() paymentClient {
	for {
		if s.main.available.Load() {
			return s.main
		} else if s.backup.available.Load() {
			return s.backup
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

type paymentClient struct {
	baseURL   *url.URL
	httpC     *http.Client
	available *atomic.Bool
	name      string
}

func createClient(name, serviceURL string, delayFirstCheck time.Duration) paymentClient {
	baseURL, err := url.Parse(serviceURL)
	if err != nil {
		panic(err)
	}

	c := paymentClient{
		baseURL: baseURL,
		httpC: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 30 * time.Second,
			},
		},
		available: &atomic.Bool{},
		name:      name,
	}

	// Keep track of the availability of the client.
	go func() {
		if delayFirstCheck > 0 {
			time.Sleep(delayFirstCheck)
		}
		c.updateAvailability()
		ticker := time.Tick(5001 * time.Millisecond)
		for {
			<-ticker
			c.updateAvailability()
		}
	}()

	return c
}

type updateAvailabilityResponse struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

func (c paymentClient) String() string {
	return c.name
}

func (c paymentClient) updateAvailability() {
	res := updateAvailabilityResponse{}
	if err := c.get("/payments/service-health", &res); err != nil {
		fmt.Println(err)
		c.available.Store(false)
	} else {
		c.available.Store(!res.Failing)
	}
}

func (c paymentClient) get(endpoint string, res any) error {
	target := c.baseURL.JoinPath(endpoint)

	resp, err := c.httpC.Get(target.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s: %q", http.StatusText(resp.StatusCode), msg)
	}

	if res == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	return nil
}

func (c paymentClient) post(ctx context.Context, endpoint string, req, res any) (int, error) {
	target := c.baseURL.JoinPath(endpoint)

	buf := bytes.Buffer{}
	if req != nil {
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			return http.StatusInternalServerError, err
		}
	}

	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), &buf)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	postReq.Header.Add("Content-Type", "application/json")

	resp, err := c.httpC.Do(postReq)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, err
		}
		return resp.StatusCode, fmt.Errorf("%s", string(msg))
	}

	if res == nil {
		return resp.StatusCode, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		if errors.Is(err, io.EOF) {
			return resp.StatusCode, nil
		}
		return http.StatusInternalServerError, err
	}

	return resp.StatusCode, nil
}
