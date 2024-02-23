# serverpilot-tools

This is a collection of tools for managing apps & servers on [serverpilot.io](https://serverpilot.io/).

## Installation

### Go

```shell
go install github.com/jfortunato/serverpilot-tools@latest
```

### Manual

Download & extract the latest binary for your platform from the [releases page](https://github.com/jfortunato/serverpilot-tools/releases/latest)

## Examples

### List all servers

```shell
serverpilot-tools servers list <client_id> <api_key>
```

### List all apps

```shell
serverpilot-tools apps list <client_id> <api_key>
```

### List apps created between two dates

```shell
serverpilot-tools apps list <client_id> <api_key> --created-after 2022-06-01 --created-before 2023-04-25
```

### List apps using outdated PHP versions

```shell
serverpilot-tools apps list <client_id> <api_key> --max-runtime php8.0
```

### Find apps that are inactive (DNS not pointing to the server)

Only show apps that are **known** to be inactive. This checks public DNS records to see if they are pointed at the server. If the DNS records are behind CloudFlare, it will automatically detect that and you will need to provide your CloudFlare API credentials.

```shell
serverpilot-tools apps inactive <client_id> <api_key>
```

## Downloads

You can download the latest version from the [releases page](https://github.com/jfortunato/serverpilot-tools/releases/latest)
