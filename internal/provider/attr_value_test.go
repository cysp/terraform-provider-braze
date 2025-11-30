package provider_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

type attrValueTypeNil struct {
	attr.Value
}

var _ attr.Value = attrValueTypeNil{}

//nolint:ireturn
func (t attrValueTypeNil) Type(_ context.Context) attr.Type {
	return nil
}
