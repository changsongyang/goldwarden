package actions

import (
	"context"
	"strings"

	"github.com/quexten/goldwarden/cli/agent/bitwarden"
	"github.com/quexten/goldwarden/cli/agent/config"
	"github.com/quexten/goldwarden/cli/agent/sockets"
	"github.com/quexten/goldwarden/cli/agent/ssh"
	"github.com/quexten/goldwarden/cli/agent/systemauth"
	"github.com/quexten/goldwarden/cli/agent/vault"
	"github.com/quexten/goldwarden/cli/ipc/messages"
)

func handleAddSSH(msg messages.IPCMessage, cfg *config.Config, vault *vault.Vault, callingContext *sockets.CallingContext) (response messages.IPCMessage, err error) {
	req := messages.ParsePayload(msg).(messages.CreateSSHKeyRequest)

	cipher, publicKey, err := ssh.NewSSHKeyCipher(req.Name, vault.Keyring)
	if err != nil {
		response, err = messages.IPCMessageFromPayload(messages.ActionResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	_, err = messages.IPCMessageFromPayload(messages.ActionResponse{
		Success: true,
	})
	if err != nil {
		response, err = messages.IPCMessageFromPayload(messages.ActionResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	token, err := cfg.GetToken()
	if err != nil {
		actionsLog.Warn(err.Error())
	}
	ctx := context.WithValue(context.TODO(), bitwarden.AuthToken{}, token.AccessToken)
	postedCipher, err := bitwarden.PostCipher(ctx, cipher, cfg)
	if err == nil {
		vault.AddOrUpdateSecureNote(postedCipher)
	} else {
		actionsLog.Warn("Error posting ssh key cipher: " + err.Error())
	}

	response, err = messages.IPCMessageFromPayload(messages.CreateSSHKeyResponse{
		Digest: strings.ReplaceAll(publicKey, "\n", "") + " " + req.Name,
	})

	return
}

func handleListSSH(msg messages.IPCMessage, cfg *config.Config, vault *vault.Vault, callingContext *sockets.CallingContext) (response messages.IPCMessage, err error) {
	keys := vault.GetSSHKeys()
	keyStrings := make([]string, 0)
	for _, key := range keys {
		keyStrings = append(keyStrings, strings.ReplaceAll(key.PublicKey+" "+key.Name, "\n", ""))
	}

	response, err = messages.IPCMessageFromPayload(messages.GetSSHKeysResponse{
		Keys: keyStrings,
	})
	return
}

func handleImportSSH(msg messages.IPCMessage, cfg *config.Config, vault *vault.Vault, callingContext *sockets.CallingContext) (response messages.IPCMessage, err error) {
	req := messages.ParsePayload(msg).(messages.ImportSSHKeyRequest)

	cipher, _, err := ssh.SSHKeyCipherFromKey(req.Name, req.Key, vault.Keyring)
	if err != nil {
		response, err = messages.IPCMessageFromPayload(messages.ActionResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	_, err = messages.IPCMessageFromPayload(messages.ActionResponse{
		Success: true,
	})
	if err != nil {
		response, err = messages.IPCMessageFromPayload(messages.ActionResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	token, err := cfg.GetToken()
	if err != nil {
		actionsLog.Warn(err.Error())
	}
	ctx := context.WithValue(context.TODO(), bitwarden.AuthToken{}, token.AccessToken)
	postedCipher, err := bitwarden.PostCipher(ctx, cipher, cfg)
	if err == nil {
		vault.AddOrUpdateSecureNote(postedCipher)
	} else {
		actionsLog.Warn("Error posting ssh key cipher: " + err.Error())
	}

	response, err = messages.IPCMessageFromPayload(messages.ImportSSHKeyResponse{
		Success: true,
	})
	return
}

func init() {
	AgentActionsRegistry.Register(messages.MessageTypeForEmptyPayload(messages.CreateSSHKeyRequest{}), ensureEverything(systemauth.SSHKey, handleAddSSH))
	AgentActionsRegistry.Register(messages.MessageTypeForEmptyPayload(messages.GetSSHKeysRequest{}), ensureIsNotLocked(ensureIsLoggedIn(handleListSSH)))
	AgentActionsRegistry.Register(messages.MessageTypeForEmptyPayload(messages.ImportSSHKeyRequest{}), ensureEverything(systemauth.SSHKey, handleImportSSH))
}
