package utils

import (
	"time"

	"github.com/labstack/echo"
	"github.com/rs/zerolog"
)

func ZeroLogger(log *zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()

			id := req.Header.Get(echo.HeaderXRequestID)
			if id == "" {
				id = res.Header().Get(echo.HeaderXRequestID)
			}

			var level zerolog.Level
			n := res.Status
			switch {
			case n >= 500:
				level = zerolog.ErrorLevel
			case n >= 400:
				level = zerolog.WarnLevel
			case n >= 300:
				level = zerolog.InfoLevel
			default:
				level = zerolog.DebugLevel
			}

			log.WithLevel(level).
				Int("status", res.Status).
				Str("latency", time.Since(start).String()).
				Str("id", id).
				Str("method", req.Method).
				Str("uri", req.RequestURI).
				Str("host", req.Host).
				Str("remote_ip", c.RealIP()).
				Msg("request")

			return nil
		}
	}
}
