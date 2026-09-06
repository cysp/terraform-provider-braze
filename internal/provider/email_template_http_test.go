package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccBrazeEmailTemplateCSSDefault(t *testing.T) {
	t.Parallel()

	var inlineCSS atomic.Bool
	inlineCSS.Store(true)

	configuration := func(css string) string {
		return fmt.Sprintf(`resource "braze_email_template" "test" {
  template_name = "welcome"
  subject = "Welcome"
  body = "Hello"
  %s
}`, css)
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

			if r.URL.Path == "/templates/email/create" {
				assert.NotContains(t, body, "should_inline_css", "Omission must use the Braze workspace default")
				w.WriteHeader(http.StatusCreated)
			} else {
				var value bool
				if assert.NoError(t, json.Unmarshal(body["should_inline_css"], &value)) {
					inlineCSS.Store(value)
				}
			}

			_, _ = w.Write([]byte(`{"email_template_id":"welcome-id","message":"success"}`))
		case "/templates/email/info":
			_, _ = fmt.Fprintf(w, `{"email_template_id":"welcome-id","template_name":"welcome","subject":"Welcome","body":"Hello","should_inline_css":%t}`, inlineCSS.Load())
		default:
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	BrazeProviderMockedResourceTest(t, handler, resource.TestCase{Steps: []resource.TestStep{
		{Config: configuration(""), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "true")},
		{Config: configuration("should_inline_css = false"), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "false")},
		{Config: configuration(""), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "false")},
		{Config: configuration("should_inline_css = true"), Check: resource.TestCheckResourceAttr("braze_email_template.test", "should_inline_css", "true")},
		{ResourceName: "braze_email_template.test", ImportState: true, ImportStateVerify: true},
	}})
}
