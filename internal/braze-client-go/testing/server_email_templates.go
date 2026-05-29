package testing

func (s *Server) SetEmailTemplate(templateID, templateName, subject, body, plaintextBody, preheader string, tags []string, shouldInlineCSS *bool) {
	s.handler.setEmailTemplate(templateID, templateName, subject, body, plaintextBody, preheader, tags, shouldInlineCSS)
}
