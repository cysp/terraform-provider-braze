package provider

func detailFromError(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
