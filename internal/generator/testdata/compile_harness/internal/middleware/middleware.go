// Package middleware is a compile-harness stub.
package middleware

import "github.com/labstack/echo/v4"

// JWT returns a no-op echo middleware. The real project enforces auth here.
func JWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}
