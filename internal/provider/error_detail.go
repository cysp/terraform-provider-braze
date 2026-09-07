package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
)

func detailFromError(err error) string {
	if err == nil {
		return ""
	}

	if response, ok := errors.AsType[*brazeclient.ErrorResponseStatusCode](err); ok {
		codes := make([]string, 0, len(response.Response.Errors))
		for _, raw := range response.Response.Errors {
			var detail struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(raw, &detail) == nil && detail.ID != "" {
				codes = append(codes, detail.ID)
			}
		}

		message := fmt.Sprintf("Braze returned HTTP %d: %s", response.StatusCode, response.Response.Message)
		if len(codes) > 0 {
			message += " (" + strings.Join(codes, ", ") + ")"
		}

		return message + "."
	}

	return err.Error()
}
