package main

// Forked from https://raw.githubusercontent.com/LYY/echo-middleware/master/nocache.go

import (
	"time"

	"github.com/labstack/echo"
	emw "github.com/labstack/echo/middleware"
)

type (
	// NoCacheConfig defines the config for nocache middleware.
	NoCacheConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper emw.Skipper
	}
)

var (
	// Unix epoch time
	epoch = time.Unix(0, 0).Format(time.RFC1123)

	// Taken from https://github.com/mytrile/nocache
	noCacheHeaders = map[string]string{
		"Expires":       epoch,
		"Cache-Control": "no-cache, private, max-age=0",
		"Pragma":        "no-cache",
	}
	// DefaultNoCacheConfig is the default nocache middleware config.
	DefaultNoCacheConfig = NoCacheConfig{
		Skipper: emw.DefaultSkipper,
	}
)

// NoCache is a simple piece of middleware that sets a number of HTTP headers to prevent
// a router (or subrouter) from being cached by an upstream proxy and/or client.
//
// As per http://wiki.nginx.org/HttpProxyModule - NoCache sets:
//      Expires: Thu, 01 Jan 1970 00:00:00 UTC
//      Cache-Control: no-cache, private, max-age=0
//      Pragma: no-cache (for HTTP/1.0 proxies/clients)
func NoCache() echo.MiddlewareFunc {
	return NoCacheWithConfig(DefaultNoCacheConfig)
}

// NoCacheWithConfig returns a nocache middleware with config.
func NoCacheWithConfig(config NoCacheConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultNoCacheConfig.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}
			// Set our NoCache headers
			res := c.Response()
			for k, v := range noCacheHeaders {
				res.Header().Set(k, v)
			}

			return next(c)
		}
	}
}
