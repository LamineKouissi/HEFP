package main

import (
	"context"
	"log"
	"os"

	"github.com/LamineKouissi/LHP/adapters"
	"github.com/LamineKouissi/LHP/filters"
	"github.com/LamineKouissi/LHP/filters/connectors"
	"github.com/LamineKouissi/LHP/listeners"
	"github.com/LamineKouissi/LHP/routers"
	"github.com/LamineKouissi/LHP/routers/routes"
)

var (
	httpFilterChaine    filters.Filter
	httpFilterChaineErr error
	httpRoute           *routes.HttpRoute
	httpsRoute          *routes.HttpsRoute
	mainHttpRouter      *routers.ForwardProxyRouter
)

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%v env variable does not exist", key)
	}
	return value
}

func init() {
	//test httpFilterChaine : cacheFilter(imp : HasNextFilter & Filter ) > transformerFilter(imp : HasNextFilter & Filter ) > httpsCnx(imp : Filter )
	cnx := context.Background()
	httpsCnxFilter, err := connectors.NewHttpsConnector()
	if err != nil {
		panic(err)
	}

	transformerFilter, err := filters.NewHttpMsgTransformerFilter(httpsCnxFilter)
	if err != nil {
		panic(err)
	}

	redisCacheAdapter, err := adapters.NewRedisCacheAdapter("localhost:6379", "", "", "0")
	if err != nil {
		panic(err)
	}

	cacheFilter, err := filters.NewCacheMgrFilter(redisCacheAdapter)
	if err != nil {
		panic(err)
	}

	hasNextFilterChaine := []filters.HasNextFilter{cacheFilter, transformerFilter}

	httpFilterChaine, httpFilterChaineErr = filters.ConstructFilterChain(cnx, hasNextFilterChaine, httpsCnxFilter)
	if httpFilterChaineErr != nil {
		log.Fatal(httpFilterChaineErr)
	}

	httpRoute, err = routes.NewHttpRoute(httpFilterChaine)
	if err != nil {
		panic(err)
	}

	httpsRoute, err = routes.NewHttspRoute()
	if err != nil {
		panic(err)
	}
	mainHttpRouter, err = routers.NewForwardProxyRouter(*httpsRoute, *httpRoute)
	if err != nil {
		panic(err)
	}

}

func main() {
	//StartTLSServer()
	address := ":7000"
	ctx := context.Background()
	tlsListener, err := listeners.NewTLSListener(ctx, address, mainHttpRouter, getEnv("TLS_SERVER"), getEnv("TLS_KEY"))
	if err != nil {
		panic(err)
	}
	err = tlsListener.Listen()
	if err != nil {
		panic(err)
	}

}
