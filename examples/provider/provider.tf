provider "signoz" {
  # example configuration here
  access_token = "your-access-token"
  endpoint     = "http://localhost:3301"

  # Set this to true to disable refresh checks and avoid "inconsistent result" warnings
  # This is useful when you trust the apply operation but want to avoid JSON formatting inconsistencies
  disable_refresh = true
}
