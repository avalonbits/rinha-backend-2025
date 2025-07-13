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
)

type Service struct {
	main   client
	backup client
}

func New(mainURL, backupURL string) *Service {
	return &Service{
		main:   createClient(mainURL, 0),
		backup: createClient(backupURL, 2500*time.Millisecond),
	}
}

func (s *Service) preferDefault() *paymentClient {
	for {
		if s.main.available.Load() {
			return &paymentClient{
				httpC: s.main,
			}
		} else if s.backup.available.Load() {
			return &paymentClient{
				httpC: s.backup,
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

type paymentClient struct {
	httpC client
}

type processPaymentRequest struct {
	CorrleationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
	RequestedAt   string  `json:"requestedAt"`
}

const ErrAlreadyProcessed = service.Error("payment was already processed. check correlationId")

func (s *Service) ProcessPayment(ctx context.Context, correlationID string, amount float64, requestedAt string) error {
	req := processPaymentRequest{
		CorrleationID: correlationID,
		Amount:        amount,
		RequestedAt:   requestedAt,
	}

	status, err := s.preferDefault().httpC.post(ctx, "/payments", req, nil)
	if err != nil {
		if status == http.StatusUnprocessableEntity {
			return ErrAlreadyProcessed
		}
		return err
	}
	return nil
}

func createClient(serviceURL string, delayFirstCheck time.Duration) client {
	baseURL, err := url.Parse(serviceURL)
	if err != nil {
		panic(err)
	}

	c := client{
		baseURL: baseURL,
		httpC: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 30 * time.Second,
			},
		},
		available: &atomic.Bool{},
	}

	// Keep track of the availability of the client.
	go func() {
		if delayFirstCheck > 0 {
			time.Sleep(delayFirstCheck)
		}
		c.updateAvailability()
		ticker := time.Tick(5010 * time.Millisecond)
		for {
			<-ticker
			c.updateAvailability()
		}
	}()

	return c
}

type client struct {
	baseURL   *url.URL
	httpC     *http.Client
	available *atomic.Bool
}

type updateAvailabilityResponse struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

func (c client) updateAvailability() {
	res := updateAvailabilityResponse{}
	if err := c.get("/payments/service-health", &res); err != nil {
		fmt.Println(err)
		c.available.Store(false)
	} else {
		c.available.Store(!res.Failing)
	}
}

func (c client) get(endpoint string, res any) error {
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

func (c client) post(ctx context.Context, endpoint string, req, res any) (int, error) {
	target := c.baseURL.JoinPath(endpoint)

	buf := bytes.Buffer{}
	if req != nil {
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			return http.StatusInternalServerError, err
		}
	}
	fmt.Println(buf.String())

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

	fmt.Println(resp)
	if resp.StatusCode >= 400 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, err
		}
		return resp.StatusCode, fmt.Errorf(string(msg))
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
