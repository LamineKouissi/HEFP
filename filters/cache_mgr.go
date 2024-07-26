package filters

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

type ErrCacheMiss struct {
	Msg string
}

func (e ErrCacheMiss) Error() string {
	return e.Msg
}

type CacheService interface {
	Get(ctx context.Context, req *http.Request) (*http.Response, error)
	Set(ctx context.Context, req *http.Request, res *http.Response, expr time.Duration) error
	Delete(ctx context.Context, req *http.Request) error
}

type cacheMgrFilter struct {
	cs         CacheService
	nextFilter Filter
}

func NewCacheMgrFilter(cacheSrvs CacheService) (*cacheMgrFilter, error) {
	if cacheSrvs == nil {
		return nil, errors.New("CacheService = <nil>")
	}
	return &cacheMgrFilter{cs: cacheSrvs}, nil
}

func (cm *cacheMgrFilter) SetNextFilter(f Filter) error {
	if f == nil {
		return errors.New("nextFilter = <nil>")
	}
	cm.nextFilter = f
	return nil
}

func (cm *cacheMgrFilter) SetCacheService(cacheSrvs CacheService) error {
	if cacheSrvs == nil {
		return errors.New("CacheService = <nil>")
	}
	cm.cs = cacheSrvs
	return nil
}

// See HTTP Caching - RFC 9111
func (cm *cacheMgrFilter) Process(ctx context.Context, req *http.Request, res *http.Response) error {
	fmt.Println("cacheMgrFilter.Process() sourse Request:------------------")
	rowSourceReq, err := httputil.DumpRequest(req, false)
	if err != nil {
		log.Fatal("cacheMgrFilter.Process() DumpRequest(): ", err)
	}
	fmt.Println(string(rowSourceReq))

	cachedRes, err := cm.cs.Get(ctx, req)
	if err != nil {
		log.Println("cacheMgrFilter.Process(){cm.cs.Get(...)}: ", err)
		err = cm.nextFilter.Process(ctx, req, res)
		if err != nil {
			log.Println("cacheMgrFilter.Process(){cm.nextFilter.Process()}: ", err)
			*res = http.Response{StatusCode: http.StatusInternalServerError}
			return err
		} else {
			err = cm.cs.Set(ctx, req, res, 0)
			if err != nil {
				log.Println("cacheMgrFilter.Process(){cm.cs.Set()}: ", err)
			}
		}
	} else {
		*res = *cachedRes
	}

	fmt.Println("cacheMgrFilter.Process() sourse Response:------------------")
	rowSourceRes, err := httputil.DumpResponse(res, false)
	if err != nil {
		log.Fatal("cacheMgrFilter.Process() DumpResponse(): ", err)
	}
	fmt.Println(string(rowSourceRes))

	return nil
}
