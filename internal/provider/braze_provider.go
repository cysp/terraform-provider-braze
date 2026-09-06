package provider

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type brazeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string

	baseURL    string
	apiKey     string
	httpClient *http.Client
}

//revive:disable:unexported-return
func NewBrazeProvider(version string, options ...BrazeProviderOption) *brazeProvider {
	provider := brazeProvider{
		version: version,
	}

	for _, option := range options {
		option(&provider)
	}

	return &provider
}

func Factory(version string, options ...BrazeProviderOption) func() provider.Provider {
	return func() provider.Provider {
		return NewBrazeProvider(version, options...)
	}
}

var (
	_ provider.Provider                  = (*brazeProvider)(nil)
	_ provider.ProviderWithListResources = (*brazeProvider)(nil)
)

type brazeProviderModel struct {
	BaseURL types.String `tfsdk:"base_url"`
	APIKey  types.String `tfsdk:"api_key"`
}

func (p *brazeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "braze"
	resp.Version = p.version
}

func (p *brazeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Braze configuration.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Description: "The absolute REST API URL for your Braze instance, for example https://rest.iad-01.braze.com. This must be set; there is no default instance.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The REST API key to use when communicating with Braze. If not provided, it will default to the value of the BRAZE_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *brazeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data brazeProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.BaseURL.IsUnknown() || data.APIKey.IsUnknown() {
		if req.ClientCapabilities.DeferralAllowed {
			resp.Deferred = &provider.Deferred{Reason: provider.DeferredReasonProviderConfigUnknown}
		} else {
			for name, value := range map[string]types.String{"base_url": data.BaseURL, "api_key": data.APIKey} {
				if value.IsUnknown() {
					resp.Diagnostics.AddAttributeError(path.Root(name), "Unknown Braze configuration", "The provider needs a known "+name+" before it can connect to Braze.")
				}
			}
		}

		return
	}

	baseURL := p.baseURL
	if !data.BaseURL.IsNull() {
		baseURL = data.BaseURL.ValueString()
	}

	apiKey := p.apiKey
	if value := os.Getenv("BRAZE_API_KEY"); value != "" {
		apiKey = value
	}

	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	if !validBrazeURL(baseURL) {
		resp.Diagnostics.AddAttributeError(path.Root("base_url"), "Invalid Braze API URL", "Set base_url to the absolute HTTP or HTTPS REST API URL for your Braze instance, without credentials, a query, or a fragment.")
	}

	if strings.TrimSpace(apiKey) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing Braze API key", "Set api_key in the provider configuration or set the BRAZE_API_KEY environment variable. An explicitly empty api_key does not use the environment fallback.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	brazeClient, err := brazeclient.NewClient(
		baseURL,
		NewBrazeAPIKeySecuritySource(apiKey),
		brazeclient.WithClient(NewHTTPClientWithUserAgent(newBrazeHTTPClient(p.httpClient), "terraform-provider-braze/"+p.version)),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Braze client", "The Braze client could not be initialized. Check the provider configuration.")

		return
	}

	providerData := brazeProviderData{
		contentBlocks:                  newGeneratedContentBlockClient(brazeClient),
		emailTemplates:                 newGeneratedEmailTemplateClient(brazeClient),
		catalogs:                       newGeneratedCatalogClient(brazeClient),
		catalogItems:                   newGeneratedCatalogItemClient(brazeClient),
		sdkAuthenticationKeys:          newGeneratedSDKAuthenticationKeyClient(brazeClient),
		sdkAuthenticationKeyCollection: newGeneratedSDKAuthenticationKeysClient(brazeClient),
	}

	resp.ListResourceData = providerData
	resp.ResourceData = providerData
}

func (p *brazeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *brazeProvider) ListResources(context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		NewBrazeCatalogListResource,
		NewBrazeCatalogItemListResource,
		NewBrazeContentBlockListResource,
		NewBrazeEmailTemplateListResource,
	}
}

func (p *brazeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBrazeCatalogResource,
		NewBrazeCatalogItemResource,
		NewBrazeContentBlockResource,
		NewBrazeEmailTemplateResource,
		NewBrazeSDKAuthenticationKeyResource,
		NewBrazeSDKAuthenticationKeysResource,
	}
}

func validBrazeURL(baseURL string) bool {
	endpoint, err := url.Parse(baseURL)

	return err == nil && endpoint.Hostname() != "" && (endpoint.Scheme == "https" || endpoint.Scheme == "http") && endpoint.User == nil && endpoint.RawQuery == "" && !endpoint.ForceQuery && endpoint.Fragment == ""
}
