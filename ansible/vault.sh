#!/usr/bin/env bash
set -euo pipefail

VAULTS=$(find inventory -name "vault.yml" | sort)

pick_command() {
  echo "Целевое действие:" >&2
  select cmd in edit encrypt decrypt; do
    [ -n "$cmd" ] && echo "$cmd" && return
    echo "Неверный выбор" >&2
  done
}

pick_vault() {
  echo "vault-файл:" >&2
  select vault in $VAULTS; do
    [ -n "$vault" ] && echo "$vault" && return
    echo "Такого номера не существует" >&2
  done
}

CMD=${1:-$(pick_command)}
case $CMD in
edit | encrypt | decrypt) ;;
*)
  echo "Неизвестная команда: $CMD. Используй edit, encrypt или decrypt." >&2
  exit 1
  ;;
esac

echo
VAULT=$(pick_vault)
echo
ansible-vault "$CMD" "$VAULT"
