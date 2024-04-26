terraform {
  required_providers {
    heroku = {
      source  = "heroku/heroku"
      version = "~> 2.0"
    }
  }
}

provider "heroku" {
  # email   = var.heroku_email
  # api_key = var.heroku_api_key
  email   = "luke@fleetdm.com"
  api_key = "ac68ed03-ac91-4253-84e7-7911f51c7245"
}

resource "heroku_app" "speedy" {
  name       = "speedy-fleetie"
  region     = "us"
  buildpacks = ["https://github.com/heroku/heroku-buildpack-go"]
}

# MySQL-compatible add-on
resource "heroku_addon" "database" {
  app  = heroku_app.speedy.name
  plan = "cleardb:punch8" # Change to the specific plan you need
}

# Redis add-on
resource "heroku_addon" "redis" {
  app  = heroku_app.speedy.name
  plan = "heroku-redis:mini" # Change to the specific plan you need
}
