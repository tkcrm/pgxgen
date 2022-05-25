package generator

type IGenerator interface {
	Generate(args []string) error
}
