package provider_test

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/stretchr/testify/require"
)

// Exercise query generation itself: terraform-plugin-testing v1.16.0 ignores
// GenerateConfig in Query steps, so those steps cannot prove this round trip.
//
//nolint:gosec // Executable paths come from the local test harness; commands never use a shell.
func TestAccQueryGeneratedConfigurationImportsCleanly(t *testing.T) {
	t.Parallel()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 for Terraform CLI integration")
	}

	binary := os.Getenv("TF_ACC_TERRAFORM_PATH")
	if binary == "" {
		var err error

		binary, err = exec.LookPath("terraform")
		require.NoError(t, err)
	}

	versionOutput, err := exec.CommandContext(t.Context(), binary, "version", "-json").Output()
	require.NoError(t, err)

	var version struct {
		Version string `json:"terraform_version"` //nolint:tagliatelle // Terraform CLI JSON contract.
	}
	require.NoError(t, json.Unmarshal(versionOutput, &version))

	if strings.HasPrefix(version.Version, "1.12.") || strings.HasPrefix(version.Version, "1.13.") {
		t.Skip("query requires Terraform 1.14+")
	}

	work := t.TempDir()
	pluginDir := t.TempDir()
	build := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(pluginDir, "terraform-provider-braze"), "../..")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)
	server, err := brazeclienttesting.NewBrazeServer()
	require.NoError(t, err)
	server.SetCatalog("products", "Products", []brazeclient.CatalogField{{Name: "id", Type: brazeclient.CatalogFieldTypeString}, {Name: "name", Type: brazeclient.CatalogFieldTypeString}})
	server.SetCatalogItem("products", "one", map[string]json.RawMessage{"name": json.RawMessage(`"One"`)})
	server.SetContentBlock("block-one", "welcome", "Hello", "Welcome", []string{})

	inline := false
	server.SetEmailTemplate("email-one", "Welcome", "Hello", "<p>Hello</p>", "Hello", "Welcome", []string{}, &inline)
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	cliConfig := filepath.Join(work, "terraform.rc")
	require.NoError(t, os.WriteFile(cliConfig, []byte("provider_installation {\n dev_overrides {\n \"cysp/braze\" = "+strconv.Quote(pluginDir)+"\n }\n direct {}\n}\n"), 0o600))
	providerConfig := "terraform {\n required_providers {\n braze = { source = \"cysp/braze\" }\n }\n}\nprovider \"braze\" {\n base_url = " + strconv.Quote(httpServer.URL) + "\n api_key = \"fixture-key\"\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(work, "main.tf"), []byte(providerConfig), 0o600))

	var queryBuilder strings.Builder

	for _, kind := range []string{"catalog", "catalog_item", "content_block", "email_template"} {
		contents, err := os.ReadFile(filepath.Join("../../examples/list-resources", "braze_"+kind, "list-resource.tfquery.hcl"))
		require.NoError(t, err)
		queryBuilder.Write(contents)
	}

	query := queryBuilder.String()

	require.NoError(t, os.WriteFile(filepath.Join(work, "existing.tfquery.hcl"), []byte(query), 0o600))

	run := func(args ...string) []byte {
		t.Helper()
		command := exec.CommandContext(t.Context(), binary, args...)
		command.Dir = work

		command.Env = append(os.Environ(), "TF_CLI_CONFIG_FILE="+cliConfig, "BRAZE_API_KEY=", "TF_LOG=", "TF_IN_AUTOMATION=1")
		output, err := command.CombinedOutput()
		require.NoError(t, err, "%v: %s", args, output)

		return output
	}
	// Validate the shipped provider, resource, and identity-import examples together.
	providerExample, err := os.ReadFile("../../examples/provider/provider.tf")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(work, "main.tf"), providerExample, 0o600))

	for _, kind := range []string{"catalog", "catalog_item", "content_block", "email_template"} {
		for _, filename := range []string{"resource.tf", "import.tf"} {
			contents, err := os.ReadFile(filepath.Join("../../examples/resources", "braze_"+kind, filename))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(work, filename), contents, 0o600))
		}

		run("validate", "-no-color")
	}

	require.NoError(t, os.Remove(filepath.Join(work, "resource.tf")))
	require.NoError(t, os.Remove(filepath.Join(work, "import.tf")))
	require.NoError(t, os.WriteFile(filepath.Join(work, "main.tf"), []byte(providerConfig), 0o600))
	run("query", "-no-color", "-generate-config-out=generated.tf")

	generated, err := os.ReadFile(filepath.Join(work, "generated.tf"))
	require.NoError(t, err)
	require.Equal(t, 4, strings.Count(string(generated), "import {"), "%s", generated)
	run("apply", "-no-color", "-auto-approve")
	run("plan", "-no-color", "-detailed-exitcode")

	var state struct {
		Values struct {
			RootModule struct {
				Resources []struct {
					Address string `json:"address"`
					Type    string `json:"type"`
				} `json:"resources"`
			} `json:"root_module"` //nolint:tagliatelle // Terraform state JSON contract.
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(run("show", "-json"), &state))

	for _, item := range state.Values.RootModule.Resources {
		if item.Type == "braze_content_block" || item.Type == "braze_email_template" {
			output := run("plan", "-no-color", "-replace="+item.Address)
			// Core discards warning-only diagnostics from its replacement replan.
			require.Contains(t, string(output), "will be replaced, as requested")
		}
	}

	destroyPlan := run("plan", "-no-color", "-destroy")
	require.Contains(t, string(destroyPlan), "Content Block will remain in Braze")
	require.Contains(t, string(destroyPlan), "Email Template will remain in Braze")

	// Refresh must reveal changes, including replacement for immutable catalogs.
	server.SetCatalog("products", "Changed remotely", []brazeclient.CatalogField{{Name: "id", Type: brazeclient.CatalogFieldTypeString}, {Name: "name", Type: brazeclient.CatalogFieldTypeString}})
	server.SetCatalogItem("products", "one", map[string]json.RawMessage{"name": json.RawMessage(`"Changed remotely"`)})
	server.SetContentBlock("block-one", "welcome", "Changed remotely", "Welcome", []string{})
	server.SetEmailTemplate("email-one", "Welcome", "Changed remotely", "<p>Hello</p>", "Hello", "Welcome", []string{}, &inline)

	checkActions := func(filename string, expected map[string][]string) {
		t.Helper()
		run("plan", "-no-color", "-out="+filename)
		output := run("show", "-json", filename)

		var plan struct {
			Changes []struct {
				Type   string `json:"type"`
				Change struct {
					Actions []string `json:"actions"`
				} `json:"change"`
			} `json:"resource_changes"` //nolint:tagliatelle // Terraform plan JSON contract.
		}
		require.NoError(t, json.Unmarshal(output, &plan))
		require.Len(t, plan.Changes, 4)

		for _, change := range plan.Changes {
			require.Equal(t, expected[change.Type], change.Change.Actions, change.Type)
		}
	}
	checkActions("drift.tfplan", map[string][]string{
		"braze_catalog": {"delete", "create"}, "braze_catalog_item": {"update"},
		"braze_content_block": {"update"}, "braze_email_template": {"update"},
	})

	_, err = server.Handler().DeleteCatalog(t.Context(), brazeclient.DeleteCatalogParams{CatalogName: "products"})
	require.NoError(t, err)
	server.RemoveContentBlock("block-one")
	server.RemoveEmailTemplate("email-one")
	checkActions("missing.tfplan", map[string][]string{
		"braze_catalog": {"create"}, "braze_catalog_item": {"create"},
		"braze_content_block": {"create"}, "braze_email_template": {"create"},
	})
}
