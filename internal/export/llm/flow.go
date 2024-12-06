package llm

import "context"

type Task interface {
	Execute(context context.Context) error
}
