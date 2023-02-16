package schema

type TableMetaData struct {
	Name    string
	Columns []string
}

type Tables map[string]*TableMetaData

func (t Tables) GetTableMetaData(tableName string) *TableMetaData {
	for name, metaData := range t {
		if name == tableName {
			return metaData
		}
	}
	return nil
}
