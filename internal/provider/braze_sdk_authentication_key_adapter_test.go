//nolint:testpackage
package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedSDKAuthenticationKeyClient(t *testing.T) {
	t.Parallel()

	t.Run("manages key lifecycle and primary promotion", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedSDKAuthenticationKeyClient(newTestBrazeClient(t, func(*brazeclienttesting.Server) {}))

		first, err := client.Create(t.Context(), brazeSDKAuthenticationKeyModel{
			AppID:        types.StringValue("app-1"),
			RSAPublicKey: types.StringValue("first public key"),
			Description:  types.StringValue("First key"),
			Primary:      types.BoolValue(true),
		})
		require.NoError(t, err)
		assert.NotEmpty(t, first.ID.ValueString())
		assert.Equal(t, "app-1", first.AppID.ValueString())
		assert.Equal(t, "first public key", first.RSAPublicKey.ValueString())
		assert.Equal(t, "First key", first.Description.ValueString())
		assert.True(t, first.Primary.ValueBool())

		second, err := client.Create(t.Context(), brazeSDKAuthenticationKeyModel{
			AppID:        types.StringValue("app-1"),
			RSAPublicKey: types.StringValue("second public key"),
			Description:  types.StringValue("Second key"),
			Primary:      types.BoolNull(),
		})
		require.NoError(t, err)
		assert.False(t, second.Primary.ValueBool())

		err = client.Delete(t.Context(), "app-1", first.ID.ValueString())
		require.ErrorIs(t, err, errSDKAuthenticationPrimaryDelete)

		second, err = client.SetPrimary(t.Context(), "app-1", second.ID.ValueString())
		require.NoError(t, err)
		assert.True(t, second.Primary.ValueBool())

		first, err = client.Read(t.Context(), "app-1", first.ID.ValueString())
		require.NoError(t, err)
		assert.False(t, first.Primary.ValueBool())

		require.NoError(t, client.Delete(t.Context(), "app-1", first.ID.ValueString()))

		_, err = client.Read(t.Context(), "app-1", first.ID.ValueString())
		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
	})

	t.Run("maps missing endpoint object to not found", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedSDKAuthenticationKeyClient(newTestBrazeClient(t, func(*brazeclienttesting.Server) {}))

		_, err := client.Read(t.Context(), "app-1", "missing-key")

		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
		require.NoError(t, client.Delete(t.Context(), "app-1", "missing-key"))
	})

	t.Run("preserves created key when verification fails", func(t *testing.T) {
		t.Parallel()

		httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.Method == http.MethodPost && req.URL.Path == "/app_group/sdk_authentication/create":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"key-1"}`))
			case req.Method == http.MethodGet && req.URL.Path == "/app_group/sdk_authentication/keys":
				http.Error(w, "verification failed", http.StatusInternalServerError)
			default:
				http.NotFound(w, req)
			}
		}))
		t.Cleanup(httpServer.Close)

		generatedClient, err := brazeclient.NewClient(
			httpServer.URL,
			NewBrazeAPIKeySecuritySource("test"),
			brazeclient.WithClient(httpServer.Client()),
		)
		require.NoError(t, err)

		client := newGeneratedSDKAuthenticationKeyClient(generatedClient)
		result, err := client.Create(t.Context(), brazeSDKAuthenticationKeyModel{
			AppID:        types.StringValue("app-1"),
			RSAPublicKey: types.StringValue("public-key"),
			Description:  types.StringValue("Key"),
			Primary:      types.BoolValue(true),
		})

		require.Error(t, err)
		assert.Equal(t, "key-1", result.ID.ValueString())
		assert.Equal(t, "app-1", result.AppID.ValueString())
		assert.True(t, result.Primary.ValueBool())
	})

	t.Run("rejects inconsistent primary response", func(t *testing.T) {
		t.Parallel()

		httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodPut || req.URL.Path != "/app_group/sdk_authentication/primary" {
				http.NotFound(w, req)

				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[{"id":"key-1","rsa_public_key":"public-key","description":"Key","is_primary":false}]}`))
		}))
		t.Cleanup(httpServer.Close)

		generatedClient, err := brazeclient.NewClient(
			httpServer.URL,
			NewBrazeAPIKeySecuritySource("test"),
			brazeclient.WithClient(httpServer.Client()),
		)
		require.NoError(t, err)

		client := newGeneratedSDKAuthenticationKeyClient(generatedClient)
		_, err = client.SetPrimary(t.Context(), "app-1", "key-1")

		require.ErrorIs(t, err, errSDKAuthenticationKeyNotPrimary)
	})

	t.Run("does not treat a list endpoint 404 as object absence", func(t *testing.T) {
		t.Parallel()

		httpServer := httptest.NewServer(http.NotFoundHandler())
		t.Cleanup(httpServer.Close)

		generatedClient, err := brazeclient.NewClient(
			httpServer.URL,
			NewBrazeAPIKeySecuritySource("test"),
			brazeclient.WithClient(httpServer.Client()),
		)
		require.NoError(t, err)

		client := newGeneratedSDKAuthenticationKeyClient(generatedClient)

		_, err = client.Read(t.Context(), "app-1", "key-1")
		require.Error(t, err)
		assert.False(t, isBrazeObjectNotFound(err))

		err = client.Delete(t.Context(), "app-1", "key-1")
		require.Error(t, err)
	})

	t.Run("rejects a successful delete response that retains the key", func(t *testing.T) {
		t.Parallel()

		httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodDelete || req.URL.Path != "/app_group/sdk_authentication/delete" {
				http.NotFound(w, req)

				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[{"id":"key-1","rsa_public_key":"public-key","description":"Key","is_primary":false}]}`))
		}))
		t.Cleanup(httpServer.Close)

		generatedClient, err := brazeclient.NewClient(
			httpServer.URL,
			NewBrazeAPIKeySecuritySource("test"),
			brazeclient.WithClient(httpServer.Client()),
		)
		require.NoError(t, err)

		client := newGeneratedSDKAuthenticationKeyClient(generatedClient)
		err = client.Delete(t.Context(), "app-1", "key-1")

		require.ErrorIs(t, err, errSDKAuthenticationKeyStillExists)
	})
}

func TestSDKAuthenticationKeyResponseValidation(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct{ method, body string }{
		"empty created identity":   {http.MethodPost, `{"id":""}`},
		"missing created identity": {http.MethodPost, `{}`},
		"missing promotion target": {http.MethodPut, `{"keys":[]}`},
		"multiple primary keys":    {http.MethodPut, `{"keys":[{"id":"key-1","rsa_public_key":"public-key","description":"Key","is_primary":true},{"id":"key-2","rsa_public_key":"other-key","description":"Other","is_primary":true}]}`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, test.method, req.Method, "Invalid create identity must not trigger a list request")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)
			generated, err := brazeclient.NewClient(server.URL, NewBrazeAPIKeySecuritySource("test"), brazeclient.WithClient(server.Client()))
			require.NoError(t, err)

			client := newGeneratedSDKAuthenticationKeyClient(generated)
			if test.method == http.MethodPost {
				_, err = client.Create(t.Context(), brazeSDKAuthenticationKeyModel{AppID: types.StringValue("app-1"), RSAPublicKey: types.StringValue("public-key"), Description: types.StringValue("Key")})
			} else {
				_, err = client.SetPrimary(t.Context(), "app-1", "key-1")
			}

			require.Error(t, err)
		})
	}
}

func TestGeneratedSDKAuthenticationKeyClientRejectsMalformedCollections(t *testing.T) {
	t.Parallel()

	bodies := map[string]string{
		"empty ID": `{"keys":[
			{"id":"key-1","rsa_public_key":"first","description":"First","is_primary":true},
			{"id":"","rsa_public_key":"other","description":"Other","is_primary":false}
		]}`,
		"duplicate ID": `{"keys":[
			{"id":"key-1","rsa_public_key":"first","description":"First","is_primary":true},
			{"id":"key-1","rsa_public_key":"other","description":"Other","is_primary":false}
		]}`,
		"more than three keys": `{"keys":[
			{"id":"key-1","rsa_public_key":"first","description":"First","is_primary":true},
			{"id":"key-2","rsa_public_key":"second","description":"Second","is_primary":false},
			{"id":"key-3","rsa_public_key":"third","description":"Third","is_primary":false},
			{"id":"key-4","rsa_public_key":"fourth","description":"Fourth","is_primary":false}
		]}`,
		"multiple primary keys": `{"keys":[
			{"id":"key-1","rsa_public_key":"first","description":"First","is_primary":true},
			{"id":"key-2","rsa_public_key":"second","description":"Second","is_primary":true}
		]}`,
		"missing keys": `{}`,
	}
	operations := map[string]func(*testing.T, generatedSDKAuthenticationKeyClient) error{
		"read present": func(t *testing.T, client generatedSDKAuthenticationKeyClient) error {
			t.Helper()

			_, err := client.Read(t.Context(), "app-1", "key-1")

			return err
		},
		"read absent": func(t *testing.T, client generatedSDKAuthenticationKeyClient) error {
			t.Helper()

			_, err := client.Read(t.Context(), "app-1", "missing-key")

			return err
		},
		"create verification": func(t *testing.T, client generatedSDKAuthenticationKeyClient) error {
			t.Helper()

			key, err := client.Create(t.Context(), brazeSDKAuthenticationKeyModel{
				AppID: types.StringValue("app-1"), RSAPublicKey: types.StringValue("first"),
				Description: types.StringValue("First"), Primary: types.BoolValue(true),
			})
			assert.Equal(t, "key-1", key.ID.ValueString(), "Malformed verification must preserve the created identity")

			return err
		},
		"set primary": func(t *testing.T, client generatedSDKAuthenticationKeyClient) error {
			t.Helper()

			_, err := client.SetPrimary(t.Context(), "app-1", "key-1")

			return err
		},
		"delete response": func(t *testing.T, client generatedSDKAuthenticationKeyClient) error {
			t.Helper()

			return client.Delete(t.Context(), "app-1", "missing-key")
		},
		"delete recovery": func(t *testing.T, client generatedSDKAuthenticationKeyClient) error {
			t.Helper()

			return client.Delete(t.Context(), "app-1", "missing-key")
		},
	}

	for name, body := range bodies {
		for operation, call := range operations {
			t.Run(name+"/"+operation, func(t *testing.T) {
				t.Parallel()

				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if operation == "delete recovery" && req.Method == http.MethodDelete {
						http.Error(w, "delete failed", http.StatusInternalServerError)

						return
					}

					w.Header().Set("Content-Type", "application/json")

					if req.Method == http.MethodPost {
						_, _ = w.Write([]byte(`{"id":"key-1"}`))

						return
					}

					_, _ = w.Write([]byte(body))
				}))
				t.Cleanup(server.Close)

				generated, err := brazeclient.NewClient(server.URL, NewBrazeAPIKeySecuritySource("test"), brazeclient.WithClient(server.Client()))
				require.NoError(t, err)

				err = call(t, newGeneratedSDKAuthenticationKeyClient(generated))

				require.Error(t, err, "A malformed collection must not establish object state or successful deletion")
				assert.False(t, isBrazeObjectNotFound(err), "A malformed collection cannot establish absence")
			})
		}
	}
}
