terraform {
  required_providers {
    couchbase-capella-aidp = {
      source = "mminichino/couchbase-capella-aidp"
    }
  }
}

provider "couchbase-capella-aidp" {
  authentication_token       = var.couchbasecapella_auth_token
  global_api_request_timeout = 300
}

variable "couchbasecapella_auth_token" {
  type      = string
  sensitive = true
}
