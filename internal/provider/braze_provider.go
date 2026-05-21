package provider

import (
	"context"
	"net/http"
	"os"
	"time"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
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
				Description: "The base URL associated with your Braze instance's REST API.",
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

	var baseURL string
	if !data.BaseURL.IsNull() {
		baseURL = data.BaseURL.ValueString()
	}

	if baseURL == "" {
		baseURL = p.baseURL
	}

	var apiKey string
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	} else {
		if apiKeyFromEnv, found := os.LookupEnv("BRAZE_API_KEY"); found {
			apiKey = apiKeyFromEnv
		}
	}

	if apiKey == "" {
		apiKey = p.apiKey
	}

	if resp.Diagnostics.HasError() {
		return
	}

	retryableClient := retryablehttp.NewClient()
	retryableClient.RetryWaitMin = time.Duration(1) * time.Second
	retryableClient.RetryWaitMax = time.Duration(3) * time.Second //nolint:mnd
	retryableClient.Backoff = retryablehttp.LinearJitterBackoff

	if p.httpClient != nil {
		retryableClient.HTTPClient = p.httpClient
	}

	brazeClient, err := brazeclient.NewClient(
		baseURL,
		NewBrazeAPIKeySecuritySource(apiKey),
		brazeclient.WithClient(NewHTTPClientWithUserAgent(retryableClient.StandardClient(), "terraform-provider-braze/"+p.version)),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Braze client", err.Error())
	}

	providerData := brazeProviderData{
		client: brazeClient,
	}

	resp.ActionData = providerData
	resp.DataSourceData = providerData
	resp.EphemeralResourceData = providerData
	resp.ListResourceData = providerData
	resp.ResourceData = providerData
}

func (p *brazeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *brazeProvider) ListResources(context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		NewBrazeContentBlockListResource,
	}
}

func (p *brazeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBrazeContentBlockResource,
		NewBrazeEmailTemplateResource,
	}
}
