package filters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockCacheService is a mock implementation of CacheService
type MockCacheService struct {
	getFunc    func(ctx context.Context, req *http.Request) (*http.Response, error)
	setFunc    func(ctx context.Context, req *http.Request, res *http.Response, expr time.Duration) error
	deleteFunc func(ctx context.Context, req *http.Request) error
}

func (m *MockCacheService) Get(ctx context.Context, req *http.Request) (*http.Response, error) {
	return m.getFunc(ctx, req)
}

func (m *MockCacheService) Set(ctx context.Context, req *http.Request, res *http.Response, expr time.Duration) error {
	return m.setFunc(ctx, req, res, expr)
}

func (m *MockCacheService) Delete(ctx context.Context, req *http.Request) error {
	return m.deleteFunc(ctx, req)
}

// MockFilter is a mock implementation of Filter
type MockFilter struct {
	processFunc func(ctx context.Context, req *http.Request, res *http.Response) error
}

func (m *MockFilter) Process(ctx context.Context, req *http.Request, res *http.Response) error {
	return m.processFunc(ctx, req, res)
}

func TestCacheMgrFilterProcess(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockCacheService, *MockFilter)
		expectedErr    error
		expectedStatus int
	}{
		{
			name: "Cache hit",
			setupMocks: func(cs *MockCacheService, nf *MockFilter) {
				cs.getFunc = func(ctx context.Context, req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK}, nil
				}
			},
			expectedErr:    nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Cache miss, next filter succeeds",
			setupMocks: func(cs *MockCacheService, nf *MockFilter) {
				cs.getFunc = func(ctx context.Context, req *http.Request) (*http.Response, error) {
					return nil, ErrCacheMiss{"Cache miss"}
				}
				nf.processFunc = func(ctx context.Context, req *http.Request, res *http.Response) error {
					res.StatusCode = http.StatusCreated
					return nil
				}
				cs.setFunc = func(ctx context.Context, req *http.Request, res *http.Response, expr time.Duration) error {
					return nil
				}
			},
			expectedErr:    nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Cache miss, next filter fails",
			setupMocks: func(cs *MockCacheService, nf *MockFilter) {
				cs.getFunc = func(ctx context.Context, req *http.Request) (*http.Response, error) {
					return nil, ErrCacheMiss{"Cache miss"}
				}
				nf.processFunc = func(ctx context.Context, req *http.Request, res *http.Response) error {
					return errors.New("Next filter error")
				}
			},
			expectedErr:    errors.New("Next filter error"),
			expectedStatus: http.StatusInternalServerError, // Default status
		},
		{
			name: "Cache miss, next filter succeeds, set fails",
			setupMocks: func(cs *MockCacheService, nf *MockFilter) {
				cs.getFunc = func(ctx context.Context, req *http.Request) (*http.Response, error) {
					return nil, ErrCacheMiss{"Cache miss"}
				}
				nf.processFunc = func(ctx context.Context, req *http.Request, res *http.Response) error {
					res.StatusCode = http.StatusAccepted
					return nil
				}
				cs.setFunc = func(ctx context.Context, req *http.Request, res *http.Response, expr time.Duration) error {
					return errors.New("Set cache error")
				}
			},
			expectedErr:    nil, // Process should not return an error even if Set fails
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCS := &MockCacheService{}
			mockNF := &MockFilter{}
			tt.setupMocks(mockCS, mockNF)

			cm := &cacheMgrFilter{
				cs:         mockCS,
				nextFilter: mockNF,
			}

			ctx := context.Background()
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			res := &http.Response{StatusCode: http.StatusOK} // Default status

			err := cm.Process(ctx, req, res)

			if (err != nil && tt.expectedErr == nil) || (err == nil && tt.expectedErr != nil) || (err != nil && tt.expectedErr != nil && err.Error() != tt.expectedErr.Error()) {
				t.Errorf("Process() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Process() status = %v, expectedStatus %v", res.StatusCode, tt.expectedStatus)
			}
		})
	}
}

func TestNewCacheMgrFilter(t *testing.T) {
	tests := []struct {
		name        string
		cacheServer CacheService
		wantErr     bool
	}{
		{
			name:        "Valid cache service",
			cacheServer: &MockCacheService{},
			wantErr:     false,
		},
		{
			name:        "Nil cache service",
			cacheServer: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCacheMgrFilter(tt.cacheServer)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCacheMgrFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Errorf("NewCacheMgrFilter() returned nil, want non-nil")
				}
				if got.cs != tt.cacheServer {
					t.Errorf("NewCacheMgrFilter() got = %v, want %v", got.cs, tt.cacheServer)
				}
			}
		})
	}
}

func TestCacheMgrFilter_SetNextFilter(t *testing.T) {
	tests := []struct {
		name       string
		nextFilter Filter
		wantErr    bool
	}{
		{
			name:       "Valid next filter",
			nextFilter: &MockFilter{},
			wantErr:    false,
		},
		{
			name:       "Nil next filter",
			nextFilter: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, _ := NewCacheMgrFilter(&MockCacheService{})
			err := cm.SetNextFilter(tt.nextFilter)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetNextFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if cm.nextFilter != tt.nextFilter {
					t.Errorf("SetNextFilter() got = %v, want %v", cm.nextFilter, tt.nextFilter)
				}
			}
			if tt.wantErr && err == nil {
				t.Errorf("SetNextFilter() expected error for nil input, got nil")
			}
			if tt.wantErr && err != nil && err.Error() != "nextFilter = <nil>" {
				t.Errorf("SetNextFilter() unexpected error message: %v", err)
			}
		})
	}
}

func TestCacheMgrFilter_SetCacheService(t *testing.T) {
	tests := []struct {
		name         string
		cacheService CacheService
		wantErr      bool
	}{
		{
			name:         "Valid cache service",
			cacheService: &MockCacheService{},
			wantErr:      false,
		},
		{
			name:         "Nil cache service",
			cacheService: nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, _ := NewCacheMgrFilter(&MockCacheService{})
			err := cm.SetCacheService(tt.cacheService)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetCacheService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if cm.cs != tt.cacheService {
					t.Errorf("SetCacheService() got = %v, want %v", cm.cs, tt.cacheService)
				}
			}
			if tt.wantErr && err == nil {
				t.Errorf("SetCacheService() expected error for nil input, got nil")
			}
			if tt.wantErr && err != nil && err.Error() != "CacheService = <nil>" {
				t.Errorf("SetCacheService() unexpected error message: %v", err)
			}
		})
	}
}
