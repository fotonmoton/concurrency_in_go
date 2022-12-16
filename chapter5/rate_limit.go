package chapter5

import (
	"context"

	"golang.org/x/time/rate"
)

type ApiConnection struct {
	rateLimiter *rate.Limiter
}

func Open() *ApiConnection {
	return &ApiConnection{
		rateLimiter: rate.NewLimiter(rate.Limit(1), 1),
	}
}

func (a *ApiConnection) ReadFile(ctx context.Context) error {
	if err := a.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	return nil
}

func (a *ApiConnection) ResolveDomain(ctx context.Context) error {
	if err := a.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	return nil
}
