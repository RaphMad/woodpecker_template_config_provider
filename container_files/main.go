package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/yaronf/httpsign"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.yaml.in/yaml/v4"

	_ "golang.org/x/crypto/x509roots/fallback"
)

var (
	webbHookPublicKey ed25519.PublicKey
	extraCABundle     []byte
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
	// Disable date/time log prefix.
	log.SetFlags(0);

	port := lookupEnvOrDefault("CONFIG_SERVICE_PORT", "8000")

	if len(os.Args) == 2 && os.Args[1] == "ping" {
		err := pinger(port);
		if err != nil {
			log.Fatalf("Error sending ping: '%v'", err)
		}

		return
	}

	log.Println("woodpecker_template_config_provider started")

	publicKeyFile := lookupEnvOrDefault("WEBHOOK_PUBLIC_KEY_PATH", "/run/secrets/webhook_public_key")
	extraCAFile, ok := os.LookupEnv("EXTRA_CA_CERT_FILE")
	if ok {
		initializeExtraCABundle(extraCAFile)
	}

	loadPublicKey(publicKeyFile)

	http.HandleFunc("/templateconfig", handleHttpRequest)
	http.HandleFunc("/healthz", handleHeartbeat)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func lookupEnvOrDefault(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	return value
}

func initializeExtraCABundle(fileName string) {
	var err error
	extraCABundle, err = os.ReadFile(fileName)
		if err != nil {
		log.Fatalf("Could not read extra CA cert file %s: '%v'", fileName, err)
	}
}

func loadPublicKey(publicKeyFile string) {
	pubKeyRaw, err := os.ReadFile(publicKeyFile)
	if err != nil {
		log.Fatalf("Failed to read %s: '%v'", publicKeyFile, err)
	}

	pemBlock, rest := pem.Decode(pubKeyRaw)
	if len(rest) != 0 {
		log.Fatal("PEM block contained rest.",)
	}

	b, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse public key file: '%v'", err)
	}

	var ok bool
	webbHookPublicKey, ok = b.(ed25519.PublicKey)
	if !ok {
		log.Fatal("Failed to parse public key file")
	}
}

func handleHttpRequest(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		log.Printf("Invalid signature")
		http.Error(writer, "Expected POST", http.StatusMethodNotAllowed)
		return
	}

	if !verifySignature(request) {
		http.Error(writer, "Could not verify signature", http.StatusBadRequest)
		return
	}

	req, ok := parseRequest(request)
	if !ok {
		http.Error(writer, "Could not parse request", http.StatusBadRequest)
		return
	}

	fileBytes, ok := getTemplateFileFromForge(req, extraCABundle)
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

func verifySignature(r *http.Request) bool {
	pubKeyID := "woodpecker-ci-extensions"

	verifier, err := httpsign.NewEd25519Verifier(
		webbHookPublicKey,
		httpsign.NewVerifyConfig(),
		httpsign.Headers("@request-target", "content-digest"),
	)

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
