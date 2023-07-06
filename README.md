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

Only show apps that are **known** to be inactive. This checks public DNS records to see if they are pointed at the server, but won't be able to resolve records behind a proxy. (i.e. DNS is managed by Cloudflare)

```shell
serverpilot-tools apps stranded <client_id> <api_key>
```

To resolve DNS records behind CloudFlare, you can provide your CloudFlare api keys via the `cloudflare-credentials` option. This will use the CloudFlare API to resolve DNS records. (Only 1 CloudFlare account is supported at this time.)

```shell
serverpilot-tools apps stranded <client_id> <api_key> --cloudflare-credentials "foo@example.com:1234567890abcdef1234567890abcdef"
```

## Downloads

You can download the latest version from the [releases page](https://github.com/jfortunato/serverpilot-tools/releases/latest)
