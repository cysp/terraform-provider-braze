//nolint:testpackage
package provider

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"

	brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"
	brazeclienttesting "github.com/cysp/terraform-provider-braze/internal/braze-client-go/testing"
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

func makeRangeStrings(count int) []string {
	items := make([]string, count)
	for i := range items {
		items[i] = strconv.Itoa(i)
	}

	return items
}
