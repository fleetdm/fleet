[Back to top](../README.md)
# Configuring the Server

## ENV VAR

Used for local development and cloud deployments alike.

### Local development

You can either source env var changes into your current terminal or set them during a specific build
to override like so.

```
MYSQL_ADDRESS=somethingelse.com ./build/fleet serve
```

### Helm deployment

helm values.yaml -> templates -> server container ...

## UI / API

TODO

## Gitops

TODO