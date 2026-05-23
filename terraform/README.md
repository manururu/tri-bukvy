# Terraform

Управление ВМ-ками у облачных провайдеров через GitHub Actions.

## Содержание

- [Структура](#структура)
- [Операции](#операции)
- [Добавить сервер](#добавить-сервер)
- [Изменить ресурсы сервера](#изменить-ресурсы-сервера)
- [Добавить провайдера](#добавить-провайдера)
- [Секреты GitHub](#секреты-github)

## Структура

```
providers/PROVIDER1/      — конфиги провайдера PROVIDER1
providers/PROVIDER2/
...
```

## Операции

Все операции запускаются вручную: **Actions > [TERRAFORM] Deploy > Run workflow**

| Action    | Что делает                                               |
| --------- | -------------------------------------------------------- |
| `plan`    | Показывает изменения без применения                      |
| `apply`   | Применяет изменения                                      |
| `destroy` | Удаляет все серверы провайдера (требует ввода `DESTROY`) |

## Добавить сервер

Открыть `providers/PROVIDER/terraform.tfvars`, добавить в `servers`:

```hcl
"my-server" = {
  name     = "my-server"
  hostname = "my-server"
  region   = "ru1"
}
```

Запустить `apply`.

## Изменить ресурсы сервера

1. В `providers/PROVIDER/terraform.tfvars` изменить нужные поля (`cpu`, `ram_mbi`, `disk_mb`)
2. Запустить `apply`.

## Добавить провайдера

1. Создать `providers/NEWPROVIDER/` (см образец `providers/BEGET/`)
2. Добавить секрет `TERRAFORM_NEWPROVIDER_TOKEN` в GitHub → Settings → Secrets
3. Создать workspace `newprovider` в [Terraform Cloud](https://app.terraform.io) (execution mode: **Local**)
4. Добавить `NEWPROVIDER` в список `provider` в `.github/workflows/terraform.yml`

## Секреты GitHub

| Секрет                         | Описание                        |
| ------------------------------ | ------------------------------- |
| `TERRAFORM_CLOUD_TOKEN`        | Токен Terraform Cloud           |
| `TERRAFORM_<PROVIDER>_TOKEN`   | API-токен провайдера            |
| `COMMON_DEPLOY_SSH_PUBLIC_KEY` | Публичный SSH-ключ для серверов |
