//nolint:testpackage
package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lifecyclePlan(t *testing.T, r resource.Resource, model any) tfsdk.Plan {
	t.Helper()

	var schemaResponse resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	require.False(t, plan.Set(t.Context(), model).HasError())

	return plan
}

func lifecycleIdentity(t *testing.T, r resource.Resource) *tfsdk.ResourceIdentity {
	t.Helper()

	var response resource.IdentitySchemaResponse

	identified, ok := r.(resource.ResourceWithIdentity)
	require.True(t, ok)
	identified.IdentitySchema(t.Context(), resource.IdentitySchemaRequest{}, &response)

	return &tfsdk.ResourceIdentity{Schema: response.IdentitySchema, Raw: tftypes.NewValue(response.IdentitySchema.Type().TerraformType(t.Context()), nil)}
}

func configureHTTPResource(t *testing.T, r resource.Resource, handler http.Handler) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	p := NewBrazeProvider("test", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	configured := configureTestProvider(t, p, nil, "fixture-key", false)
	require.False(t, configured.Diagnostics.HasError(), "%v", configured.Diagnostics)

	var response resource.ConfigureResponse

	configurable, ok := r.(resource.ResourceWithConfigure)
	require.True(t, ok)
	configurable.Configure(t.Context(), resource.ConfigureRequest{ProviderData: configured.ResourceData}, &response)
	require.False(t, response.Diagnostics.HasError())
}

func TestCreateRetainsIdentityWhenReadFails(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		r            resource.Resource
		model        any
		response, id string
	}{
		"SDK authentication key": {
			NewBrazeSDKAuthenticationKeyResource(),
			brazeSDKAuthenticationKeyModel{ID: types.StringUnknown(), AppID: types.StringValue("app-1"), RSAPublicKey: types.StringValue("public-key"), Description: types.StringValue("Key"), Primary: types.BoolUnknown()},
			`{"id":"created"}`, "created",
		},
		"content block": {
			NewBrazeContentBlockResource(),
			brazeContentBlockModel{IDIdentityModel: IDIdentityModel{ID: types.StringUnknown()}, Tags: types.ListNull(types.StringType), Name: types.StringValue("welcome"), Content: types.StringValue("Hello")},
			`{"content_block_id":"created","liquid_tag":"welcome","created_at":"2026-09-06T00:00:00Z","message":"success"}`, "created",
		},
		"email template": {
			NewBrazeEmailTemplateResource(),
			brazeEmailTemplateModel{IDIdentityModel: IDIdentityModel{ID: types.StringUnknown()}, Tags: types.ListNull(types.StringType), TemplateName: types.StringValue("welcome"), Subject: types.StringValue("Welcome"), Body: types.StringValue("Hello"), ShouldInlineCSS: types.BoolUnknown()},
			`{"email_template_id":"created","message":"success"}`, "created",
		},
		"catalog item": {
			NewBrazeCatalogItemResource(),
			brazeCatalogItemModel{ID: types.StringUnknown(), CatalogName: types.StringValue("products"), ItemID: types.StringValue("created"), ValuesJSON: jsontypes.NewNormalizedValue(`{"active":true}`)},
			`{"message":"success"}`, "products/created",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			creates := 0

			configureHTTPResource(t, test.r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.Method == http.MethodPost {
					creates++

					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(test.response))

					return
				}

				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"permission denied"}`))
			}))
			plan := lifecyclePlan(t, test.r, test.model)
			response := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}, Identity: lifecycleIdentity(t, test.r)}
			test.r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
			assert.Equal(t, 1, creates)
			require.True(t, response.Diagnostics.HasError())
			require.False(t, response.State.Raw.IsNull(), "A completed create must remain recoverable after a read error")

			var id types.String
			require.False(t, response.State.GetAttribute(t.Context(), path.Root("id"), &id).HasError())
			assert.Equal(t, test.id, id.ValueString())
			assert.True(t, response.State.Raw.IsFullyKnown())
		})
	}
}

func TestTemplatePlansExplainGeneratedIDsAndDestroy(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		r     resource.Resource
		model any
	}{
		"content block":  {NewBrazeContentBlockResource(), brazeContentBlockModel{IDIdentityModel: IDIdentityModel{ID: types.StringValue("existing")}, Tags: types.ListNull(types.StringType), Name: types.StringValue("welcome"), Content: types.StringValue("Hello")}},
		"email template": {NewBrazeEmailTemplateResource(), brazeEmailTemplateModel{IDIdentityModel: IDIdentityModel{ID: types.StringValue("existing")}, Tags: types.ListNull(types.StringType), TemplateName: types.StringValue("welcome"), Subject: types.StringValue("Welcome"), Body: types.StringValue("Hello")}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			modifier, ok := test.r.(resource.ResourceWithModifyPlan)
			require.True(t, ok, "Generated IDs and non-deleting destruction need plan diagnostics")
			plan := lifecyclePlan(t, test.r, test.model)
			state := tfsdk.State(plan)
			config := tfsdk.Config(plan)
			response := resource.ModifyPlanResponse{Plan: plan}
			modifier.ModifyPlan(t.Context(), resource.ModifyPlanRequest{Config: config, Plan: plan, State: tfsdk.State{Schema: plan.Schema}}, &response)
			assert.True(t, response.Diagnostics.HasError(), "Creating with a configured generated ID must fail before a POST")

			response = resource.ModifyPlanResponse{Plan: plan}
			modifier.ModifyPlan(t.Context(), resource.ModifyPlanRequest{Config: config, Plan: plan, State: state}, &response)
			assert.False(t, response.Diagnostics.HasError(), "Existing configurations that pin their actual ID must remain usable")

			destroy := tfsdk.Plan{Schema: plan.Schema}
			destroy.Raw = tftypes.NewValue(plan.Raw.Type(), nil)
			response = resource.ModifyPlanResponse{Plan: destroy}
			modifier.ModifyPlan(t.Context(), resource.ModifyPlanRequest{Plan: destroy, State: state}, &response)
			assert.False(t, response.Diagnostics.HasError())
			require.Len(t, response.Diagnostics.Warnings(), 1)
			assert.Contains(t, response.Diagnostics.Warnings()[0].Detail(), "Terraform state")
			assert.True(t, response.Plan.Raw.IsNull())

			deletion := resource.DeleteResponse{State: state}
			test.r.Delete(t.Context(), resource.DeleteRequest{State: state}, &deletion)
			require.False(t, deletion.Diagnostics.HasError())
			require.Len(t, deletion.Diagnostics.Warnings(), 1)
			assert.Contains(t, deletion.Diagnostics.Warnings()[0].Detail(), "Terraform state")
		})
	}
}

func TestReadOnlyRemovesConfirmedMissingCatalogItems(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		status  int
		body    string
		missing bool
	}{
		"documented not found": {http.StatusNotFound, `{"errors":[{"id":"item-not-found","message":"Could not find item","parameters":["item_id"],"parameter_values":["existing"]}],"message":"Invalid Request"}`, true},
		"forbidden":            {http.StatusForbidden, `{"message":"permission denied"}`, false},
		"unauthorized":         {http.StatusUnauthorized, `{"message":"invalid API key"}`, false},
		"malformed not found":  {http.StatusNotFound, `<html>gateway error</html>`, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := NewBrazeCatalogItemResource()
			configureHTTPResource(t, r, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			plan := lifecyclePlan(t, r, brazeCatalogItemModel{ID: types.StringValue("products/existing"), CatalogName: types.StringValue("products"), ItemID: types.StringValue("existing"), ValuesJSON: jsontypes.NewNormalizedValue(`{}`)})
			state := tfsdk.State(plan)
			response := resource.ReadResponse{State: state, Identity: lifecycleIdentity(t, r)}
			r.Read(t.Context(), resource.ReadRequest{State: state}, &response)
			assert.Equal(t, test.missing, response.State.Raw.IsNull(), "%v", response.Diagnostics)
			assert.Equal(t, !test.missing, response.Diagnostics.HasError(), "%v", response.Diagnostics)
		})
	}
}

func TestReadRejectsUnexpectedIdentity(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		r     resource.Resource
		model any
		body  string
	}{
		"content block":  {NewBrazeContentBlockResource(), brazeContentBlockModel{IDIdentityModel: IDIdentityModel{ID: types.StringValue("expected")}, Tags: types.ListNull(types.StringType), Name: types.StringValue("welcome"), Content: types.StringValue("Hello")}, `{"content_block_id":"other","name":"welcome","content":"Hello"}`},
		"email template": {NewBrazeEmailTemplateResource(), brazeEmailTemplateModel{IDIdentityModel: IDIdentityModel{ID: types.StringValue("expected")}, Tags: types.ListNull(types.StringType), TemplateName: types.StringValue("welcome"), Subject: types.StringValue("Welcome"), Body: types.StringValue("Hello")}, `{"email_template_id":"other","template_name":"welcome","subject":"Welcome","body":"Hello"}`},
		"catalog item":   {NewBrazeCatalogItemResource(), brazeCatalogItemModel{ID: types.StringValue("products/expected"), CatalogName: types.StringValue("products"), ItemID: types.StringValue("expected"), ValuesJSON: jsontypes.NewNormalizedValue(`{}`)}, `{"items":[{"id":"other"}],"message":"success"}`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			configureHTTPResource(t, test.r, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(test.body))
			}))
			plan := lifecyclePlan(t, test.r, test.model)
			state := tfsdk.State(plan)
			response := resource.ReadResponse{State: state, Identity: lifecycleIdentity(t, test.r)}
			test.r.Read(t.Context(), resource.ReadRequest{State: state}, &response)
			assert.True(t, response.Diagnostics.HasError())
			assert.True(t, state.Raw.Equal(response.State.Raw), "A mismatched response must not change state")
		})
	}
}

func TestStructuredErrorsExcludeParameterPayloads(t *testing.T) {
	t.Parallel()

	r := NewBrazeCatalogItemResource()
	configureHTTPResource(t, r, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"id":"invalid-fields","message":"payload secret-message","parameter_values":["secret-payload"]}],"message":"Invalid Request"}`))
	}))
	plan := lifecyclePlan(t, r, brazeCatalogItemModel{CatalogName: types.StringValue("products"), ItemID: types.StringValue("one"), ValuesJSON: jsontypes.NewNormalizedValue(`{}`)})
	response := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}, Identity: lifecycleIdentity(t, r)}
	r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
	require.True(t, response.Diagnostics.HasError())
	detail := response.Diagnostics.Errors()[0].Detail()
	assert.Contains(t, detail, "HTTP 400")
	assert.Contains(t, detail, "invalid-fields")
	assert.NotContains(t, detail, "secret-payload")
	assert.NotContains(t, detail, "secret-message")
}

func TestDeleteToleratesConfirmedAbsenceOnly(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		r     resource.Resource
		model any
	}{
		"catalog": {NewBrazeCatalogResource(), brazeCatalogModel{Name: types.StringValue("products"), Fields: types.ListNull(BrazeCatalogFieldObjectType())}},
		"item":    {NewBrazeCatalogItemResource(), brazeCatalogItemModel{CatalogName: types.StringValue("products"), ItemID: types.StringValue("one"), ValuesJSON: jsontypes.NewNormalizedValue(`{}`)}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
				configureHTTPResource(t, test.r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodDelete, r.Method)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"errors":[{"id":"item-not-found","message":"Could not find item"}],"message":"Invalid Request"}`))
				}))
				plan := lifecyclePlan(t, test.r, test.model)
				state := tfsdk.State(plan)
				response := resource.DeleteResponse{State: state}
				test.r.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)
				assert.Equal(t, status != http.StatusNotFound, response.Diagnostics.HasError())
			}
		})
	}
}
