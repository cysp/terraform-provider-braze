package testing

import brazeclient "github.com/cysp/terraform-provider-braze/internal/braze-client-go"

func (s *Server) SetSDKAuthenticationKey(appID string, key brazeclient.SDKAuthenticationKey) {
	s.handler.mu.Lock()
	defer s.handler.mu.Unlock()

	keys := s.handler.sdkAuthenticationKeys[appID]
	if keys == nil {
		keys = make(map[string]brazeclient.SDKAuthenticationKey)
		s.handler.sdkAuthenticationKeys[appID] = keys
	}

	if key.IsPrimary {
		clearPrimarySDKAuthenticationKey(keys)
	}

	keys[key.ID] = key
}

// ResetSDKAuthenticationKeys removes an app's test data without simulating a
// Braze API operation. It exists only to clean up Terraform lifecycle tests
// whose final primary key is intentionally not deletable through the API.
func (s *Server) ResetSDKAuthenticationKeys(appID string) {
	s.handler.mu.Lock()
	defer s.handler.mu.Unlock()

	delete(s.handler.sdkAuthenticationKeys, appID)
}
