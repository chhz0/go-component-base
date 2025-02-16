package fields

import "github.com/chhz0/go-component-base/pkg/meta/selection"

type Requirements []Requirement

type Requirement struct {
	Operator selection.Operator
	Field    string
	Value    string
}
