package api

import (
	"bytes"
	"context"
	"encoding/json"
	goer "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
	"github.com/mminichino/couchbase-capella-aidp/version"
)

const clientName = "terraform-provider-couchbase-capella-aidp"

var userAgent = fmt.Sprintf("%s/%s", clientName, version.ProviderVersion)

// Client is responsible for constructing and executing HTTP requests.
type Client struct {
	*http.Client
}

// NewClient instantiates a new Client with the provided timeout.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		Client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Response encapsulates the response details.
type Response struct {
	Response *http.Response
	Body     []byte
}

// EndpointCfg encapsulates request details to endpoints.
type EndpointCfg struct {
	Url             string
	Method          string
	SuccessStatus   int
	SuccessStatuses []int // optional additional accepted success status codes
}

func (e EndpointCfg) isSuccess(code int) bool {
	if e.SuccessStatus != 0 && code == e.SuccessStatus {
		return true
	}
	for _, s := range e.SuccessStatuses {
		if code == s {
			return true
		}
	}
	return false
}

const defaultWaitAttempt = time.Second * 2

// ExecuteWithRetry constructs and executes an HTTP request with retry.
func (c *Client) ExecuteWithRetry(
	ctx context.Context,
	endpointCfg EndpointCfg,
	payload any,
	authToken string,
	headers map[string]string,
) (response *Response, err error) {
	var requestBody []byte
	var dur time.Duration
	if payload != nil {
		requestBody, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errors.ErrMarshallingPayload, err)
		}
	}

	fn := func() (response *Response, backoff time.Duration, err error) {
		var body io.Reader
		if requestBody != nil {
			body = bytes.NewReader(requestBody)
		}
		req, err := http.NewRequestWithContext(ctx, endpointCfg.Method, endpointCfg.Url, body)
		if err != nil {
			return nil, dur, fmt.Errorf("%s: %w", errors.ErrConstructingRequest, err)
		}

		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("User-Agent", userAgent)
		if requestBody != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		for header, value := range headers {
			req.Header.Set(header, value)
		}

		apiRes, err := c.Do(req)
		if err != nil {
			return nil, dur, fmt.Errorf("%s: %w", errors.ErrExecutingRequest, err)
		}
		defer apiRes.Body.Close()

		responseBody, err := io.ReadAll(apiRes.Body)
		if err != nil {
			return nil, dur, err
		}

		if endpointCfg.isSuccess(apiRes.StatusCode) {
			// success
		} else {
			if os.Getenv("AIDP_DEBUG_HTTP") == "1" {
				fmt.Fprintf(os.Stderr, "AIDP_DEBUG_HTTP method=%s url=%s status=%d reqBody=%s respBody=%s\n",
					endpointCfg.Method, endpointCfg.Url, apiRes.StatusCode, string(requestBody), string(responseBody))
			}
			switch apiRes.StatusCode {
			case http.StatusTooManyRequests:
				header := apiRes.Header.Get("Retry-After")
				retryAfter, parseErr := strconv.Atoi(header)
				if parseErr != nil {
					return nil, dur, fmt.Errorf("error parsing Retry-After value from response header")
				}
				dur = time.Second * time.Duration(retryAfter)
				tflog.Debug(ctx, "API rate limited", map[string]interface{}{
					"method":      endpointCfg.Method,
					"url":         endpointCfg.Url,
					"retry_after": dur.Seconds(),
				})
				return nil, dur, errors.ErrRatelimit
			case http.StatusServiceUnavailable:
				tflog.Debug(ctx, "API returned 503 Service Unavailable, retrying", map[string]interface{}{
					"method": endpointCfg.Method,
					"url":    endpointCfg.Url,
				})
				return nil, 0, errors.ErrServiceUnavailable
			case http.StatusGatewayTimeout:
				return nil, dur, errors.ErrGatewayTimeout
			default:
				var apiError Error
				if unmarshalErr := json.Unmarshal(responseBody, &apiError); unmarshalErr != nil {
					return nil, dur, fmt.Errorf(
						"unexpected code: %d, expected: %d, body: %s",
						apiRes.StatusCode, endpointCfg.SuccessStatus, responseBody)
				}
				if apiError.Code == 0 {
					return nil, dur, fmt.Errorf(
						"unexpected code: %d, expected: %d, body: %s",
						apiRes.StatusCode, endpointCfg.SuccessStatus, responseBody)
				}
				return nil, dur, &apiError
			}
		}

		return &Response{
			Response: apiRes,
			Body:     responseBody,
		}, dur, nil
	}

	return exec(ctx, fn, defaultWaitAttempt)
}

func exec(
	ctx context.Context, fn func() (response *Response, dur time.Duration, err error), waitOnReattempt time.Duration,
) (*Response, error) {
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()

	var (
		err      error
		backOff  time.Duration
		response *Response
	)

	const timeout = time.Minute * 10

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out executing request against api: %w", ctx.Err())
		case <-timer.C:
			response, backOff, err = fn()
			switch {
			case err == nil:
				return response, nil
			case goer.Is(err, errors.ErrRatelimit):
			case goer.Is(err, errors.ErrServiceUnavailable):
			case !goer.Is(err, errors.ErrGatewayTimeout):
				return response, err
			}

			if backOff > 0 {
				timer.Reset(backOff)
			} else {
				timer.Reset(waitOnReattempt)
			}
		}
	}
}
