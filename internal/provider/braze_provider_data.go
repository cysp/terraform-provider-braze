package provider

type brazeProviderData struct {
	contentBlocks  contentBlockClient
	emailTemplates emailTemplateClient
	catalogs       catalogClient
	catalogItems   catalogItemClient
}
