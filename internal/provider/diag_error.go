package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type diagnosticsError struct {
	diags diag.Diagnostics
}

func diagError(diags diag.Diagnostics) error {
	return diagnosticsError{diags: diags}
}

func (e diagnosticsError) Error() string {
	messages := make([]string, 0, len(e.diags))
	for _, d := range e.diags {
		messages = append(messages, d.Summary()+": "+d.Detail())
	}

	return strings.Join(messages, "; ")
}
