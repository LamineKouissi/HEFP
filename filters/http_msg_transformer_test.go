package filters

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockFilter struct {
}

func (mf *mockFilter) Process(ctx context.Context, req *http.Request, res *http.Response) error {
	return nil
}

func TestNewHttpMsgTransformerFilter(t *testing.T) {
	tests := []struct {
		name            string
		nextFilterInput Filter
		expectedRslt    error
		expectedErr     bool
	}{
		{
			name:            "successfull creation",
			nextFilterInput: &mockFilter{},
			expectedRslt:    nil,
			expectedErr:     false,
		},
		{
			name:            "faild creation <nil input>",
			nextFilterInput: nil,
			expectedRslt:    errors.New("invalid input, nextFilter <nil>"),
			expectedErr:     true,
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			transormer, err := NewHttpMsgTransformerFilter(tst.nextFilterInput)
			if tst.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, transormer)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, transormer)
		})

	}

}

func TestSetNextFilter(t *testing.T) {
	transormer := HttpMsgTransformerFilter{}
	tests := []struct {
		name            string
		nextFilterInput Filter
		expectedRslt    error
		expectedErr     bool
	}{
		{
			name:            "successfull Set",
			nextFilterInput: &mockFilter{},
			expectedRslt:    nil,
			expectedErr:     false,
		},
		{
			name:            "faild Set <nil input>",
			nextFilterInput: nil,
			expectedRslt:    errors.New("invalid input, nextFilter <nil>"),
			expectedErr:     true,
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			err := transormer.SetNextFilter(tst.nextFilterInput)
			if tst.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})

	}

}

func TestRemoveHopHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string][]string
		expected map[string][]string
	}{
		{
			name: "Remove all hop headers",
			headers: map[string][]string{
				"Connection":          {"keep-alive"},
				"Proxy-Connection":    {"keep-alive"},
				"Keep-Alive":          {"timeout=5, max=1000"},
				"Proxy-Authenticate":  {"Basic"},
				"Proxy-Authorization": {"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="},
				"Te":                  {"trailers, deflate"},
				"Trailer":             {"Max-Forwards"},
				"Transfer-Encoding":   {"chunked"},
				"Upgrade":             {"websocket"},
				"Content-Type":        {"application/json"},
			},
			expected: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "No hop headers present",
			headers: map[string][]string{
				"Content-Type":   {"text/plain"},
				"Content-Length": {"100"},
				"User-Agent":     {"test-agent"},
			},
			expected: map[string][]string{
				"Content-Type":   {"text/plain"},
				"Content-Length": {"100"},
				"User-Agent":     {"test-agent"},
			},
		},
		{
			name:     "Empty headers",
			headers:  map[string][]string{},
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hmt := &HttpMsgTransformerFilter{}
			header := http.Header(tt.headers)
			hmt.removeHopHeaders(header)

			if len(header) != len(tt.expected) {
				t.Errorf("Expected %d headers, got %d", len(tt.expected), len(header))
			}

			for k, v := range tt.expected {
				if !equalSlices(header[k], v) {
					t.Errorf("Expected header %s to be %v, got %v", k, v, header[k])
				}
			}

			for _, h := range []string{
				"Connection",
				"Proxy-Connection",
				"Keep-Alive",
				"Proxy-Authenticate",
				"Proxy-Authorization",
				"Te",
				"Trailer",
				"Transfer-Encoding",
				"Upgrade",
			} {
				if _, exists := header[h]; exists {
					t.Errorf("Hop header %s should have been removed", h)
				}
			}
		})
	}
}

// Helper function to compare slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestRemoveConnectionHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		expected http.Header
	}{
		{
			name: "Remove single Connection header",
			headers: http.Header{
				"Connection":   {"Keep-Alive"},
				"Keep-Alive":   {"timeout=5, max=1000"},
				"Content-Type": {"application/json"},
			},
			expected: http.Header{
				"Connection":   {"Keep-Alive"},
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "Remove multiple Connection headers",
			headers: http.Header{
				"Connection":   {"Keep-Alive, Upgrade"},
				"Keep-Alive":   {"timeout=5, max=1000"},
				"Upgrade":      {"websocket"},
				"Content-Type": {"application/json"},
			},
			expected: http.Header{
				"Connection":   {"Keep-Alive, Upgrade"},
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "Handle comma-separated Connection values",
			headers: http.Header{
				"Connection":    {"Keep-Alive, Upgrade, Custom-Header"},
				"Keep-Alive":    {"timeout=5, max=1000"},
				"Upgrade":       {"websocket"},
				"Custom-Header": {"value"},
				"Content-Type":  {"application/json"},
			},
			expected: http.Header{
				"Connection":   {"Keep-Alive, Upgrade, Custom-Header"},
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "Handle multiple Connection headers",
			headers: http.Header{
				"Connection":   {"Keep-Alive", "Upgrade"},
				"Keep-Alive":   {"timeout=5, max=1000"},
				"Upgrade":      {"websocket"},
				"Content-Type": {"application/json"},
			},
			expected: http.Header{
				"Connection":   {"Keep-Alive", "Upgrade"},
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "Handle empty Connection header",
			headers: http.Header{
				"Connection":   {""},
				"Content-Type": {"application/json"},
			},
			expected: http.Header{
				"Connection":   {""},
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "No Connection header present",
			headers: http.Header{
				"Content-Type": {"application/json"},
			},
			expected: http.Header{
				"Content-Type": {"application/json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hmt := &HttpMsgTransformerFilter{}
			hmt.removeConnectionHeaders(tt.headers)

			if !reflect.DeepEqual(tt.headers, tt.expected) {
				t.Errorf("removeConnectionHeaders() = %v, want %v", tt.headers, tt.expected)
			}
		})
	}
}

func TestAppendHostToXForwardHeader(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		host     string
		expected http.Header
	}{
		{
			name:    "Append to empty header",
			headers: http.Header{},
			host:    "192.168.1.1",
			expected: http.Header{
				"X-Forwarded-For": {"192.168.1.1"},
			},
		},
		{
			name: "Append to existing single value",
			headers: http.Header{
				"X-Forwarded-For": {"10.0.0.1"},
			},
			host: "192.168.1.1",
			expected: http.Header{
				"X-Forwarded-For": {"10.0.0.1, 192.168.1.1"},
			},
		},
		{
			name: "Append to existing multiple values",
			headers: http.Header{
				"X-Forwarded-For": {"10.0.0.1", "172.16.0.1"},
			},
			host: "192.168.1.1",
			expected: http.Header{
				"X-Forwarded-For": {"10.0.0.1, 172.16.0.1, 192.168.1.1"},
			},
		},
		{
			name: "Append with other headers present",
			headers: http.Header{
				"X-Forwarded-For": {"10.0.0.1"},
				"Content-Type":    {"application/json"},
				"User-Agent":      {"test-agent"},
			},
			host: "192.168.1.1",
			expected: http.Header{
				"X-Forwarded-For": {"10.0.0.1, 192.168.1.1"},
				"Content-Type":    {"application/json"},
				"User-Agent":      {"test-agent"},
			},
		},
		{
			name: "Append IPv6 address",
			headers: http.Header{
				"X-Forwarded-For": {"10.0.0.1"},
			},
			host: "2001:db8::1",
			expected: http.Header{
				"X-Forwarded-For": {"10.0.0.1, 2001:db8::1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hmt := &HttpMsgTransformerFilter{}
			hmt.appendHostToXForwardHeader(tt.headers, tt.host)

			if !reflect.DeepEqual(tt.headers, tt.expected) {
				t.Errorf("appendHostToXForwardHeader() = %v, want %v", tt.headers, tt.expected)
			}

			// Check if the X-Forwarded-For header is set correctly
			xForwardedFor := tt.headers.Get("X-Forwarded-For")
			expectedXForwardedFor := tt.expected.Get("X-Forwarded-For")
			if xForwardedFor != expectedXForwardedFor {
				t.Errorf("X-Forwarded-For header = %v, want %v", xForwardedFor, expectedXForwardedFor)
			}
		})
	}
}
