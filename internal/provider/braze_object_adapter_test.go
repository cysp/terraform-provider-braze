//nolint:testpackage
package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestBrazeObjectFetch = errors.New("fetch failed")

func TestCollectBrazeObjectPages(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		query         brazeObjectListQuery
		fetch         func(offset, limit int) ([]string, error)
		expectedItems []string
		expectedCalls []string
		expectedErr   error
	}{
		"zero limit does not fetch": {
			query: brazeObjectListQuery{Limit: 0},
			fetch: func(int, int) ([]string, error) {
				t.Fatal("fetch should not be called")

				return nil, nil
			},
		},
		"stops after short page": {
			query: brazeObjectListQuery{Limit: 100},
			fetch: func(offset, limit int) ([]string, error) {
				return []string{fmt.Sprintf("%d/%d", offset, limit)}, nil
			},
			expectedItems: []string{"0/100"},
			expectedCalls: []string{"0/100"},
		},
		"fetches until requested limit": {
			query: brazeObjectListQuery{Limit: 101},
			fetch: func(offset, limit int) ([]string, error) {
				items := make([]string, limit)
				for i := range items {
					items[i] = strconv.Itoa(offset + i)
				}

				return items, nil
			},
			expectedItems: makeRangeStrings(101),
			expectedCalls: []string{"0/100", "100/100"},
		},
		"returns fetch error": {
			query: brazeObjectListQuery{Limit: 101},
			fetch: func(offset, limit int) ([]string, error) {
				if offset == 100 {
					return nil, errTestBrazeObjectFetch
				}

				return makeRangeStrings(limit), nil
			},
			expectedItems: nil,
			expectedCalls: []string{"0/100", "100/100"},
			expectedErr:   errTestBrazeObjectFetch,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls []string

			actual, err := collectBrazeObjectPages(test.query, func(offset, limit int) ([]string, error) {
				calls = append(calls, fmt.Sprintf("%d/%d", offset, limit))

				return test.fetch(offset, limit)
			})

			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, test.expectedItems, actual)
			assert.Equal(t, test.expectedCalls, calls)
		})
	}
}

func TestGeneratedContentBlockClient(t *testing.T) {
	t.Parallel()

	t.Run("extracts update response content block ID", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			response    brazeclient.UpdateContentBlockRes
			expectedID  string
			expectedErr error
		}{
			"ok": {
				response: &brazeclient.UpdateContentBlockOK{
					ContentBlockID: "ok-content-block",
				},
				expectedID: "ok-content-block",
			},
			"created": {
				response: &brazeclient.UpdateContentBlockCreated{
					ContentBlockID: "created-content-block",
				},
				expectedID: "created-content-block",
			},
			"unexpected": {
				expectedErr: errUnexpectedUpdateContentBlockResponse,
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				actual, err := contentBlockIDFromUpdateContentBlockResponse(test.response)

				if test.expectedErr != nil {
					require.ErrorIs(t, err, test.expectedErr)
				} else {
					require.NoError(t, err)
				}

				assert.Equal(t, test.expectedID, actual)
			})
		}
	})

	t.Run("read maps not found", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedContentBlockClient(newTestBrazeClient(t, func(*brazeclienttesting.Server) {}))

		_, err := client.Read(t.Context(), "missing-content-block")

		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
	})

	t.Run("create returns hydrated model", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedContentBlockClient(newTestBrazeClient(t, func(*brazeclienttesting.Server) {}))

		actual, err := client.Create(t.Context(), brazeContentBlockModel{
			Name:        types.StringValue("Created content block"),
			Description: types.StringValue("created description"),
			Content:     types.StringValue("<p>Created</p>"),
			Tags:        NewTypedListFromStringSlice([]string{"tag2"}),
		})

		require.NoError(t, err)
		assert.NotEmpty(t, actual.ID.ValueString())
		assert.Equal(t, "Created content block", actual.Name.ValueString())
		assert.Equal(t, "created description", actual.Description.ValueString())
		assert.Equal(t, "<p>Created</p>", actual.Content.ValueString())
		assert.Equal(t, []string{"tag2"}, TypedListToStringSlice(actual.Tags))
	})

	t.Run("update returns hydrated model", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedContentBlockClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			server.SetContentBlock("existing-content-block", "Existing content block", "<p>Existing</p>", "description", []string{"tag1"})
		}))

		actual, err := client.Update(t.Context(), brazeContentBlockModel{
			IDIdentityModel: IDIdentityModel{
				ID: types.StringValue("existing-content-block"),
			},
			Name:        types.StringValue("Updated content block"),
			Description: types.StringValue("updated description"),
			Content:     types.StringValue("<p>Updated</p>"),
			Tags:        NewTypedListFromStringSlice([]string{"tag2"}),
		})

		require.NoError(t, err)
		assert.Equal(t, "existing-content-block", actual.ID.ValueString())
		assert.Equal(t, "Updated content block", actual.Name.ValueString())
		assert.Equal(t, "updated description", actual.Description.ValueString())
		assert.Equal(t, "<p>Updated</p>", actual.Content.ValueString())
		assert.Equal(t, []string{"tag2"}, TypedListToStringSlice(actual.Tags))
	})

	t.Run("list hydrates resources", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedContentBlockClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			server.SetContentBlock("existing-content-block", "Existing content block", "<p>Existing</p>", "description", []string{"tag1"})
		}))

		entries, err := client.List(t.Context(), brazeObjectListQuery{
			Limit:           1,
			IncludeResource: true,
		})

		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "existing-content-block", entries[0].ID)
		assert.Equal(t, "Existing content block", entries[0].DisplayName)
		require.NotNil(t, entries[0].Resource)
		assert.Equal(t, "<p>Existing</p>", entries[0].Resource.Content.ValueString())
		assert.NoError(t, entries[0].ResourceErr)
	})
}

func TestGeneratedEmailTemplateClient(t *testing.T) {
	t.Parallel()

	t.Run("read maps not found", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedEmailTemplateClient(newTestBrazeClient(t, func(*brazeclienttesting.Server) {}))

		_, err := client.Read(t.Context(), "missing-email-template")

		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
	})

	t.Run("create returns hydrated model", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedEmailTemplateClient(newTestBrazeClient(t, func(*brazeclienttesting.Server) {}))

		actual, err := client.Create(t.Context(), brazeEmailTemplateModel{
			TemplateName:    types.StringValue("Created email template"),
			Subject:         types.StringValue("Created subject"),
			Body:            types.StringValue("<p>Created</p>"),
			PlaintextBody:   types.StringValue("Created"),
			Preheader:       types.StringValue("Created preview"),
			Tags:            NewTypedListFromStringSlice([]string{"tag2"}),
			ShouldInlineCSS: types.BoolValue(true),
		})

		require.NoError(t, err)
		assert.NotEmpty(t, actual.ID.ValueString())
		assert.Equal(t, "Created email template", actual.TemplateName.ValueString())
		assert.Equal(t, "Created subject", actual.Subject.ValueString())
		assert.Equal(t, "<p>Created</p>", actual.Body.ValueString())
		assert.Equal(t, "Created", actual.PlaintextBody.ValueString())
		assert.Equal(t, "Created preview", actual.Preheader.ValueString())
		assert.Equal(t, []string{"tag2"}, TypedListToStringSlice(actual.Tags))
		assert.True(t, actual.ShouldInlineCSS.ValueBool())
	})

	t.Run("update returns hydrated model", func(t *testing.T) {
		t.Parallel()

		shouldInlineCSS := true
		client := newGeneratedEmailTemplateClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			server.SetEmailTemplate("existing-email-template", "Existing email template", "Subject", "<p>Body</p>", "Body", "Preview", []string{"tag1"}, &shouldInlineCSS)
		}))

		actual, err := client.Update(t.Context(), brazeEmailTemplateModel{
			IDIdentityModel: IDIdentityModel{
				ID: types.StringValue("existing-email-template"),
			},
			TemplateName:    types.StringValue("Updated email template"),
			Subject:         types.StringValue("Updated subject"),
			Body:            types.StringValue("<p>Updated</p>"),
			PlaintextBody:   types.StringValue("Updated"),
			Preheader:       types.StringValue("Updated preview"),
			Tags:            NewTypedListFromStringSlice([]string{"tag2"}),
			ShouldInlineCSS: types.BoolValue(false),
		})

		require.NoError(t, err)
		assert.Equal(t, "existing-email-template", actual.ID.ValueString())
		assert.Equal(t, "Updated email template", actual.TemplateName.ValueString())
		assert.Equal(t, "Updated subject", actual.Subject.ValueString())
		assert.Equal(t, "<p>Updated</p>", actual.Body.ValueString())
		assert.Equal(t, "Updated", actual.PlaintextBody.ValueString())
		assert.Equal(t, "Updated preview", actual.Preheader.ValueString())
		assert.Equal(t, []string{"tag2"}, TypedListToStringSlice(actual.Tags))
		assert.False(t, actual.ShouldInlineCSS.ValueBool())
	})

	t.Run("list hydrates resources", func(t *testing.T) {
		t.Parallel()

		shouldInlineCSS := true
		client := newGeneratedEmailTemplateClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			server.SetEmailTemplate("existing-email-template", "Existing email template", "Subject", "<p>Body</p>", "Body", "Preview", []string{"tag1"}, &shouldInlineCSS)
		}))

		entries, err := client.List(t.Context(), brazeObjectListQuery{
			Limit:           1,
			IncludeResource: true,
		})

		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "existing-email-template", entries[0].ID)
		assert.Equal(t, "Existing email template", entries[0].DisplayName)
		require.NotNil(t, entries[0].Resource)
		assert.Equal(t, "Subject", entries[0].Resource.Subject.ValueString())
		assert.NoError(t, entries[0].ResourceErr)
	})
}

func TestGeneratedCatalogClient(t *testing.T) {
	t.Parallel()

	t.Run("read returns hydrated model", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogClient(newTestBrazeClient(t, withTestCatalog(t)))

		actual, err := client.Read(t.Context(), "centres")

		require.NoError(t, err)
		assert.Equal(t, "centres", actual.Name.ValueString())
		assert.Equal(t, "Centre metadata", actual.Description.ValueString())
		assert.Equal(t, int64(0), actual.NumItems.ValueInt64())
		assert.False(t, actual.UpdatedAt.IsNull())
	})

	t.Run("read missing maps not found", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogClient(newTestBrazeClient(t, func(*brazeclienttesting.Server) {}))

		_, err := client.Read(t.Context(), "missing-catalog")

		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
	})

	t.Run("delete removes catalog", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogClient(newTestBrazeClient(t, withTestCatalog(t)))

		err := client.Delete(t.Context(), "centres")
		require.NoError(t, err)

		_, err = client.Read(t.Context(), "centres")
		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
	})

	t.Run("list returns resource entries", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogClient(newTestBrazeClient(t, withTestCatalog(t)))

		entries, err := client.List(t.Context())

		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "centres", entries[0].ID)
		assert.Equal(t, "centres", entries[0].DisplayName)
		require.NotNil(t, entries[0].Resource)
		assert.Equal(t, "Centre metadata", entries[0].Resource.Description.ValueString())
		assert.NoError(t, entries[0].ResourceErr)
	})
}

func TestGeneratedCatalogItemClient(t *testing.T) {
	t.Parallel()

	t.Run("create uses item-addressed endpoint", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogItemClient(newTestBrazeClient(t, withTestCatalog(t)))

		actual, err := client.Create(t.Context(), brazeCatalogItemModel{
			CatalogName: types.StringValue("centres"),
			ItemID:      types.StringValue("airportwest"),
			ValuesJSON:  jsontypes.NewNormalizedValue(`{"name":"Airport West"}`),
		})

		require.NoError(t, err)
		assert.Equal(t, "centres/airportwest", actual.ID.ValueString())
		assert.JSONEq(t, `{"name":"Airport West"}`, actual.ValuesJSON.ValueString())
	})

	t.Run("read maps not found", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogItemClient(newTestBrazeClient(t, withTestCatalog(t)))

		_, err := client.Read(t.Context(), "centres", "missing-centre")

		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
	})

	t.Run("update replaces item values", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogItemClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			createTestCatalog(t, server)
			server.SetCatalogItem("centres", "airportwest", map[string]json.RawMessage{
				"name": json.RawMessage(`"Airport West"`),
			})
		}))

		actual, err := client.Update(t.Context(), brazeCatalogItemModel{
			CatalogName: types.StringValue("centres"),
			ItemID:      types.StringValue("airportwest"),
			ValuesJSON:  jsontypes.NewNormalizedValue(`{"active":true,"name":"Airport West"}`),
		})

		require.NoError(t, err)
		assert.Equal(t, "centres/airportwest", actual.ID.ValueString())
		assert.JSONEq(t, `{"active":true,"name":"Airport West"}`, actual.ValuesJSON.ValueString())
	})

	t.Run("delete removes item", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogItemClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			createTestCatalog(t, server)
			server.SetCatalogItem("centres", "airportwest", map[string]json.RawMessage{
				"name": json.RawMessage(`"Airport West"`),
			})
		}))

		err := client.Delete(t.Context(), "centres", "airportwest")
		require.NoError(t, err)

		_, err = client.Read(t.Context(), "centres", "airportwest")
		require.Error(t, err)
		assert.True(t, isBrazeObjectNotFound(err))
	})

	t.Run("write item rejects id in request body", func(t *testing.T) {
		t.Parallel()

		_, err := (brazeCatalogItemModel{
			CatalogName: types.StringValue("centres"),
			ItemID:      types.StringValue("airportwest"),
			ValuesJSON:  jsontypes.NewNormalizedValue(`{"id":"airportwest","name":"Airport West"}`),
		}).ToCatalogItemWrite()

		require.ErrorIs(t, err, errCatalogItemValuesJSONIncludesID)
	})

	t.Run("list follows link header pagination", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogItemClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			createTestCatalog(t, server)
			itemClient := newGeneratedCatalogItemClient(serverClient(t, server))

			for i := range 55 {
				id := fmt.Sprintf("centre%02d", i)
				_, err := itemClient.Create(t.Context(), brazeCatalogItemModel{
					CatalogName: types.StringValue("centres"),
					ItemID:      types.StringValue(id),
					ValuesJSON:  jsontypes.NewNormalizedValue(fmt.Sprintf(`{"name":%q}`, id)),
				})
				require.NoError(t, err)
			}
		}))

		entries, err := client.List(t.Context(), "centres", 55)

		require.NoError(t, err)
		require.Len(t, entries, 55)
		assert.Equal(t, "centres/centre00", entries[0].ID)
		assert.Equal(t, "centres/centre54", entries[54].ID)
	})

	t.Run("list stops at requested limit", func(t *testing.T) {
		t.Parallel()

		client := newGeneratedCatalogItemClient(newTestBrazeClient(t, func(server *brazeclienttesting.Server) {
			createTestCatalog(t, server)
			itemClient := newGeneratedCatalogItemClient(serverClient(t, server))

			for i := range 55 {
				id := fmt.Sprintf("centre%02d", i)
				_, err := itemClient.Create(t.Context(), brazeCatalogItemModel{
					CatalogName: types.StringValue("centres"),
					ItemID:      types.StringValue(id),
					ValuesJSON:  jsontypes.NewNormalizedValue(fmt.Sprintf(`{"name":%q}`, id)),
				})
				require.NoError(t, err)
			}
		}))

		entries, err := client.List(t.Context(), "centres", 51)

		require.NoError(t, err)
		require.Len(t, entries, 51)
		assert.Equal(t, "centres/centre50", entries[50].ID)
	})
}

func newTestBrazeClient(t *testing.T, configure func(*brazeclienttesting.Server)) *brazeclient.Client {
	t.Helper()

	server, err := brazeclienttesting.NewBrazeServer()
	require.NoError(t, err)

	configure(server)

	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	client, err := brazeclient.NewClient(
		httpServer.URL,
		NewBrazeAPIKeySecuritySource("test"),
		brazeclient.WithClient(httpServer.Client()),
	)
	require.NoError(t, err)

	return client
}

func serverClient(t *testing.T, server *brazeclienttesting.Server) *brazeclient.Client {
	t.Helper()

	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	client, err := brazeclient.NewClient(
		httpServer.URL,
		NewBrazeAPIKeySecuritySource("test"),
		brazeclient.WithClient(httpServer.Client()),
	)
	require.NoError(t, err)

	return client
}

func createTestCatalog(t *testing.T, server *brazeclienttesting.Server) {
	t.Helper()

	client := newGeneratedCatalogClient(serverClient(t, server))
	_, err := client.Create(t.Context(), brazeCatalogModel{
		Name:        types.StringValue("centres"),
		Description: types.StringValue("Centre metadata"),
		Fields: types.ListValueMust(BrazeCatalogFieldObjectType(), []attr.Value{
			types.ObjectValueMust(BrazeCatalogFieldObjectType().AttrTypes, map[string]attr.Value{
				"name": types.StringValue("id"),
				"type": types.StringValue("string"),
			}),
			types.ObjectValueMust(BrazeCatalogFieldObjectType().AttrTypes, map[string]attr.Value{
				"name": types.StringValue("name"),
				"type": types.StringValue("string"),
			}),
		}),
	})
	require.NoError(t, err)
}

func withTestCatalog(t *testing.T) func(*brazeclienttesting.Server) {
	t.Helper()

	return func(server *brazeclienttesting.Server) {
		createTestCatalog(t, server)
	}
}

func makeRangeStrings(count int) []string {
	items := make([]string, count)
	for i := range items {
		items[i] = strconv.Itoa(i)
	}

	return items
}
