//go:build !windows && cgo
// +build !windows,cgo

package postgresql

import (
	nodes "github.com/pganalyze/pg_query_go/v5"
)

var Parse = nodes.Parse
var Fingerprint = nodes.Fingerprint
