package tool

import (
	"context"
	"fmt"
	toolkit "github.com/yurika0211/luckyagent-tools"
	"github.com/yurika0211/luckyagent/internal/credentials"
	"strings"
)

type credentialProvider struct{ homeDir string }

func (p credentialProvider) WithCredential(ctx context.Context, ref string, use func([]byte) error) error {
	if strings.TrimSpace(p.homeDir) == "" {
		return fmt.Errorf("credential home directory is unavailable")
	}
	store, err := credentials.NewStore(p.homeDir)
	if err != nil {
		return err
	}
	defer store.Close()
	return store.WithCredential(ctx, ref, use)
}
func NewEphemeralCredentialToolService(home string) *toolkit.EphemeralCredentialToolService {
	return toolkit.NewEphemeralCredentialToolService(credentialProvider{home})
}
func NewEphemeralCredentialTerminalToolService(home string) *toolkit.EphemeralCredentialTerminalToolService {
	return toolkit.NewEphemeralCredentialTerminalToolService(credentialProvider{home})
}
