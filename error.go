package gadb

import "strings"

// ErrWarnings represents a list of warnings
type ErrWarnings []string

func (e ErrWarnings) Error() string {
	return "warnings: " + strings.Join(e, ", ")
}
