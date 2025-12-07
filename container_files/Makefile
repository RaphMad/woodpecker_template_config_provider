all: /out/woodpecker_template_config_provider

/out/woodpecker_template_config_provider: main.go
	CGO_ENABLED=0 GOAMD64=v4 go build -o /out/woodpecker_template_config_provider

clean:
	rm -rf /out/woodpecker_template_config_provider

.PHONY: all clean
