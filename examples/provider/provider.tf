terraform {
  required_version = ">= 1.12"
  required_providers {
    braze = { source = "cysp/braze" }
  }
}

provider "braze" {
  # Choose the REST endpoint for your Braze instance.
  base_url = "https://rest.iad-01.braze.com"
  # Set BRAZE_API_KEY in your environment.
}
