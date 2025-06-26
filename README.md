# AlertCLI - CLI для управления Alertmanager

[![Go Report Card](https://goreportcard.com/badge/github.com/romashqua/alertcli)](https://goreportcard.com/report/github.com/romashqua/alertcli)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/romashqua/alertcli/blob/main/LICENSE)

## Установка

### Сборка из исходников

1. Убедитесь, что у вас установлен Go (версии 1.16 или выше)
2. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/romashqua/alertcli.git
   cd alertcli
   ```
3. Соберите проект:
   ```bash
   go build -o alertctl
    ```

## Использование

### Основные команды
```
    # Просмотр алертов
    alertctl alerts list [flags]

    # Управление silence'ами
    alertctl silences list
    alertctl silences create [flags]
    alertctl silences delete <id>

    # Генерация автодополнения для shell
    alertctl completion bash|zsh|fish
```

### Фильтрация алертов

```
    # Все алерты (включая silenced/inhibited)
    alertctl alerts list -A

    # Только silenced алерты
    alertctl alerts list -s

    # Только inhibited алерты
    alertctl alerts list -i

    # Только активные алерты (по умолчанию)
    alertctl alerts list

    # Фильтр по severity
    alertctl alerts list -l critical

    # Фильтр по instance
    alertctl alerts list -n "node-exporter:9100"

    # Комбинированные фильтры
    alertctl alerts list -A -l warning -n "node-exporter:9100"
```

### Управление silence'ами

```
    # Создание silence
    alertctl silences create \
    --comment "Planned maintenance" \
    --duration 2h \
    --alertname "HighLoad" \
    --instance "node-exporter:9100"

    # Просмотр всех silence'ов
    alertctl silences list

    # Удаление silence
    alertctl silences delete abc123def456
```

### Примеры вывода
```bash
    ALERT       SEVERITY  STATE     SINCE     INSTANCE              SUMMARY
    HighLoad    warning   active    15m       node-exporter:9100   Instance under high load
    NodeDown    critical  silenced  1h        server-01            Node is down
    ####
    ID          STATUS    START               END                 CREATOR    COMMENT
    abc123      active    2023-05-20 14:00    2023-05-20 16:00    admin      Maintenance window
```

### Более подробно есть в --help