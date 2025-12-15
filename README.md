# Overview

Provides a [ConfigurationExtension](https://woodpecker-ci.org/docs/usage/extensions/configuration-extension) for [Woodpecker](https://github.com/woodpecker-ci/woodpecker/), based on [go text templates](https://pkg.go.dev/text/template).

# Configuration

The following env vars are can be set to modify behaviour the config extension:

| Name                    | Default                         | Description                                                                                                                                                                       |
|-------------------------|---------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| CONFIG_SERVICE_PORT     | 8000                            | Listening port of the service.                                                                                                                                                    |
| WEBHOOK_PUBLIC_KEY_PATH | /run/secrets/webhook_public_key | Path to a file containing your woodpecker webhook public key. It can be obtained via _any repo_ -> Settings -> Extensions.                                                        |
| TEMPLATES_PATH          | /templates/                     | Path to your template directories.                                                                                                                                                |
| EXTRA_CA_CERT_FILE      | _empty_                         | Path to an extra CA cert, useful when your forge uses a local CA. Will be passed as `CABundle` to [CloneOptions](https://pkg.go.dev/github.com/go-git/go-git/v5#CloneOptions). |

The following env vars should be set on your woodpecker server:

* [WOODPECKER_CONFIG_SERVICE_ENDPOINT](https://woodpecker-ci.org/docs/usage/extensions/configuration-extension#global-configuration) (can also be set individually per repo)
  * e.g.: `WOODPECKER_CONFIG_SERVICE_ENDPOINT: http://woodpecker_template_config_provider:8000/templateconfig`
* [WOODPECKER_EXTENSIONS_ALLOWED_HOSTS](https://woodpecker-ci.org/docs/administration/configuration/server#extensions_allowed_hosts) (host part of your template config service)
  * e.g.: `WOODPECKER_EXTENSIONS_ALLOWED_HOSTS: woodpecker_template_config_provider`

# Usage

* In your project, create a file called `.woodpecker/woodpecker-template.yaml`, e.g.:

```
template: <templatename>
data:
  lines:
    - one
    - two
    - three
```

* Inside your template config provider, provide one or multiple `*.yaml.template` files under `/templates/<templatename>/`, e.g.:

```
when:
  - event:
    - push

steps:
  - name: output
    image: alpine
    commands:{{range .lines }}
      - echo {{ . }}{{ end }}
```

* `data:` is optional, templates can also be useful when multiple repos share the exact same pipeline without any text templating
* A `<templatename>` dir corresponds to a `Pipeline` and can contain multiple `*.yaml.template` files, which correspond to individual `Workflows` (see Woodpecker documentation for more info)

# Debugging tips

* Watch output of config provider container
* In the Woodpecker UI, can inspect the generated configs under pipeline - Configs

# Example compose snippet

```
  woodpecker_template_config_provider:
    image: raphmad/woodpecker_template_config_provider
    container_name: woodpecker_template_config_provider
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

# Notes

* An image was published on Dockerhub under `raphmad/woodpecker_template_config_provider`, but feel free to modify `build.sh` and create or you own
* Build and output are docker-based, but you can extract the `/woodpecker_template_config_provider` binary from the published image and run it standalone
