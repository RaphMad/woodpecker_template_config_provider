#!/bin/sh

docker build --pull -f Dockerfile -t registry.traefik.lan/raphmad/woodpecker_template_config_provider container_files/
