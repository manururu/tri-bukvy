# terraform/providers/BEGET/main.tf

provider "beget" {
  token = var.provider_token
}

resource "beget_ssh_key" "deploy" {
  name       = "deploy"
  public_key = var.ssh_public_key
}

data "beget_software" "os" {
  for_each = var.servers
  slug     = each.value.software_slug
}

resource "beget_compute_instance" "servers" {
  for_each = var.servers

  name        = each.value.name
  description = each.value.description
  hostname    = each.value.hostname
  region      = each.value.region

  configuration = {
    cpu       = each.value.cpu
    ram_mb    = each.value.ram_mb
    disk_mb   = each.value.disk_mb
    cpu_class = each.value.cpu_class
  }

  image = {
    software = {
      id = data.beget_software.os[each.key].id
    }
  }

  access = {
    ssh_keys = [beget_ssh_key.deploy.id]
  }
}

output "server_ips" {
  description = "Map of server key => public IPv4 address"
  value = {
    for k, v in beget_compute_instance.servers : k => v.ip_address
  }
}
