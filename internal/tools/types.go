package tools

import "context"

type CallInput struct {
	Arguments map[string]string
}

type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, input CallInput) (string, error)
}
