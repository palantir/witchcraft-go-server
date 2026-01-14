package witchcraft

import (
	"context"

	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-tasks/function"
	"github.com/palantir/witchcraft-go-tasks/runnable"
)

type RunnableManager interface {
	AddNamedRunnable(ctx context.Context, namedRunnables ...function.NamedRunnable)
}

type defaultRunnableManager struct {
	serverShutdown func(ctx context.Context)
}

func NewRunnableManager(serverShutdown func(ctx context.Context)) RunnableManager {
	return &defaultRunnableManager{
		serverShutdown: serverShutdown,
	}
}

func (d *defaultRunnableManager) AddNamedRunnable(ctx context.Context, namedRunnables ...function.NamedRunnable) {
	for _, runnable := range namedRunnables {
		d.startRunnable(ctx, runnable)
	}
}

func (d *defaultRunnableManager) startRunnable(ctx context.Context, runnableArg function.NamedRunnable) {
	finalRunnable := runnable.WithWrappers(
		runnable.WithServiceLogging(),
		runnable.WithFatalLogging(),
	)(runnableArg)
	go func() {
		err := finalRunnable.Run(ctx)
		if err != nil {
			svc1log.FromContext(ctx).Error("Terminal runnable returned an error; shutting down server", svc1log.Stacktrace(err))
			d.serverShutdown(ctx)
			return
		}
		svc1log.FromContext(ctx).Error("Terminal runnable unexpectedly terminated, shutting down server")
		d.serverShutdown(ctx)
		return
	}()

}
