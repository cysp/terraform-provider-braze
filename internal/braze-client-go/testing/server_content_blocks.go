package testing

func (s *Server) SetContentBlock(contentBlockID, name, content, description string, tags []string) {
	s.handler.setContentBlock(contentBlockID, name, content, description, tags)
}

func (s *Server) SetOrphanedContentBlock(contentBlockID, name string, tags []string) {
	s.handler.setOrphanedContentBlock(contentBlockID, name, tags)
}
