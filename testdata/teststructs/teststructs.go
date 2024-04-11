package teststructs

import (
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/internal/ver"
)

type SomeType int

type TestStruct struct {
	ID               string
	Data             []byte
	SomeType         SomeType
	SF               structs.StructField
	AnotherField     structs.Types
	SomeExternalType ver.CheckLastestReleaseVersionResponse
}
