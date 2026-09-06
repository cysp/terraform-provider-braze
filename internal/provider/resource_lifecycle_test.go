//nolint:testpackage
package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	return &tfsdk.ResourceIdentity{Schema: response.IdentitySchema}
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
