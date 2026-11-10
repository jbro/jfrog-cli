package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	coreconfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func runStorageCLI(t *testing.T, args ...string) error {
	t.Helper()
	app := cli.NewApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	app.Commands = []cli.Command{{Name: "config", Subcommands: GetCommands()}}
	return app.Run(append([]string{"jf", "config", "storage"}, args...))
}

func TestStorageCommandSelectsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv(coreutils.HomeDir, home)
	t.Setenv(coreutils.EncryptionKey, "")
	require.NoError(t, runStorageCLI(t, "file"))
	data, err := os.ReadFile(filepath.Join(home, coreutils.JfrogConfigFile+".v"+strconv.Itoa(coreutils.GetCliConfigVersion())))
	require.NoError(t, err)
	var conf coreconfig.Config
	require.NoError(t, json.Unmarshal(data, &conf))
	require.Equal(t, coreconfig.CredentialStoreFile, conf.CredentialStore)
}

func TestStorageCommandRejectsInvalidArguments(t *testing.T) {
	t.Setenv(coreutils.HomeDir, t.TempDir())
	for _, args := range [][]string{nil, {"keychain", "file"}, {"unknown"}} {
		require.Error(t, runStorageCLI(t, args...))
	}
}
