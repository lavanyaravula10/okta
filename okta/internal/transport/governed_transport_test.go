package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/okta/terraform-provider-okta/okta/internal/apimutex"
)

func TestPreRequestHook(t *testing.T) {
	percentage := 10
	limit := 25
	remaining := 23
	reset := time.Now().Unix() + 30
	path := "/api/v1/apps"

	client := &http.Client{}
	apiMutex, _ := apimutex.NewAPIMutex(percentage)
	transport := NewGovernedTransport(client.Transport, apiMutex, hclog.NewNullLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	apiMutex.Update(http.MethodGet, path, limit, remaining, reset)
	if err := transport.preRequestHook(ctx, http.MethodGet, path); err != nil {
		t.Errorf("Didn't expect error, got %+v", err)
	}

	remaining--
	apiMutex.Update(http.MethodGet, path, limit, remaining, reset)
	if err := transport.preRequestHook(ctx, http.MethodGet, path); err != context.Canceled {
		t.Errorf("Expected %v error, got %+v", context.Canceled, err)
	}
}

func TestPostRequestHook(t *testing.T) {
	percentage := 10
	client := &http.Client{}
	apiMutex, _ := apimutex.NewAPIMutex(percentage)
	transport := NewGovernedTransport(client.Transport, apiMutex, hclog.NewNullLogger())

	path := "/api/v1/apps"
	request := http.Request{
		URL: &url.URL{
			Path: path,
		},
	}
	limit := 25
	remaining := 17
	reset := time.Now().Unix() + 30
	headers := http.Header{}
	headers.Add("x-rate-limit-limit", fmt.Sprintf("%v", limit))
	headers.Add("x-rate-limit-remaining", fmt.Sprintf("%v", remaining))
	headers.Add("x-rate-limit-reset", fmt.Sprintf("%v", reset))
	response := http.Response{
		Request: &request,
		Header:  headers,
	}

	transport.postRequestHook(http.MethodGet, path, &response)
	status := apiMutex.Status(http.MethodGet, path)
	if status.Reset() != reset || status.Limit() != limit || status.Remaining() != remaining {
		t.Fatalf("expected %q api mutex status %+v to have reset %d, limit %d, and remaining %d values", path, status, reset, limit, remaining)
	}
}
