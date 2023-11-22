package sdk

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc/pattern"
	"github.com/tkcrm/pgxgen/pkg/sqlc/plugin"
)

func DataType(n *plugin.Identifier) string {
	if n.Schema != "" {
		return n.Schema + "." + n.Name
	} else {
		return n.Name
	}
}

func MatchString(pat, target string) bool {
	matcher, err := pattern.MatchCompile(pat)
	if err != nil {
		panic(err)
	}
	return matcher.MatchString(target)
}

func SameTableName(tableID, f *plugin.Identifier, defaultSchema string) bool {
	if tableID == nil {
		return false
	}
	schema := tableID.Schema
	if tableID.Schema == "" {
		schema = defaultSchema
	}
	return tableID.Catalog == f.Catalog && schema == f.Schema && tableID.Name == f.Name
}
