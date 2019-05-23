****docker-nginx-reload****

Sidecar container for K8s nginx CRL reload.
It exposes HTTP API endpoint which triggers CRL fetching from Vault and updating CRL file, then sending reload signal to nginx in order to re-load CRL file.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                                | Default               |
|-------------------------------------|------------------------------------------------------------|-----------------------|
| VAULT_API_URL                         | [Vault instance API CRL read endpoint](https://www.vaultproject.io/api/secret/pki/index.html#read-crl)                                           | "http://locahost/v1/pki/crl/pem" 
| VAULT_API_TOKEN                       | Vault instance access token  | "123"                 |
| CRL_FILE_PATH                         | Path to CRL pem file                                          | "crl.pem"                  |
| CMD_TO_EXEC                           | Command which will be executed to send signal and reload nginx                | "kill -HUP nginx"             |
| API_PORT                              |   API listening port                                  | "8000"              |
| API_ENDPOINT                       | API Endpoint                                        |    "/reload"                   |
