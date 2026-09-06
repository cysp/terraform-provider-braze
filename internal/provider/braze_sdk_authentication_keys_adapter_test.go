//nolint:testpackage
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedSDKAuthenticationKeysClientList(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	client := newGeneratedSDKAuthenticationKeysClient(newSDKAuthenticationKeysAdapterTestClient(t, func(
		w http.ResponseWriter,
		req *http.Request,
	) {
		requests.Add(1)
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "/app_group/sdk_authentication/keys", req.URL.Path)
		assert.Equal(t, "app-1", req.URL.Query().Get("app_id"))
		writeSDKAuthenticationKeysAdapterTestJSON(t, w, `{
			"keys": [
				{"id":"key-1","rsa_public_key":"shared-key","description":"same","is_primary":false},
				{"id":"key-2","rsa_public_key":"shared-key","description":"same","is_primary":false},
				{"id":"key-3","rsa_public_key":"third-key","description":"","is_primary":false}
			]
		}`)
	}))

	keys, err := client.List(t.Context(), "app-1")

	require.NoError(t, err)
	assert.Equal(t, []sdkAuthenticationCollectionKey{
		{ID: "key-1", RSAPublicKey: "shared-key", Description: "same", Primary: false},
		{ID: "key-2", RSAPublicKey: "shared-key", Description: "same", Primary: false},
		{ID: "key-3", RSAPublicKey: "third-key", Description: "", Primary: false},
	}, keys)
	assert.Equal(t, int32(1), requests.Load())
}

func TestGeneratedSDKAuthenticationKeysClientListEmptyCollection(t *testing.T) {
	t.Parallel()

	client := newGeneratedSDKAuthenticationKeysClient(newSDKAuthenticationKeysAdapterTestClient(t, func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		writeSDKAuthenticationKeysAdapterTestJSON(t, w, `{"keys":[]}`)
	}))

	keys, err := client.List(t.Context(), "app-1")

	require.NoError(t, err)
	assert.NotNil(t, keys)
	assert.Empty(t, keys)
}

func TestGeneratedSDKAuthenticationKeysClientCreateReturnsGeneratedIDOnly(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	client := newGeneratedSDKAuthenticationKeysClient(newSDKAuthenticationKeysAdapterTestClient(t, func(
		w http.ResponseWriter,
		req *http.Request,
	) {
		requests.Add(1)
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/app_group/sdk_authentication/create", req.URL.Path)

		var request brazeclient.CreateSDKAuthenticationKeyRequest

		decodeErr := json.NewDecoder(req.Body).Decode(&request)
		if !assert.NoError(t, decodeErr) {
			http.Error(w, "invalid request", http.StatusBadRequest)

			return
		}

		assert.Equal(t, "app-1", request.GetAppID())
		assert.Equal(t, "public-key", request.GetRsaPublicKeyStr())
		assert.Empty(t, request.GetDescription())
		assert.Equal(t, brazeclient.NewOptBool(true), request.GetMakePrimary())

		writeSDKAuthenticationKeysAdapterTestJSON(t, w, `{"id":"generated-key"}`)
	}))

	keyID, err := client.Create(t.Context(), "app-1", sdkAuthenticationCollectionKey{
		ID:           "must-not-be-sent",
		RSAPublicKey: "public-key",
		Description:  "",
		Primary:      true,
	})

	require.NoError(t, err)
	assert.Equal(t, "generated-key", keyID)
	assert.Equal(t, int32(1), requests.Load(), "Create must not perform its lifecycle verification List")
}

func TestGeneratedSDKAuthenticationKeysClientReturnsFullMutationCollections(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method string
		path   string
		call   func(context.Context, generatedSDKAuthenticationKeysClient) ([]sdkAuthenticationCollectionKey, error)
		body   string
		want   []sdkAuthenticationCollectionKey
	}{
		"set primary": {
			method: http.MethodPut,
			path:   "/app_group/sdk_authentication/primary",
			call: func(ctx context.Context, client generatedSDKAuthenticationKeysClient) ([]sdkAuthenticationCollectionKey, error) {
				return client.SetPrimary(ctx, "app-1", "key-2")
			},
			// The target remains non-primary. The adapter must preserve this
			// structurally valid contradiction for executor verification.
			body: `{"keys":[
				{"id":"key-1","rsa_public_key":"first","description":"First","is_primary":true},
				{"id":"key-2","rsa_public_key":"second","description":"Second","is_primary":false},
				{"id":"key-3","rsa_public_key":"third","description":"Third","is_primary":false}
			]}`,
			want: []sdkAuthenticationCollectionKey{
				{ID: "key-1", RSAPublicKey: "first", Description: "First", Primary: true},
				{ID: "key-2", RSAPublicKey: "second", Description: "Second", Primary: false},
				{ID: "key-3", RSAPublicKey: "third", Description: "Third", Primary: false},
			},
		},
		"delete": {
			method: http.MethodDelete,
			path:   "/app_group/sdk_authentication/delete",
			call: func(ctx context.Context, client generatedSDKAuthenticationKeysClient) ([]sdkAuthenticationCollectionKey, error) {
				return client.Delete(ctx, "app-1", "key-2")
			},
			// The target remains present. The adapter must return the entire
			// response so the executor can stop on the contradiction.
			body: `{"keys":[
				{"id":"key-1","rsa_public_key":"first","description":"First","is_primary":true},
				{"id":"key-2","rsa_public_key":"second","description":"Second","is_primary":false}
			]}`,
			want: []sdkAuthenticationCollectionKey{
				{ID: "key-1", RSAPublicKey: "first", Description: "First", Primary: true},
				{ID: "key-2", RSAPublicKey: "second", Description: "Second", Primary: false},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int32

			client := newGeneratedSDKAuthenticationKeysClient(newSDKAuthenticationKeysAdapterTestClient(t, func(
				w http.ResponseWriter,
				req *http.Request,
			) {
				requests.Add(1)
				assert.Equal(t, test.method, req.Method)
				assert.Equal(t, test.path, req.URL.Path)

				var request map[string]string

				decodeErr := json.NewDecoder(req.Body).Decode(&request)
				if !assert.NoError(t, decodeErr) {
					http.Error(w, "invalid request", http.StatusBadRequest)

					return
				}

				assert.Equal(t, "app-1", request["app_id"])
				assert.Equal(t, "key-2", request["key_id"])
				writeSDKAuthenticationKeysAdapterTestJSON(t, w, test.body)
			}))

			keys, err := test.call(t.Context(), client)

			require.NoError(t, err)
			assert.Equal(t, test.want, keys)
			assert.Equal(t, int32(1), requests.Load())
		})
	}
}

func TestGeneratedSDKAuthenticationKeysClientRejectsMalformedCollections(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"empty ID": `{"keys":[
			{"id":"","rsa_public_key":"first","description":"First","is_primary":true}
		]}`,
		"duplicate ID": `{"keys":[
			{"id":"key-1","rsa_public_key":"first","description":"First","is_primary":true},
			{"id":"key-1","rsa_public_key":"second","description":"Second","is_primary":false}
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
		"invalid JSON": `{"keys":`,
		"missing keys": `{}`,
	}

	for name, body := range tests {
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			t.Run(name+"/"+method, func(t *testing.T) {
				t.Parallel()

				client := newGeneratedSDKAuthenticationKeysClient(newSDKAuthenticationKeysAdapterTestClient(t, func(
					w http.ResponseWriter,
					req *http.Request,
				) {
					assert.Equal(t, method, req.Method)
					writeSDKAuthenticationKeysAdapterTestJSON(t, w, body)
				}))

				var (
					keys []sdkAuthenticationCollectionKey
					err  error
				)

				switch method {
				case http.MethodGet:
					keys, err = client.List(t.Context(), "app-1")
				case http.MethodPut:
					keys, err = client.SetPrimary(t.Context(), "app-1", "key-1")
				case http.MethodDelete:
					keys, err = client.Delete(t.Context(), "app-1", "missing-key")
				}

				require.Error(t, err)
				assert.Nil(t, keys)
			})
		}
	}
}

func TestGeneratedSDKAuthenticationKeysClientDoesNotRetryHTTPFailures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method string
		path   string
		call   func(*testing.T, context.Context, generatedSDKAuthenticationKeysClient) error
	}{
		"list": {
			method: http.MethodGet,
			path:   "/app_group/sdk_authentication/keys",
			call: func(t *testing.T, ctx context.Context, client generatedSDKAuthenticationKeysClient) error {
				t.Helper()

				keys, err := client.List(ctx, "app-1")
				assert.Nil(t, keys)

				return err
			},
		},
		"create": {
			method: http.MethodPost,
			path:   "/app_group/sdk_authentication/create",
			call: func(t *testing.T, ctx context.Context, client generatedSDKAuthenticationKeysClient) error {
				t.Helper()

				keyID, err := client.Create(ctx, "app-1", sdkAuthenticationCollectionKey{})
				assert.Empty(t, keyID)

				return err
			},
		},
		"set primary": {
			method: http.MethodPut,
			path:   "/app_group/sdk_authentication/primary",
			call: func(t *testing.T, ctx context.Context, client generatedSDKAuthenticationKeysClient) error {
				t.Helper()

				keys, err := client.SetPrimary(ctx, "app-1", "key-1")
				assert.Nil(t, keys)

				return err
			},
		},
		"delete": {
			method: http.MethodDelete,
			path:   "/app_group/sdk_authentication/delete",
			call: func(t *testing.T, ctx context.Context, client generatedSDKAuthenticationKeysClient) error {
				t.Helper()

				keys, err := client.Delete(ctx, "app-1", "key-1")
				assert.Nil(t, keys)

				return err
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int32

			client := newGeneratedSDKAuthenticationKeysClient(newSDKAuthenticationKeysAdapterTestClient(t, func(
				w http.ResponseWriter,
				req *http.Request,
			) {
				requests.Add(1)
				assert.Equal(t, test.method, req.Method)
				assert.Equal(t, test.path, req.URL.Path)
				http.Error(w, "failed", http.StatusInternalServerError)
			}))

			err := test.call(t, t.Context(), client)

			require.Error(t, err)
			assert.Equal(t, int32(1), requests.Load())
		})
	}
}

func TestSDKAuthenticationCollectionFromResponseRejectsAbsentResponses(t *testing.T) {
	t.Parallel()

	for name, response := range map[string]*brazeclient.SDKAuthenticationKeysResponseStatusCode{
		"nil response": nil,
		"missing keys": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keys, err := sdkAuthenticationCollectionFromResponse(response)

			require.Error(t, err)
			assert.Nil(t, keys)
		})
	}
}

func TestGeneratedSDKAuthenticationKeysClientRejectsEmptyCreateIDWithoutListing(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	client := newGeneratedSDKAuthenticationKeysClient(newSDKAuthenticationKeysAdapterTestClient(t, func(
		w http.ResponseWriter,
		req *http.Request,
	) {
		requests.Add(1)
		assert.Equal(t, http.MethodPost, req.Method)
		writeSDKAuthenticationKeysAdapterTestJSON(t, w, `{"id":""}`)
	}))

	keyID, err := client.Create(t.Context(), "app-1", sdkAuthenticationCollectionKey{})

	require.ErrorIs(t, err, errSDKAuthenticationKeysEmptyID)
	assert.Empty(t, keyID)
	assert.Equal(t, int32(1), requests.Load())
}

func newSDKAuthenticationKeysAdapterTestClient(
	t *testing.T,
	handler http.HandlerFunc,
) *brazeclient.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := brazeclient.NewClient(
		server.URL,
		NewBrazeAPIKeySecuritySource("test"),
		brazeclient.WithClient(server.Client()),
	)
	require.NoError(t, err)

	return client
}

func writeSDKAuthenticationKeysAdapterTestJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(body))
	assert.NoError(t, err)
}
