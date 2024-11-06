package bitwarden

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/quexten/goldwarden/cli/agent/bitwarden/crypto"
	"github.com/quexten/goldwarden/cli/agent/bitwarden/models"
	"github.com/quexten/goldwarden/cli/agent/config"
	"github.com/quexten/goldwarden/cli/agent/vault"
	"github.com/quexten/goldwarden/cli/logging"
)

var log = logging.GetLogger("Goldwarden", "Bitwarden API")

func Sync(ctx context.Context, config *config.Config) (models.SyncData, error) {
	var sync models.SyncData
	if err := authenticatedHTTPGet(ctx, config.ConfigFile.ApiUrl+"/sync", &sync); err != nil {
		return models.SyncData{}, fmt.Errorf("could not sync: %v", err)
	}

	return sync, nil
}

func DoFullSync(ctx context.Context, vault *vault.Vault, config *config.Config, userSymmetricKey *crypto.SymmetricEncryptionKey, allowCache bool) error {
	log.Info("Performing full sync...")
	sync, err := Sync(ctx, config)
	if err != nil {
		log.Error("Could not sync: %v", err)
		return err
	} else {
		log.Info("Sync successful, initializing keyring and vault...")
	}

	var orgKeys map[string]string = make(map[string]string)
	log.Info("Reading  %d org keys...", len(sync.Profile.Organizations))
	for _, org := range sync.Profile.Organizations {
		orgId := org.Id.String()
		orgKeys[orgId] = org.Key
	}
	if userSymmetricKey != nil {
		log.Info("Initializing keyring from user symmetric key...")
		err = crypto.InitKeyringFromUserSymmetricKey(vault.Keyring, *userSymmetricKey, sync.Profile.PrivateKey, orgKeys)
		if err != nil {
			return err
		}
	}

	log.Info("Clearing vault...")
	vault.Clear()
	vault.SetLastSynced(time.Now().Unix())
	log.Info("Adding %d ciphers to vault...", len(sync.Ciphers))
	for _, cipher := range sync.Ciphers {
		switch cipher.Type {
		case models.CipherLogin:
			vault.AddOrUpdateLogin(cipher)
		case models.CipherNote:
			vault.AddOrUpdateSecureNote(cipher)
		case models.CipherSSHKey:
			vault.AddOrUpdateSSHKey(cipher)
		}
	}

	return nil
}

func WriteVault(data models.SyncData, path string) error {
	dataJson, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// write to disk
	os.Remove(path)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.Write(dataJson)
	if err != nil {
		return err
	}
	return nil
}

func ReadVault(path string) (models.SyncData, error) {
	file, err := os.Open(path)
	if err != nil {
		return models.SyncData{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	data := models.SyncData{}
	err = decoder.Decode(&data)
	if err != nil {
		return models.SyncData{}, err
	}
	return data, nil
}
