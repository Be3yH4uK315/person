package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/labstack/echo/v5"
)

type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic recovered: %v", e.Value)
}

func Recovery() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					err = &PanicError{Value: recovered, Stack: debug.Stack()}
				}
			}()
			return next(c)
		}
	}
}
