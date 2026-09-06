package testing

func (s *Server) SetEmailTemplate(templateID, templateName, subject, body, plaintextBody, preheader string, tags []string, shouldInlineCSS *bool) {
	s.handler.setEmailTemplate(templateID, templateName, subject, body, plaintextBody, preheader, tags, shouldInlineCSS)
}

// RemoveEmailTemplate simulates deletion through the Braze dashboard, not an API.
func (s *Server) RemoveEmailTemplate(templateID string) {
	s.handler.mu.Lock()
	defer s.handler.mu.Unlock()

	delete(s.handler.emailTemplates, templateID)
}
