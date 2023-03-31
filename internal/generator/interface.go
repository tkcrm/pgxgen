package generator

import "context"

type IGenerator interface {
	Generate(ctx context.Context, args []string) error
}
