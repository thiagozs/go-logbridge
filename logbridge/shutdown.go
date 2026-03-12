package logbridge

import "context"

type shutdowner interface {
	Shutdown(context.Context) error
}

func Shutdown(ctx context.Context, logger Logger) error {
	if shutdownable, ok := logger.(shutdowner); ok {
		return shutdownable.Shutdown(ctx)
	}

	return nil
}
