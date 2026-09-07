package provider_test

import (
	"fmt"
	"net/http"
	"regexp"
	"sync/atomic"
	"testing"

	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBrazeInvalidNamesFailDuringPlan(t *testing.T) {
	t.Parallel()

	for name, configuration := range map[string]string{
		"content block": `resource "braze_content_block" "test" {
 name = %[1]q
 content = "Hello"
}`,
		"email template": `resource "braze_email_template" "test" {
 template_name = %[1]q
 subject = "Welcome"
 body = "Hello"
}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server, err := brazeclienttesting.NewBrazeServer()
			require.NoError(t, err)

			var mutations atomic.Int32

			handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					mutations.Add(1)
				}

				server.ServeHTTP(w, req)
			})
			invalid := fmt.Sprintf(configuration, "")
			valid := fmt.Sprintf(configuration, "welcome")
			BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{
				{Config: invalid, ExpectError: regexp.MustCompile("Invalid Attribute Value")},
				{PreConfig: func() {
					assert.Zero(t, mutations.Load(), "invalid create must fail before an API mutation")
				}, Config: valid},
				{PreConfig: func() {
					assert.EqualValues(t, 1, mutations.Swap(0), "the valid configuration must create one object")
				}, Config: invalid, ExpectError: regexp.MustCompile("Invalid Attribute Value")},
				{PreConfig: func() {
					assert.Zero(t, mutations.Load(), "invalid update must fail before an API mutation")
				}, Config: valid, ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				}},
			}})
			assert.Zero(t, mutations.Load(), "the rejected update must leave the original object unchanged")
		})
	}
}
