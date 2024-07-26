package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/LamineKouissi/LHP/filters"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewRedisCacheAdapter(t *testing.T) {
	rd, err := NewRedisCacheAdapter("localhost:6379", "", "", "0")
	assert.NoError(t, err)
	assert.NotEmpty(t, rd)

	rdc, err := rd.GetClient()
	assert.NoError(t, err, "RedisCacheAdapter.GetClient() : ")

	// Perform basic diagnostic to check if the connection is working
	// Expected result > ping: PONG
	// If Redis is not running, error case is taken instead
	ctx := context.Background()
	status, err := rdc.Ping(ctx).Result()
	assert.NoError(t, err, "Redis connection was refused")
	assert.Equal(t, "PONG", status)
	t.Log(status)

}

func TestRedisCacheAdapterGet(t *testing.T) {

	adapter, err := NewRedisCacheAdapter("localhost:6379", "", "", "0")
	assert.NoError(t, err)
	ctx := context.Background()

	t.Run("Successful cache hit", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com/keyexist", nil)

		cachedRes := cacheHttpResponse{
			Status:     "200 OK",
			StatusCode: 200,
			HeaderJSON: `{"Content-Type":["application/json"]}`,
			Body:       []byte(`{"message":"Hello, World!"}`),
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		}

		res, err := adapter.Get(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, cachedRes.Status, res.Status)
		assert.Equal(t, cachedRes.StatusCode, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
		assert.Equal(t, cachedRes.Proto, res.Proto)
		assert.Equal(t, cachedRes.ProtoMajor, res.ProtoMajor)
		assert.Equal(t, cachedRes.ProtoMinor, res.ProtoMinor)

		bodyBytes, _ := json.Marshal(map[string]string{"message": "Hello, World!"})
		assert.Equal(t, bodyBytes, cachedRes.Body)
	})

	t.Run("Cache miss", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com/notfoundkey", nil)

		res, err := adapter.Get(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.IsType(t, filters.ErrCacheMiss{}, err)
	})

	t.Run("Redis error", func(t *testing.T) {
		//mocking redis to simulate HGetAll() withe an err
		req, _ := http.NewRequest("GET", "http://example.com/error", nil)
		cacheKey := "cache:GET:http://example.com/error"
		db, mock := redismock.NewClientMock()
		mockedAdapter := &redisCacheAdapter{
			client: db,
		}
		mock.ExpectHGetAll(cacheKey).SetErr(assert.AnError)

		res, err := mockedAdapter.Get(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "redis Get() failed")
	})

	t.Run("Invalid JSON header", func(t *testing.T) {
		//mocking redis to simulate HGetAll() with invalid JSON header
		req, _ := http.NewRequest("GET", "http://example.com/invalidheader", nil)
		cacheKey := "cache:GET:http://example.com/invalidheader"
		db, mock := redismock.NewClientMock()
		mockedAdapter := &redisCacheAdapter{
			client: db,
		}

		mock.ExpectHGetAll(cacheKey).SetVal(map[string]string{
			"status":      "OK",
			"status_code": "200",
			"header":      `{"Invalid JSON"`,
			"body":        `{"message":"Hello, World!"}`,
			"proto":       "HTTP/1.1",
			"proto_major": "1",
			"proto_minor": "1",
		})

		res, err := mockedAdapter.Get(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "unexpected end of JSON input")
	})

	t.Run("Nil request", func(t *testing.T) {
		res, err := adapter.Get(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "getKey(*http.Request = nil)")
	})

}

func TestRedisCacheAdapterSet(t *testing.T) {
	// Create a mock redisCacheAdapter
	//db, mock := redismock.NewClientMock()

	adapter, err := NewRedisCacheAdapter("localhost:6379", "", "", "0")
	assert.NoError(t, err)
	ctx := context.Background()

	tests := []struct {
		name           string
		inputResponse  *http.Response
		inputRequest   *http.Request
		expectedOutput error
		expectError    bool
	}{
		{
			name: "Successful Set",
			inputResponse: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"message":"Hello, World!"}`)),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			inputRequest:   mustNewRequest("GET", "http://example.com/keyexist", nil),
			expectedOutput: nil,
			expectError:    false,
		},
		{
			name: "Upadate Already Existing key",
			inputResponse: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"message":"Hello, World!"}`)),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			inputRequest:   mustNewRequest("GET", "http://example.com/newkey", nil),
			expectedOutput: nil,
			expectError:    false,
		},
		{
			name:           "nil input Response",
			inputResponse:  nil,
			inputRequest:   mustNewRequest("GET", "http://www.google.com/", nil),
			expectedOutput: errors.New("error"),
			expectError:    true,
		}, {
			name: "nil input Request",
			inputResponse: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":    []string{"application/json"},
					"X-Request-ID":    []string{"123456"},
					"X-Response-Time": []string{"100ms"},
				},
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"data":"Some data"}`)),
				Proto:      "HTTP/2.0",
				ProtoMajor: 2,
				ProtoMinor: 0,
			},
			inputRequest:   nil,
			expectedOutput: errors.New("error"),
			expectError:    true,
		},
		{
			name:           "nil input Request and Response",
			inputResponse:  nil,
			inputRequest:   nil,
			expectedOutput: errors.New("error"),
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// key, err := adapter.getKey(tt.inputRequest)
			// assert.NoError(t, err)
			// mock.ExpectHSet(key, tt.inputRequest).SetVal()
			err := adapter.Set(ctx, tt.inputRequest, tt.inputResponse, 0)
			if tt.expectError {
				assert.Error(t, err)
				//assert.EqualError(t,err,)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisSet(t *testing.T) {

	adapter, err := NewRedisCacheAdapter("localhost:6379", "", "", "0")
	assert.NoError(t, err)
	ctx := context.Background()

	inputResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"message":"Hello, World!"}`)),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	inputRequest := mustNewRequest("GET", "http://example.com/keyexist", nil)
	t.Log(adapter.Set(ctx, inputRequest, inputResponse, 0))

}

func TestGetCacheHttpRes(t *testing.T) {
	adapter := &redisCacheAdapter{}

	tests := []struct {
		name           string
		inputResponse  *http.Response
		expectedOutput *cacheHttpResponse
		expectError    bool
	}{
		{
			name: "Simple response",
			inputResponse: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"message":"Hello, World!"}`)),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			expectedOutput: &cacheHttpResponse{
				Status:     "200 OK",
				StatusCode: 200,
				HeaderJSON: `{"Content-Type":["application/json"]}`,
				Body:       []byte(`{"message":"Hello, World!"}`),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			expectError: false,
		},
		{
			name: "Response with multiple headers",
			inputResponse: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				Header: http.Header{
					"Content-Type":    []string{"application/json"},
					"X-Request-ID":    []string{"123456"},
					"X-Response-Time": []string{"100ms"},
				},
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"data":"Some data"}`)),
				Proto:      "HTTP/2.0",
				ProtoMajor: 2,
				ProtoMinor: 0,
			},
			expectedOutput: &cacheHttpResponse{
				Status:     "200 OK",
				StatusCode: 200,
				HeaderJSON: `{"Content-Type":["application/json"],"X-Request-ID":["123456"],"X-Response-Time":["100ms"]}`,
				Body:       []byte(`{"data":"Some data"}`),
				Proto:      "HTTP/2.0",
				ProtoMajor: 2,
				ProtoMinor: 0,
			},
			expectError: false,
		},
		{
			name: "Response with empty body",
			inputResponse: &http.Response{
				Status:     "204 No Content",
				StatusCode: 204,
				Header:     http.Header{"Content-Length": []string{"0"}},
				Body:       ioutil.NopCloser(bytes.NewBufferString("")),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			expectedOutput: &cacheHttpResponse{
				Status:     "204 No Content",
				StatusCode: 204,
				HeaderJSON: `{"Content-Length":["0"]}`,
				Body:       []byte(``),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter.getCacheHttpRes(tt.inputResponse)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				assert.Equal(t, tt.expectedOutput.Status, result.Status)
				assert.Equal(t, tt.expectedOutput.StatusCode, result.StatusCode)
				assert.Equal(t, tt.expectedOutput.Body, result.Body)
				assert.Equal(t, tt.expectedOutput.Proto, result.Proto)
				assert.Equal(t, tt.expectedOutput.ProtoMajor, result.ProtoMajor)
				assert.Equal(t, tt.expectedOutput.ProtoMinor, result.ProtoMinor)

				// Compare HeaderJSON by unmarshaling and comparing the resulting maps
				var expectedHeader, resultHeader map[string][]string
				err = json.Unmarshal([]byte(tt.expectedOutput.HeaderJSON), &expectedHeader)
				assert.NoError(t, err)
				err = json.Unmarshal([]byte(result.HeaderJSON), &resultHeader)
				assert.NoError(t, err)
				assert.Equal(t, expectedHeader, resultHeader)

				// Verify that the original response body can still be read
				bodyBytes, err := ioutil.ReadAll(tt.inputResponse.Body)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput.Body, bodyBytes)
			}
		})
	}
}

func TestJSONToHeader(t *testing.T) {
	tests := []struct {
		name           string
		jsonInput      string
		expectedHeader http.Header
		expectError    bool
	}{
		{
			name:           "Simple header",
			jsonInput:      `{"Content-Type":["application/json"],"X-Request-ID":["123456"]}`,
			expectedHeader: http.Header{"Content-Type": []string{"application/json"}, "X-Request-ID": []string{"123456"}},
			expectError:    false,
		},
		{
			name:           "Header with multiple values",
			jsonInput:      `{"Set-Cookie":["session=abc123","user=john"]}`,
			expectedHeader: http.Header{"Set-Cookie": []string{"session=abc123", "user=john"}},
			expectError:    false,
		},
		{
			name:           "Empty JSON object",
			jsonInput:      `{}`,
			expectedHeader: http.Header{},
			expectError:    false,
		},
		{
			name:           "Invalid JSON",
			jsonInput:      `{"Invalid JSON":}`,
			expectedHeader: nil,
			expectError:    true,
		},
		{
			name:           "JSON with non-array values",
			jsonInput:      `{"Single-Value": "not an array"}`,
			expectedHeader: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := JSONToHeader(tt.jsonInput)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, header)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHeader, header)

				// Additional check: convert back to JSON and compare
				jsonBytes, err := json.Marshal(header)
				assert.NoError(t, err)

				var originalMap, convertedMap map[string][]string
				err = json.Unmarshal([]byte(tt.jsonInput), &originalMap)
				assert.NoError(t, err)
				err = json.Unmarshal(jsonBytes, &convertedMap)
				assert.NoError(t, err)

				assert.Equal(t, originalMap, convertedMap)
			}
		})
	}
}
func TestHeaderToJSON(t *testing.T) {
	tests := []struct {
		name          string
		header        http.Header
		expectedJSON  string
		expectedError bool
	}{
		{
			name: "Simple header",
			header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-ID": []string{"123456"},
			},
			expectedJSON:  `{"Content-Type":["application/json"],"X-Request-ID":["123456"]}`,
			expectedError: false,
		},
		{
			name: "Header with multiple values",
			header: http.Header{
				"Set-Cookie": []string{"session=abc123", "user=john"},
			},
			expectedJSON:  `{"Set-Cookie":["session=abc123","user=john"]}`,
			expectedError: false,
		},
		{
			name:          "Nil header",
			header:        nil,
			expectedJSON:  "{}",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr, err := headerToJSON(tt.header)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, tt.expectedJSON, jsonStr)

				// Verify that we can unmarshal the JSON back into a map
				var headerMap map[string][]string
				err = json.Unmarshal([]byte(jsonStr), &headerMap)
				assert.NoError(t, err)

				// Compare the original header with the unmarshaled map
				for key, values := range tt.header {
					assert.Equal(t, values, headerMap[key])
				}
			}
		})
	}
}

func TestRedisCacheAdapter_GetClient(t *testing.T) {
	rdAdapter, _ := NewRedisCacheAdapter("localhost:6379", "", "", "0")
	client, err := rdAdapter.GetClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetKey(t *testing.T) {
	adapter := &redisCacheAdapter{}

	tests := []struct {
		name        string
		request     *http.Request
		expectedKey string
		expectError bool
	}{
		{
			name:        "Valid GET request",
			request:     mustNewRequest("GET", "http://example.com/path", nil),
			expectedKey: "cache:GET:http://example.com/path",
			expectError: false,
		},
		{
			name:        "Valid POST request",
			request:     mustNewRequest("POST", "http://example.com/api?param=value", nil),
			expectedKey: "cache:POST:http://example.com/api?param=value",
			expectError: false,
		},
		{
			name:        "Nil request",
			request:     nil,
			expectedKey: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := adapter.getKey(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if key != tt.expectedKey {
				t.Errorf("Expected key %q, but got %q", tt.expectedKey, key)
			}
		})
	}
}

// Helper function to create http.Request without error handling
func mustNewRequest(method, url string, body []byte) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

func TestRedisCacheAdapter_getExprDur(t *testing.T) {
	cm := &redisCacheAdapter{}
	now := time.Now()

	tests := []struct {
		name           string
		headers        http.Header
		expectedDur    time.Duration
		expectedErrMsg string
	}{
		{
			name: "Cache-Control max-age",
			headers: http.Header{
				"Cache-Control": []string{"max-age=3600"},
			},
			expectedDur: 1 * time.Hour,
		},
		{
			name: "Cache-Control no-store",
			headers: http.Header{
				"Cache-Control": []string{"no-store"},
			},
			expectedDur: 0,
		},
		{
			name: "Expires header",
			headers: http.Header{
				"Expires": []string{now.Add(2 * time.Hour).Format(time.RFC1123)},
			},
			expectedDur: 2 * time.Hour,
		},
		{
			name:        "No cache headers",
			headers:     http.Header{},
			expectedDur: 5 * time.Minute,
		},
		{
			name: "Invalid Expires header",
			headers: http.Header{
				"Expires": []string{"invalid date"},
			},
			expectedDur: 5 * time.Minute,
		},
		{
			name: "Both Cache-Control and Expires",
			headers: http.Header{
				"Cache-Control": []string{"max-age=3600"},
				"Expires":       []string{now.Add(2 * time.Hour).Format(time.RFC1123)},
			},
			expectedDur: 1 * time.Hour, // Cache-Control takes precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &http.Response{Header: tt.headers}
			dur, err := cm.getExprDur(res)

			if tt.expectedErrMsg != "" {
				if err == nil || err.Error() != tt.expectedErrMsg {
					t.Errorf("getExprDur() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
				}
			} else if err != nil {
				t.Errorf("getExprDur() unexpected error: %v", err)
			}

			if tt.name == "Expires header" {
				// For Expires header, we can't predict the exact duration due to time passing during the test
				if dur < (tt.expectedDur-1*time.Second) || dur > tt.expectedDur {
					t.Errorf("getExprDur() got duration = %v, want close to %v", dur, tt.expectedDur)
				}
			} else if dur != tt.expectedDur {
				t.Errorf("getExprDur() got duration = %v, want %v", dur, tt.expectedDur)
			}
		})
	}
}

func TestRedisCacheAdapter_extractMaxAge(t *testing.T) {
	cm := &redisCacheAdapter{}

	tests := []struct {
		name         string
		cacheControl string
		expectedAge  int
	}{
		{
			name:         "Valid max-age",
			cacheControl: "max-age=3600",
			expectedAge:  3600,
		},
		{
			name:         "max-age with other directives",
			cacheControl: "public, max-age=7200, must-revalidate",
			expectedAge:  7200,
		},
		{
			name:         "No max-age",
			cacheControl: "public, must-revalidate",
			expectedAge:  0,
		},
		{
			name:         "Invalid max-age",
			cacheControl: "max-age=invalid",
			expectedAge:  0,
		},
		{
			name:         "Empty string",
			cacheControl: "",
			expectedAge:  0,
		},
		{
			name:         "max-age=0",
			cacheControl: "max-age=0",
			expectedAge:  0,
		},
		{
			name:         "Multiple max-age (first one should be used)",
			cacheControl: "max-age=3600, max-age=7200",
			expectedAge:  3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := cm.extractMaxAge(tt.cacheControl)
			if age != tt.expectedAge {
				t.Errorf("extractMaxAge() got = %v, want %v", age, tt.expectedAge)
			}
		})
	}
}
