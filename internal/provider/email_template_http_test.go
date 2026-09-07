package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
)

func TestAccBrazeEmailTemplateCSSDefault(t *testing.T) {
	t.Parallel()

	var inlineCSS atomic.Bool
	inlineCSS.Store(true)

	var observedCSS atomic.Bool

	var subject atomic.Value
	subject.Store("Welcome")

	configuration := func(css, subject string) string {
		return fmt.Sprintf(`resource "braze_email_template" "test" {
  template_name = "welcome"
  subject = %q
  body = "Hello"
  %s
}`, subject, css)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/templates/email/create", "/templates/email/update":
			var body map[string]json.RawMessage

			err := json.NewDecoder(r.Body).Decode(&body)
			if !assert.NoError(t, err) {
				w.WriteHeader(http.StatusBadRequest)

				return
			}

			var requestSubject string
			if !assert.NoError(t, json.Unmarshal(body["subject"], &requestSubject)) {
				w.WriteHeader(http.StatusBadRequest)

				return
			}

			subject.Store(requestSubject)

			if r.URL.Path == "/templates/email/create" {
				assert.NotContains(t, body, "should_inline_css", "Omission must use the Braze workspace default")
				w.WriteHeader(http.StatusCreated)
			} else {
				observedCSS.Store(true)
			}

			if valueJSON, exists := body["should_inline_css"]; exists {
				var value bool
				if assert.NoError(t, json.Unmarshal(valueJSON, &value)) {
					inlineCSS.Store(value)
				}
			} else {
				// Braze also uses the AppGroup default when update omits this field.
				// https://www.braze.com/docs/api/endpoints/templates/email_templates/post_update_email_template
				inlineCSS.Store(true)
			}

			_, _ = w.Write([]byte(`{"email_template_id":"welcome-id","message":"success"}`))
		case "/templates/email/info":
			// Exercise a missing optional observation before the first update.
			if !observedCSS.Load() {
				_, _ = fmt.Fprintf(w, `{"email_template_id":"welcome-id","template_name":"welcome","subject":%q,"body":"Hello"}`, subject.Load())

				return
			}

			_, _ = fmt.Fprintf(w, `{"email_template_id":"welcome-id","template_name":"welcome","subject":%q,"body":"Hello","should_inline_css":%t}`, subject.Load(), inlineCSS.Load())
		default:
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{
		{Config: configuration("", "Welcome"), Check: resource.TestCheckNoResourceAttr("braze_email_template.test", "should_inline_css")},
		{Config: configuration("", "Default subject"), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "true")},
		{Config: configuration("should_inline_css = false", "Welcome"), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "false")},
		{Config: configuration("", "Welcome"), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "false")},
		{Config: configuration("", "Updated subject"), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "false")},
		{
			Config: configuration("should_inline_css = terraform_data.css.output", "Updated subject") + `
resource "terraform_data" "css" { input = true }
`,
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectUnknownValue("braze_email_template.test", tfjsonpath.New("should_inline_css")),
			}},
			Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "true"),
		},
		{ResourceName: "braze_email_template.test", ImportState: true, ImportStateVerify: true},
	}})
}
