package complaint

import "context"

type Repository interface {
	ListDelayedProgramComplainers(ctx context.Context) ([]DelayedProgramComplainer, error)
}
