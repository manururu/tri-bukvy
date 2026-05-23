# terraform/providers/BEGET/variables.tf

variable "provider_token" {
  type        = string
  sensitive   = true
  description = "API token for the cloud provider"
}

variable "ssh_public_key" {
  type        = string
  sensitive   = true
  description = "Public SSH key to attach to all servers"
}

variable "servers" {
  description = "Map of servers to provision. Key is used as the Terraform resource key."
  type = map(object({
    name          = string
    description   = optional(string, "")
    hostname      = string
    region        = string
    cpu           = optional(number, 1)
    ram_mb        = optional(number, 1024)
    disk_mb       = optional(number, 10240)
    cpu_class     = optional(string, "normal_cpu")
    software_slug = optional(string, "debian-12")
  }))
}
