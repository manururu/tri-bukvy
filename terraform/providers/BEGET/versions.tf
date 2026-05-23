# terraform/providers/beget/versions.tf

terraform {
  cloud {
    organization = "tri-bukvy"
    workspaces {
      name = "beget"
    }
  }

  required_providers {
    beget = {
      source  = "tf.beget.com/beget/beget"
      version = "~> 0.0.68"
    }
  }

  required_version = ">= 1.6"
}
