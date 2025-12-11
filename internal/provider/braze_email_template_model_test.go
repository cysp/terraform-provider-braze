package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestBrazeEmailTemplateModelToCreateRequest(t *testing.T) {
	t.Parallel()

	model := brazeEmailTemplateModel{
		TemplateName:    types.StringValue("Test Template"),
		Description:     types.StringValue("Test Description"),
		Subject:         types.StringValue("Test Subject"),
		Preheader:       types.StringValue("Test Preheader"),
		Body:            types.StringValue("<h1>Test Body</h1>"),
		PlaintextBody:   types.StringValue("Test Body"),
		ShouldInlineCSS: types.BoolValue(true),
		Tags:            NewTypedListFromStringSlice([]string{"tag1", "tag2"}),
	}

	req := model.ToCreateEmailTemplateRequest()

	assert.Equal(t, "Test Template", req.TemplateName)
	assert.True(t, req.Description.IsSet())
	assert.Equal(t, "Test Description", req.Description.Value)
	assert.Equal(t, "Test Subject", req.Subject)
	assert.True(t, req.Preheader.IsSet())
	assert.Equal(t, "Test Preheader", req.Preheader.Value)
	assert.Equal(t, "<h1>Test Body</h1>", req.Body)
	assert.True(t, req.PlaintextBody.IsSet())
	assert.Equal(t, "Test Body", req.PlaintextBody.Value)
	assert.True(t, req.ShouldInlineCSS.IsSet())
	assert.True(t, req.ShouldInlineCSS.Value)
	assert.True(t, req.Tags.IsSet())
	assert.Equal(t, []string{"tag1", "tag2"}, req.Tags.Value)
}

func TestBrazeEmailTemplateModelToCreateRequestMinimal(t *testing.T) {
	t.Parallel()

	model := brazeEmailTemplateModel{
		TemplateName: types.StringValue("Test Template"),
		Subject:      types.StringValue("Test Subject"),
		Body:         types.StringValue("<h1>Test Body</h1>"),
		Tags:         NewTypedListNull[types.String](),
	}

	req := model.ToCreateEmailTemplateRequest()

	assert.Equal(t, "Test Template", req.TemplateName)
	assert.False(t, req.Description.IsSet())
	assert.Equal(t, "Test Subject", req.Subject)
	assert.False(t, req.Preheader.IsSet())
	assert.Equal(t, "<h1>Test Body</h1>", req.Body)
	assert.False(t, req.PlaintextBody.IsSet())
	assert.False(t, req.ShouldInlineCSS.IsSet())
	assert.True(t, req.Tags.IsNull())
}

func TestBrazeEmailTemplateModelToUpdateRequest(t *testing.T) {
	t.Parallel()

	model := brazeEmailTemplateModel{
		IDIdentityModel: IDIdentityModel{
			ID: types.StringValue("template-id-123"),
		},
		TemplateName:    types.StringValue("Updated Template"),
		Description:     types.StringValue("Updated Description"),
		Subject:         types.StringValue("Updated Subject"),
		Preheader:       types.StringValue("Updated Preheader"),
		Body:            types.StringValue("<h1>Updated Body</h1>"),
		PlaintextBody:   types.StringValue("Updated Body"),
		ShouldInlineCSS: types.BoolValue(false),
		Tags:            NewTypedListFromStringSlice([]string{"tag3"}),
	}

	req := model.ToUpdateEmailTemplateRequest()

	assert.Equal(t, "template-id-123", req.EmailTemplateID)
	assert.True(t, req.TemplateName.IsSet())
	assert.Equal(t, "Updated Template", req.TemplateName.Value)
	assert.True(t, req.Description.IsSet())
	assert.Equal(t, "Updated Description", req.Description.Value)
	assert.True(t, req.Subject.IsSet())
	assert.Equal(t, "Updated Subject", req.Subject.Value)
	assert.True(t, req.Preheader.IsSet())
	assert.Equal(t, "Updated Preheader", req.Preheader.Value)
	assert.True(t, req.Body.IsSet())
	assert.Equal(t, "<h1>Updated Body</h1>", req.Body.Value)
	assert.True(t, req.PlaintextBody.IsSet())
	assert.Equal(t, "Updated Body", req.PlaintextBody.Value)
	assert.True(t, req.ShouldInlineCSS.IsSet())
	assert.False(t, req.ShouldInlineCSS.Value)
	assert.True(t, req.Tags.IsSet())
	assert.Equal(t, []string{"tag3"}, req.Tags.Value)
}

func TestBrazeEmailTemplateModelToUpdateRequestWithNulls(t *testing.T) {
	t.Parallel()

	model := brazeEmailTemplateModel{
		IDIdentityModel: IDIdentityModel{
			ID: types.StringValue("template-id-123"),
		},
		TemplateName:    types.StringValue("Updated Template"),
		Description:     types.StringNull(),
		Subject:         types.StringValue("Updated Subject"),
		Preheader:       types.StringNull(),
		Body:            types.StringValue("<h1>Updated Body</h1>"),
		PlaintextBody:   types.StringNull(),
		ShouldInlineCSS: types.BoolNull(),
		Tags:            NewTypedListNull[types.String](),
	}

	req := model.ToUpdateEmailTemplateRequest()

	assert.Equal(t, "template-id-123", req.EmailTemplateID)
	assert.True(t, req.TemplateName.IsSet())
	assert.Equal(t, "Updated Template", req.TemplateName.Value)
	assert.True(t, req.Description.IsSet())
	assert.True(t, req.Description.IsNull())
	assert.True(t, req.Subject.IsSet())
	assert.Equal(t, "Updated Subject", req.Subject.Value)
	assert.True(t, req.Preheader.IsSet())
	assert.True(t, req.Preheader.IsNull())
	assert.True(t, req.Body.IsSet())
	assert.Equal(t, "<h1>Updated Body</h1>", req.Body.Value)
	assert.True(t, req.PlaintextBody.IsSet())
	assert.True(t, req.PlaintextBody.IsNull())
	assert.False(t, req.ShouldInlineCSS.IsSet())
	assert.True(t, req.Tags.IsNull())
}
