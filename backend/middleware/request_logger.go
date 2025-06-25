package middleware

import (
	"log"
	"time"

	"github.com/labstack/echo/v5"
)

// RequestLogger creates middleware that logs all incoming requests
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			
			// Log request details
			log.Printf("[REQUEST] %s %s from %s", 
				c.Request().Method, 
				c.Request().URL.String(), 
				c.RealIP())
			
			// Log headers (excluding sensitive ones)
			log.Printf("[REQUEST] Headers: User-Agent=%s, Content-Type=%s", 
				c.Request().Header.Get("User-Agent"),
				c.Request().Header.Get("Content-Type"))
			
			// Log if there's an authorization header (but not the value)
			if authHeader := c.Request().Header.Get("Authorization"); authHeader != "" {
				if len(authHeader) > 20 {
					log.Printf("[REQUEST] Authorization: %s...%s", authHeader[:7], authHeader[len(authHeader)-4:])
				} else {
					log.Printf("[REQUEST] Authorization: [REDACTED]")
				}
			}
			
			// Process request
			err := next(c)
			
			// Log response details
			duration := time.Since(start)
			status := c.Response().Status
			
			log.Printf("[RESPONSE] %s %s - Status: %d, Duration: %v", 
				c.Request().Method, 
				c.Request().URL.String(), 
				status, 
				duration)
			
			return err
		}
	}
} 