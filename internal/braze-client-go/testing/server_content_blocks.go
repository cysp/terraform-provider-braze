package testing

func (s *Server) SetContentBlock(contentBlockID, name, content, description string, tags []string) {
	s.handler.setContentBlock(contentBlockID, name, content, description, tags)
}

// RemoveContentBlock simulates deletion through the Braze dashboard, not an API.
func (s *Server) RemoveContentBlock(contentBlockID string) {
	s.handler.mu.Lock()
	defer s.handler.mu.Unlock()

	delete(s.handler.contentBlocks, contentBlockID)
}
