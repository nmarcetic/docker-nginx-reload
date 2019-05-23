package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	syspids "github.com/nmarcetic/docker-nginx-reload/pids"
	log "github.com/sirupsen/logrus"
)

const (
	defVaultCRLAPIURL = "http://localhost/v1/pki/crl/pem"
	defVaultAPIToken  = "123"
	defCRLFileName    = "crl.pem"
	defCMDKill        = ".*nginx.*"
	defAPIPort        = "8000"
	defAPIEndpoint    = "/reload"

	envVaultCRLAPIURL = "VAULT_API_URL"
	envVaultAPIToken  = "VAULT_API_TOKEN"
	envCRLFileName    = "CRL_FILE_PATH"
	envCMDKill        = "CMD_TO_EXEC"
	envAPIPort        = "API_PORT"
	envAPIEndpoint    = "API_ENDPOINT"
)

type config struct {
	Port        string
	Endpoint    string
	VaultURL    string
	VaultToken  string
	CRLFileName string
	CMDKill     string
}

type content struct {
	Payload []byte
	File    string
}

const (
	errFetchCRL = "Faild to fetch CRL from Vault"
	errExecCMD  = "Faild to execute cmd"
	errWriteCRL = "Faild to update CRL"
	errParseCRL = "Faild to parse CRL"
)

func main() {
	c := loadConfig()
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// Only log the warning severity or above.
	log.SetLevel(log.ErrorLevel)
	http.HandleFunc(c.Endpoint, reloadHanlder)
	http.ListenAndServe(fmt.Sprintf(":%s", c.Port), nil)
}

func reloadHanlder(w http.ResponseWriter, r *http.Request) {
	c := loadConfig()
	req, err := http.NewRequest(http.MethodGet, c.VaultURL, nil)
	req.Header.Add("X-Vault-Token", c.VaultToken)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf("%s:%s", errFetchCRL, err.Error()))
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(errFetchCRL))
		return
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(fmt.Sprintf("%s:%s", errParseCRL, err.Error()))
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(errParseCRL))
		return
	}
	// Write content to CRL file
	crl := content{
		File:    c.CRLFileName,
		Payload: bodyBytes,
	}
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

func (c *content) Write() error {
	// check if file exists
	var _, err = os.Stat(c.File)
	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(c.File)
		if err != nil {
			return err
		}
		defer file.Close()
	}
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
	return config{
		Port:        getEnv(envAPIPort, defAPIPort),
		Endpoint:    getEnv(envAPIEndpoint, defAPIEndpoint),
		VaultURL:    getEnv(envVaultCRLAPIURL, defVaultCRLAPIURL),
		VaultToken:  getEnv(envVaultAPIToken, defVaultAPIToken),
		CRLFileName: getEnv(envCRLFileName, defCRLFileName),
		CMDKill:     getEnv(envCMDKill, defCMDKill),
	}
}
