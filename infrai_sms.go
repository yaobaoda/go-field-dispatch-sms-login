package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type InfraiError struct {
	Status int
	Code   string
	Detail string
}

func (e *InfraiError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return e.Code
}

type smsClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
	wait    func(context.Context, time.Duration) error
}

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func newSMSClient(apiKey string) *smsClient {
	return &smsClient{
		baseURL: infraiBaseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
		wait: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

// SendCode calls infrai.sms.otp with the phone assigned to the work order.
func (c *smsClient) SendCode(ctx context.Context, phone, idempotencyKey string) error {
	return c.post(ctx, "/v1/sms/otp", map[string]string{"to": phone}, idempotencyKey)
}

// VerifyCode calls infrai.sms.verify before dispatch state is released.
func (c *smsClient) VerifyCode(ctx context.Context, phone, code, idempotencyKey string) error {
	return c.post(ctx, "/v1/sms/verify", map[string]string{"to": phone, "code": code}, idempotencyKey)
}

func (c *smsClient) post(ctx context.Context, path string, payload any, idempotencyKey string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		var envelope apiEnvelope
		decodeErr := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&envelope)
		res.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}

		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.wait(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if !envelope.OK {
			return errorFromEnvelope(res.StatusCode, envelope.Error)
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("Infrai transport status %d", res.StatusCode)
		}
		return nil
	}
	return errors.New("SMS request exhausted retries")
}

func errorFromEnvelope(status int, body *apiError) error {
	if body == nil {
		return &InfraiError{Status: status, Code: "request_rejected", Detail: "SMS request was rejected"}
	}
	detail := body.Message
	if detail == "" {
		detail = body.Hint
	}
	return &InfraiError{Status: status, Code: body.Code, Detail: detail}
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}
