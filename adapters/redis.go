package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/LamineKouissi/LHP/filters"
	"github.com/LamineKouissi/LHP/util"
	"github.com/redis/go-redis/v9"
)

type redisCacheAdapter struct {
	client *redis.Client
}

func NewRedisCacheAdapter(addr, usr, pass, DBnum string) (*redisCacheAdapter, error) {
	redisUrl := fmt.Sprintf("redis://%s:%s@%s/%s", usr, pass, addr, DBnum)
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		return nil, err
	}
	cl := redis.NewClient(opt)
	return &redisCacheAdapter{client: cl}, nil
}
func (r *redisCacheAdapter) GetClient() (*redis.Client, error) {
	return r.client, nil
}

type cachedBody []byte

func (cb cachedBody) MarshalBinary() ([]byte, error) {
	return cb, nil
}

type cacheHttpResponse struct {
	Status     string `redis:"status"`
	StatusCode int    `redis:"status_code"`
	Proto      string `redis:"proto"`
	ProtoMajor int    `redis:"proto_major"`
	ProtoMinor int    `redis:"proto_minor"`
	HeaderJSON string `redis:"header"`
	Body       []byte `redis:"body"`
}

func (r *redisCacheAdapter) Get(ctx context.Context, req *http.Request) (*http.Response, error) {

	k, err := r.getKey(req)
	if err != nil {
		log.Println("err : redisCacheAdapter.Get(){getKey()} : ", err)
		return nil, err
	}
	var cachedRes cacheHttpResponse
	err = r.client.HGetAll(ctx, k).Scan(&cachedRes)
	fmt.Println("cachedRes : ", cachedRes)

	if err != nil {
		log.Println("err : redisCacheAdapter.Get(){r.client.HGetAll(ctx, k).Scan()} : ", err)
		switch {
		case err == redis.Nil:
			return nil, filters.ErrCacheMiss{Msg: "key does not exist"}
		default:
			return nil, errors.Join(errors.New("redis Get() failed"), err)
		}
	}
	isEmpty, err := util.IsStructEmpty(cachedRes)
	if err != nil {
		log.Println("err : redisCacheAdapter.Get(){IsStructEmpty(cachedRes)} : ", err)
		return nil, err
	}
	if isEmpty {
		log.Println("err : redisCacheAdapter.Get(){r.client.HGetAll(ctx, k).Scan()} : ", "Empty cachedRes")
		return nil, filters.ErrCacheMiss{Msg: "key does not exist"}
	}

	// Parse the JSON header
	httpHeader, err := JSONToHeader(cachedRes.HeaderJSON)
	if err != nil {
		log.Println("err : redisCacheAdapter.Get(){JSONToHeader()} : ", err)
		return nil, err
	}

	res := &http.Response{
		Status:     cachedRes.Status,
		StatusCode: cachedRes.StatusCode,
		Header:     httpHeader,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(cachedRes.Body))),
		Proto:      cachedRes.Proto,
		ProtoMajor: cachedRes.ProtoMajor,
		ProtoMinor: cachedRes.ProtoMinor,
	}

	return res, nil
}

func (r *redisCacheAdapter) Set(ctx context.Context, req *http.Request, res *http.Response, expr time.Duration) error {
	k, err := r.getKey(req)
	if err != nil {
		log.Println("err : redisCacheAdapter.Set(...){getKey(...)} : ", err)
		return err
	}
	fmt.Println("key : ", k)
	if expr == 0 {
		expr, err = r.getExprDur(res)
		if err != nil || expr <= 0 {
			expr = time.Duration(5 * time.Minute)
		}
	}
	fmt.Println("Experation : ", expr)
	cacheHttpRes, err := r.getCacheHttpRes(res)
	if err != nil {
		log.Println("err : redisCacheAdapter.Set(...){r.getValue(...)} : ", err)
		return err
	}
	r.client.Expire(ctx, k, expr)
	err = r.client.HSet(ctx, k, cacheHttpRes).Err()

	if err != nil {
		log.Println("err : redisCacheAdapter.Set(...){r.client.HSet(...).Err()} : ", err)
		return errors.Join(errors.New("redis Set(...) failed"), err)
	}
	return nil
}

func (r *redisCacheAdapter) Delete(ctx context.Context, req *http.Request) error {
	return errors.New("redisCacheAdapter.Delete() : no implementation")
}

func (cm *redisCacheAdapter) getKey(req *http.Request) (string, error) {
	if req == nil {
		return "", errors.New("getKey(*http.Request = nil)")
	}
	k := "cache:" + req.Method + ":" + req.URL.String()
	return k, nil
}

func (cm *redisCacheAdapter) getCacheHttpRes(res *http.Response) (*cacheHttpResponse, error) {
	// Read the response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	// Reset the response body so it can be read again
	res.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	jsonFormatHeader, err := headerToJSON(res.Header)
	if err != nil {
		return nil, err
	}

	resToBeCached := cacheHttpResponse{
		Status:     res.Status,
		StatusCode: res.StatusCode,
		Proto:      res.Proto,
		ProtoMajor: res.ProtoMajor,
		ProtoMinor: res.ProtoMinor,
		HeaderJSON: jsonFormatHeader,
		Body:       body,
	}
	return &resToBeCached, nil
}

func headerToJSON(header http.Header) (string, error) {
	if header == nil {
		return "{}", nil
	}

	// Create a map to hold the header data
	headerMap := make(map[string][]string)

	// Copy the header data into the map
	for key, values := range header {
		headerMap[key] = values
	}

	// Marshal the map to JSON
	jsonBytes, err := json.Marshal(headerMap)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (cm *redisCacheAdapter) getExprDur(res *http.Response) (time.Duration, error) {
	// Check for Cache-Control header
	cacheControl := res.Header.Get("Cache-Control")
	if cacheControl != "" {
		if cacheControl == "no-store" {
			return 0, nil // Do not cache
		}
		if maxAge := cm.extractMaxAge(cacheControl); maxAge > 0 {
			return time.Duration(maxAge) * time.Second, nil
		}
	}

	// Check for Expires header
	expires := res.Header.Get("Expires")
	if expires != "" {
		expiresTime, err := time.Parse(time.RFC1123, expires)
		if err == nil {
			return time.Until(expiresTime), nil
		}
	}

	// Default expiration
	return 5 * time.Minute, nil
}

// Helper function to extract max-age from Cache-Control header
func (cm *redisCacheAdapter) extractMaxAge(cacheControl string) int {
	// This is a simplified version. A real implementation would need to handle
	// multiple directives and quoted strings.
	if maxAgeIndex := bytes.Index([]byte(cacheControl), []byte("max-age=")); maxAgeIndex != -1 {
		maxAgeStr := cacheControl[maxAgeIndex+8:]
		if commaIndex := bytes.IndexByte([]byte(maxAgeStr), ','); commaIndex != -1 {
			maxAgeStr = maxAgeStr[:commaIndex]
		}
		if maxAge, err := strconv.Atoi(maxAgeStr); err == nil {
			return maxAge
		}
	}
	return 0
}

func (cm *redisCacheAdapter) getHttpRes(cacheHttpRes cacheHttpResponse) (*http.Response, error) {
	header, err := JSONToHeader(cacheHttpRes.HeaderJSON)
	if err != nil {

		return nil, err
	}

	return &http.Response{
		Status:     cacheHttpRes.Status,
		StatusCode: cacheHttpRes.StatusCode,
		Header:     header,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(cacheHttpRes.Body))),
		Proto:      cacheHttpRes.Proto,
		ProtoMajor: cacheHttpRes.ProtoMajor,
		ProtoMinor: cacheHttpRes.ProtoMinor,
	}, nil
}

func JSONToHeader(stringStringJSON string) (http.Header, error) {
	var header http.Header
	err := json.Unmarshal([]byte(stringStringJSON), &header)
	if err != nil {
		return nil, err
	}
	return header, nil

}
