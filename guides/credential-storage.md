# OS keychain credential storage

JFrog CLI uses file-based credential storage by default, with its existing optional
configuration encryption. OS keychain storage is an explicit, per-CLI-home choice.

## Enable or disable

```sh
# Migrate existing credentials and enable keychain storage for future writes.
jf config storage keychain

# Explicitly migrate back to file storage (existing encryption settings apply).
jf config storage file
```

You can enable keychain storage before configuring any servers. The selection
persists in the configuration; subsequent `jf login`, `jf config add/edit/import`,
and automatic token refresh use it without additional flags.

Keychain storage covers passwords, access tokens, both types of refresh token, and
SSH key passphrases. URLs, usernames, certificate/private-key paths, and other
non-secret settings remain in the configuration file. Private-key files themselves
are not moved into the keychain.

## Platform requirements

The implementation uses [zalando/go-keyring](https://github.com/zalando/go-keyring):

- **macOS:** Keychain, accessed through the `security` executable.
- **Windows:** Windows Credential Manager, for the current user.
- **Linux:** an accessible session D-Bus and a Secret Service implementation,
  such as GNOME Keyring or KWallet with Secret Service enabled.

The OS may request permission or unlock confirmation. Keychain storage does not
silently fall back to file storage if the service is missing, locked, or denies
access. This applies to credential reads and token refresh as well as login.
For unattended CI, containers, and SSH sessions without a credential service,
retain file storage and use the existing environment-variable/configuration
options. Do not enable keychain storage in automation unless the OS credential
service is provisioned and usable without interaction.

## Migration, portability, and recovery

Migration writes credentials to the destination before replacing the configuration.
Changed credentials use new keychain entries so a failed configuration write
leaves the old credentials usable. Unchanged credentials keep their existing
entries. Configuration reads, migration, automatic encryption, and updates share
the same lock, preventing stale writes and deletion during secret retrieval.

Superseded entries are removed only after replacement and durability confirmation
(directory synchronization on Unix; write-through replacement on Windows). If
replacement succeeds but durability confirmation fails, both generations are
retained and the error reports that state. A cleanup failure is reported as such,
even though the configuration was saved. A process crash can leave orphaned
entries; there is no transaction spanning the filesystem and OS credential store.

Keychain entries are scoped to the absolute CLI configuration path and current OS
user. Copying the configuration file to another machine, user, or CLI home does
**not** copy credentials. Use `jf config export` / `jf config import` for portable
configuration instead. Export continues to include credentials: the exported token
is sensitive and is not encrypted by keychain storage. Import uses the destination
CLI home's storage selection.

Keep a supported CLI version when using keychain-backed configuration. Older
versions do not understand the credential references. To downgrade, first run
`jf config storage file` with a keychain-aware version.

Migration does not erase historical backups, older versioned config files,
filesystem snapshots, or previously exported tokens. Review these separately if
you are removing existing on-disk credentials. If keychain entries are deleted
externally, recover them or restore/reconfigure credentials; the configuration
file alone cannot recover the secrets.

## Local development

This feature changes both this repository and `jfrog-cli-core`. This branch pins
core to a published commit in `github.com/jbro/jfrog-cli-core/v2`, so a sibling
checkout is not required to build it. Replace that fork dependency with an
upstream release once the core changes are merged and released.

For local core development, temporarily use a sibling checkout:

```sh
go mod edit -replace=github.com/jfrog/jfrog-cli-core/v2=../jfrog-cli-core
go mod tidy
```

Restore the pinned fork dependency before committing CLI changes.

The public configuration read/write APIs acquire the configuration lock. Core
commands that already hold that lock across a read-modify-write operation use the
explicit `*Locked` variants; they must not call the locking variants recursively.

Run the regular suites from each repository with `go test ./...`. Core's regular
config tests include mocked keychains, concurrent reads/migration/encryption,
lock-acquisition failures, pre- and post-replacement failures, retained references,
and malformed chunks. Test dependencies are injected into individual store
instances, not package-global variables.

Core's `.github/workflows/native-keychain.yml` runs the real credential-store
test on Linux, macOS, and Windows for pushes, pull requests, and manual dispatch.
It requires no repository secrets: Linux uses an isolated D-Bus session and
GNOME Keyring, macOS uses a temporary unlocked keychain with search-list/default
restoration, and Windows uses the disposable runner account's Credential Manager.
The workflow has read-only repository permissions and bounded test/job timeouts.

Locally, the same test is opt-in because it accesses a real OS store and may prompt:

```sh
cd ../jfrog-cli-core
JFROG_CLI_TEST_KEYCHAIN=true go test ./utils/config -run '^TestCredentialStoreNative$' -count=1 -timeout 60s
```

Provision an unlocked credential store first. Prefer a disposable D-Bus session
and isolated keyring directories on Linux. Native macOS and Windows validation
must be performed on those platforms; cross-compilation alone does not test OS
credential-store behavior.
