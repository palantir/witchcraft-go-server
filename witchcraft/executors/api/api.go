package api

import "context"

type Executor interface {
	Run(ctx context.Context) error
}

type NamedExecutor interface {
	Executor
	GetName() string
}
