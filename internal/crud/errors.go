package crud

import "fmt"

var ErrUndefinedPrimaryColumn = fmt.Errorf("undefined primary column")
var ErrWhileProcessTemplate = "error while process \"%s\" method for table \"%s\""
