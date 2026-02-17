package gitconfig

import "fmt"

// Ensure Configs implements fmt.Stringer at compile time.
var _ fmt.Stringer = (*Configs)(nil)
