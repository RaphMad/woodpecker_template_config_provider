# Overview

Provides a [ConfigurationExtension](https://woodpecker-ci.org/docs/usage/extensions/configuration-extension) for [Woodpecker](https://github.com/woodpecker-ci/woodpecker/), based on [go text templates](https://pkg.go.dev/text/template).

# Configuration

The following env vars are can be set to modify behaviour the config extension.

| Name                    | Default                         | Description                                                                                                                                                                       |
|-------------------------|---------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| CONFIG_SERVICE_PORT     | 8000                            | Listening port of the service.                                                                                                                                                    |
| WEBHOOK_PUBLIC_KEY_PATH | /run/secrets/webhook_public_key | Path to a file containing your woodpecker webhook public key. It can be obtained via <any repo> -> Settings -> Extensions.                                                        |
| TEMPLATES_PATH          | /templates/                     | Path to your template files                                                                                                                                                       |
| EXTRA_CA_CERT_FILE      | _empty_                         | Path to an extra CA cert, useful when your forge uses a local CA. Will be passed as `CABundle` to [CloneOptions](CA`https://pkg.go.dev/github.com/go-git/go-git/v5#CloneOptions). |

The following env vars should be set on your server:

* [WOODPECKER_CONFIG_SERVICE_ENDPOINT](https://woodpecker-ci.org/docs/usage/extensions/configuration-extension#global-configuration) (can also be set individually per repo)
* [WOODPECKER_EXTENSIONS_ALLOWED_HOSTS](https://woodpecker-ci.org/docs/administration/configuration/server#extensions_allowed_hosts) (host part of your template config service)

# Usage

* In your project, create a file called `.woodpecker/woodpecker-template.yaml`, e.g.:

´´´
template: <templatename>
data:
  lines:
    - one
    - two
    - three
´´´

* Inside your template config provider, provide one or multiple `*.yaml.template` files under `/templates/<templatename>/`, e.g.:

```
when:
  - event:
    - push

steps:
  - name: output lines
    image: alpine
    commands:{{range .lines }}
      - echo {{ . }}{{ end }}
```

* `data:` is optional, templates can also be useful when multiple repos use the same pipeline without any differences
* A `<templatename>` dir corresponds to a `Pipeline` and can contain multiple `*.yaml.template` files, which correspond to individual `Workflows` (see Woodpecker documentation for more info)

# Example compose snippet

```
  woodpecker_template_config_provider:
    image: raphmad/woodpecker_template_config_provider
    restart: unless-stopped
    security_opt:
      - no-new-privileges=true
    cap_drop:
      - ALL
    read_only: true
    user: 1000:1000
    secrets:
      - webhook_public_key
    volumes:
      - /mnt/docker_volumes/woodpecker/template_config_provider/templates/:/templates/:ro
```
