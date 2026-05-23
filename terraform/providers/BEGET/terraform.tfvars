# terraform/providers/beget/terraform.tfvars

servers = {
  "sts-ru" = {
    name          = "sts-ru"
    description   = "Status page server"
    hostname      = "sts-ru"
    region        = "ru1"
    cpu           = 1
    ram_mb        = 1024
    disk_mb       = 10240
    cpu_class     = "normal_cpu"
    software_slug = "debian-12"
  }
}
