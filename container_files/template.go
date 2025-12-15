package main

import (
	"bytes"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
)

func generateConfigs(templateName string, data any) []configData {
	templatesPath := lookupEnvOrDefault("TEMPLATES_PATH", "/templates/")

	dirs, err := os.ReadDir(templatesPath)
	if err != nil {
		log.Printf("Failed to read '%s': '%v'", templatesPath, err)
		return nil
	}

	dir, ok := find(dirs, func(dir os.DirEntry) bool { return dir.IsDir() && dir.Name() == templateName })
	if !ok {
		log.Printf("Could not find template directory for: '%s'", templateName)
		return nil
	}

	entries, err := os.ReadDir(path.Join(templatesPath, dir.Name()))
	if err != nil {
		log.Printf("Failed to read '%s': '%v'", path.Join(templatesPath, dir.Name()), err)
		return nil
	}

	var configs []configData

	for _, entry := range entries {
		var name = entry.Name()
		var fullPath = path.Join(templatesPath, dir.Name(), name)

		if (!entry.IsDir() && strings.HasSuffix(name, ".yaml.template")) {
			config, ok := applyTemplate(name, fullPath, data)
			if ok {
				configs = append(configs, configData{
					Name: strings.TrimSuffix(name, ".template"),
					Data: config,
				})
			}
		}
	}

	return configs
}

func applyTemplate(name string, path string, data any) (string, bool) {
	template, err := template.New(name).ParseFiles(path)
	if err != nil {
		log.Printf("Failed to parse template file: '%v'", err)
		return "", false
	}

	var buffer bytes.Buffer
	err = template.Execute(&buffer, data)
	if err != nil {
		log.Printf("Failed to execute template: '%v'", err)
		return "", false
	}

	return buffer.String(), true
}

func find[T any](values []T, condition func(T) bool) (T, bool) {
	for _, value := range values {
        if condition(value) {
            return value, true
        }
    }

    return *new(T), false
}
