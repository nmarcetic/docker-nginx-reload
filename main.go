package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"

	syspids "github.com/nmarcetic/docker-nginx-reload/pids"
	log "github.com/sirupsen/logrus"
)

const (
	defVaultCRLAPIURL         = "http://localhost"
	defVaultCAHasIntermadiate = "false"
	defVaultSecretRootName    = "pki"
	defVaultSecretIntName     = "pki_int"
	defCRLFileName            = "crl.pem"
	defCMDKill                = ".*nginx: master.*"
	defAPIPort                = "8000"
	defAPIEndpoint            = "/reload"

	envVaultCRLAPIURL         = "VAULT_API_URL"
	envVaultCAHasIntermediate = "VAULT_CA_INTERMEDIATE"
	envVaultSecretRootName    = "VAULT_SECRET_ROOT"
	envVaultSecretIntName     = "VAULT_SECRET_INTERMEDIATE"
	envCRLFileName            = "CRL_FILE_PATH"
	envCMDKill                = "CMD_TO_EXEC"
	envAPIPort                = "API_PORT"
	envAPIEndpoint            = "API_ENDPOINT"
)

type config struct {
	Port                  string
	Endpoint              string
	VaultURL              string
	VaultCAIntermediate   bool
	VaultSecretRootName   string
	VaultSecretIntName    string
	VaultIntermediateName string
	CRLFileName           string
	CMDKill               string
}

type content struct {
	Payload []byte
	File    string
}

const (
	errFetchCRL         = "Faild to fetch CRL from Vault"
	errFaildToCreateCRL = "Faild to create CRL file"
	errFetchIntCRL      = "Faild to fetch Intermediate CRL from Vault"
	errExecCMD          = "Faild to execute cmd"
	errWriteCRL         = "Faild to update CRL"
	errParseCRL         = "Faild to parse CRL"
)

func main() {
	c := loadConfig()
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// Only log the warning severity or above.
	log.SetLevel(log.ErrorLevel)
	// Check if CRL file exists on system
	var _, err = os.Stat(c.CRLFileName)
	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(c.CRLFileName)
		if err != nil {
			log.Error(fmt.Sprintf("%s:%s", errFaildToCreateCRL, err.Error()))
		}
		defer file.Close()
	}
	http.HandleFunc(c.Endpoint, reloadHanlder)
	http.ListenAndServe(fmt.Sprintf(":%s", c.Port), nil)
}

func reloadHanlder(w http.ResponseWriter, r *http.Request) {
	c := loadConfig()
	// Write content to CRL file
	crl := content{
		File: c.CRLFileName,
	}
	u := fmt.Sprintf("%s/v1/%s/crl/pem", c.VaultURL, c.VaultSecretRootName)
	fmt.Printf(u)
	bodyBytes, err := crl.GetCRL(u, c)
	if err != nil {
		log.Error(fmt.Sprintf("%s:%s", errFetchCRL, err.Error()))
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(errFetchCRL))
		return
	}
	if c.VaultCAIntermediate {
		u = fmt.Sprintf("%s/v1/%s/crl/pem", c.VaultURL, c.VaultIntermediateName)
		bodyIntBytes, err := crl.GetCRL(u, c)
		if err != nil {
			log.Error(fmt.Sprintf("%s:%s", errFetchIntCRL, err.Error()))
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(errFetchCRL))
			return
		}
		bodyBytes = append(bodyBytes, "\n"...)
		bodyBytes = append(bodyBytes, bodyIntBytes...)
	}
	crl.Payload = bodyBytes

	if err := crl.Write(); err != nil {
		log.Error(fmt.Sprintf("%s:%s", errWriteCRL, err.Error()))
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(errWriteCRL))
		return
	}
	pids, err := syspids.FindPIDs(regexp.MustCompile(c.CMDKill))

	if err != nil {
		log.Error(fmt.Sprintf("%s:%s", "Faild to find PID's", err.Error()))
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(errWriteCRL))
		return
	}
	// Log if there is no pids found
	if len(pids) > 0 {
		// Send kill signal to all PIDS
		if err := syspids.SendSignal(pids); err != nil {
			log.Error(fmt.Sprintf("%s:%s", "Error sending Kill signal to PID", err.Error()))
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(errExecCMD))
			return
		}
		// Successfully sent
		log.SetLevel(log.InfoLevel)
		log.Info("Successfully sent signal to all PIDS")
	} else {
		log.Error("Didn't find active PIDS")
	}

	w.Write([]byte(http.StatusText(200)))

}

func (c *content) GetCRL(url string, conf config) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf("%s:%s", errFetchCRL, err.Error()))
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(fmt.Sprintf("%s:%s", errParseCRL, err.Error()))
		return nil, err
	}
	return bodyBytes, nil
}

func (c *content) Write() error {
	if err := ioutil.WriteFile(c.File, c.Payload, 0644); err != nil {
		return err
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func loadConfig() config {
	vaultCAIntermediate, err := strconv.ParseBool(getEnv(envVaultCAHasIntermediate, defVaultCAHasIntermadiate))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envVaultCAHasIntermediate)
	}
	return config{
		Port:                  getEnv(envAPIPort, defAPIPort),
		Endpoint:              getEnv(envAPIEndpoint, defAPIEndpoint),
		VaultURL:              getEnv(envVaultCRLAPIURL, defVaultCRLAPIURL),
		VaultCAIntermediate:   vaultCAIntermediate,
		VaultSecretRootName:   getEnv(envVaultSecretRootName, defVaultSecretRootName),
		VaultIntermediateName: getEnv(envVaultSecretIntName, defVaultSecretIntName),
		CRLFileName:           getEnv(envCRLFileName, defCRLFileName),
		CMDKill:               getEnv(envCMDKill, defCMDKill),
	}
}
