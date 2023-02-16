// Package conf is responsible for modeling the application configuration. This
// includes configuration for any dependencies. The construct package will be
// responsible for mapping or using these configurations to build instances
// of those dependencies.
//
// This package use the conf package at github.com/rsb/conf which allows you
// to describe env and cli var using annotated tags
package conf

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type PrologAPI struct {
	Version
	API
	Kubernetes
}

type Version struct {
	Build string `conf:"env:PROLOG_API_BUILD_VERSION, cli:api-build-version,cli-u:version of the web api"`
	Desc  string `conf:"env:PROLOG_API_BUILD_DESC, cli:api-build-desc, cli-u:summary of the build"`
}

type API struct {
	Host            string        `conf:"env:PROLOG_API_HOST, cli:api-host, default:0.0.0.0:3000, cli-u:web api host"`
	DebugHost       string        `conf:"env:PROLOG_API_DEBUG_HOST, cli:debug-host, default:0.0.0.0:4000, cli-u:debug host"`
	IsCaseSensitive bool          `conf:"env:PROLOG_API_ROUTE_CASE_SENSITIVE, cli:api-route-case-sensitive, default:false, cli-u:will routes be case sensitive"`
	IsETag          bool          `conf:"env:PROLOG_API_ETAG, cli:api-etag, default:false, cli-u:enable/disable etag header generation"`
	ReadTimeout     time.Duration `conf:"env:PROLOG_API_READ_TIMEOUT,cli:api-read-timeout, default:5s"`
	WriteTimeout    time.Duration `conf:"env:PROLOG_API_WRITE_TIMEOUT,cli:api-write-timeout, default:20s"`
	IdleTimeout     time.Duration `conf:"env:PROLOG_API_IDLE_TIMEOUT, cli:api-idle-timeout, default:120s"`
	ShutdownTimeout time.Duration `conf:"env:PROLOG_API_SHUTDOWN_TIMEOUT,cli:api-shutdown-timeout, default:20s"`
}

func (a API) NewFiberConfig() fiber.Config {
	config := fiber.Config{
		IdleTimeout:   a.IdleTimeout,
		ReadTimeout:   a.ReadTimeout,
		WriteTimeout:  a.WriteTimeout,
		CaseSensitive: a.IsCaseSensitive,
		ETag:          a.IsETag,
	}

	return config
}

type HTTPClient struct {
	Timeout            time.Duration `conf:"default: 5s,  env:LOLA_HTTP_CLIENT_TIMEOUT, cli:http-client-timeout, cli-u:timeout for http clients"`
	MaxIdleConn        int           `conf:"default: 100, env:LOLA_HTTP_CLIENT_MAX_IDLE_CONN, cli:http-client-max-idle-con, cli-u:http client max idle connections"`
	MaxConnPerHost     int           `conf:"default: 100, env:LOLA_HTTP_CLIENT_MAX_CONN_PER_HOST, cli:http-client-max-con-per-host, cli-u:http client max connection per host"`
	MaxIdleConnPerHost int           `conf:"default: 100, env:LOLA_HTTP_CLIENT_MAX_IDLE_PER_HOST, cli:http-client-max-idle-per-host, cli-u:http client max idle connections per host"`
}

type Kubernetes struct {
	Pod       string `conf:"env:KUBERNETES_PODNAME"`
	PodIP     string `conf:"env:KUBERNETES_NAMESPACE_POD_IP"`
	Node      string `conf:"env:KUBERNETES_NODENAME"`
	Namespace string `conf:"env:KUBERNETES_NAMESPACE"`
}
