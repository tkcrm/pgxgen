package structs

type StructParameters struct {
	ExternalPackage string
	Name            string
	Imports         []string
	Fields          []*StructField

	originalName string
}

func (s *StructParameters) ExistFieldIndex(name string) int {
	for index, f := range s.Fields {
		if f.Name == name {
			return index
		}
	}
	return -1
}

type TypesParameters struct {
	Name string
	Type string
}
