package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/yaronf/httpsign"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.yaml.in/yaml/v4"
)

type woodpeckerRequest struct {
	Repo     *model.Repo     `json:"repo"`
	Pipeline *model.Pipeline `json:"pipeline"`
	Netrc    *model.Netrc    `json:"netrc"`
}

type woodpeckerResponse struct {
	Configs []configData `json:"configs"`
}

type configData struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type templateData struct {
	Template string `yaml:"template"`
	Data     any    `yaml:"data"`
}

// Based on https://github.com/woodpecker-ci/example-config-service/blob/main/main.go
func main() {
	log.Println("woodpecker_template_config_provider started")

	pubKeyRaw, err := os.ReadFile("/run/secrets/woodpecker_public_key")
	if err != nil {
		log.Fatalf("Failed to read /run/secrets/woodpecker_public_key: '%v'", err)
	}

	pemBlock, _ := pem.Decode(pubKeyRaw)

	b, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse public key file: '%v'", err)
	}

	pubKey, ok := b.(ed25519.PublicKey)
	if !ok {
		log.Fatal("Failed to parse public key file")
	}

	http.HandleFunc("/templateconfig", func(w http.ResponseWriter, r *http.Request) { handleHttpRequest(w, r, pubKey) })

	err = http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatalf("Error on listen: '%v'", err)
	}
}

func handleHttpRequest(writer http.ResponseWriter, request *http.Request, pubKey ed25519.PublicKey) {
	if request.Method != http.MethodPost {
		log.Printf("Invalid signature")
		http.Error(writer, "Expected POST", http.StatusMethodNotAllowed)
		return
	}

	if !verifySignature(pubKey, request) {
		http.Error(writer, "Could not verify signature", http.StatusBadRequest)
		return
	}

	req, ok := parseRequest(request)
	if !ok {
		http.Error(writer, "Could not parse request", http.StatusBadRequest)
		return
	}

	fileBytes, ok := getTemplateFileFromForge(req)
	if !ok {
		// Provided request did not contain template data, use config as-is.
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	templateData, ok := parseTemplateData(fileBytes)
	if !ok {
		http.Error(writer, "Could not parse template data", http.StatusBadRequest)
		return
	}

	generatedConfigs := generateConfigs(templateData.Template, templateData.Data)

	if generatedConfigs != nil {
		writer.WriteHeader(http.StatusOK)
		err := json.NewEncoder(writer).Encode(woodpeckerResponse{
			Configs: generatedConfigs,
		})

		if err != nil {
			log.Printf("Could not encode generated configs as json: '%v'", err)
			http.Error(writer, "Could not encode generated configs as json", http.StatusBadRequest)
			return
		}
	} else {
		// No configs could be generated from template data, try to use it as-is (still most likely an error).
		writer.WriteHeader(http.StatusNoContent)
	}
}

func verifySignature(pubKey ed25519.PublicKey, r *http.Request) bool {
	pubKeyID := "woodpecker-ci-extensions"

	verifier, err := httpsign.NewEd25519Verifier(pubKey,
		httpsign.NewVerifyConfig(),
		httpsign.Headers("@request-target", "content-digest"))
	if err != nil {
		log.Printf("Missing required headers: '%v'", err)
		return false
	}

	err = httpsign.VerifyRequest(pubKeyID, *verifier, r)
	if err != nil {
		log.Printf("Invalid signature: '%v'", err)
		return false
	}

	return true
}

func parseRequest(request *http.Request) (woodpeckerRequest, bool) {
	var req woodpeckerRequest

	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Printf("Error reading body: '%v'", err)
		return req, false
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error parsing json: '%v'", err)
		return req, false
	}

	return req, true
}

func parseTemplateData(bytes []byte) (templateData, bool) {
	var data templateData

	err := yaml.Unmarshal(bytes, &data)
	if err != nil {
		log.Printf("Error parsing temlpate data: '%v'", err)
		return data, false
	}

	return data, true
}
