package build

import (
	"github.com/elishacatherasoo/rain/cft"
	"github.com/elishacatherasoo/rain/cft/spec"
)

// iamBuilder contains specific code for building IAM policies
type iamBuilder struct {
	builder
}

// newIamBuilder creates a new iamBuilder
func newIamBuilder() iamBuilder {
	var b iamBuilder
	b.Spec = spec.Iam

	return b
}

// Policy generates a an IAM policy body
func (b iamBuilder) Policy() (interface{}, []*cft.Comment) {
	b.tracker = newTracker()
	return b.newPropertyType("", "Policy")
}
