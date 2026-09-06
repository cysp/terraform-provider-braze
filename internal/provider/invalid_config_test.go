package provider_test

import (
	"net/http"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccBrazeInvalidNamesFailDuringPlan(t *testing.T) {
	t.Parallel()

	for name, configuration := range map[string]string{
		"content block": `resource "braze_content_block" "test" {
 name = ""` + "\n" + `content = "Hello"
}`,
		"email template": `resource "braze_email_template" "test" {
 template_name = ""` + "\n" + `subject = "Welcome"` + "\n" + `body = "Hello"
}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"name is required"}`))
			})
			BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{{Config: configuration, ExpectError: regexp.MustCompile("Invalid Attribute Value")}}})
			assert.Zero(t, calls.Load(), "invalid names must fail before an API mutation")
		})
	}
}
