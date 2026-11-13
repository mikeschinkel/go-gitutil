package gitutil

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// SetConfig sets key to value in the repository's local config. It does not
// need Open.
func (r *Repo) SetConfig(ctx context.Context, key, value string) error {
	return r.setConfig(ctx, nil, key, value)
}

// GetConfig returns key's value from the repository's local config, or an
// error wrapping ErrConfigNotSet when the key is not set there. It does not
// need Open.
func (r *Repo) GetConfig(ctx context.Context, key string) (string, error) {
	return r.getConfig(ctx, nil, key, "--local")
}

func (r *Repo) setConfig(ctx context.Context, call *RepoArgs, key, value string) (err error) {
	_, err = r.run(ctx, call, runArgs{
		Command: []string{"config", "--local", "--", key, value},
	})
	if err != nil {
		err = NewErr(ErrSetConfig, "key", key, err)
	}
	return err
}

// getConfig returns key's value as git resolves it, from every config file
// git reads unless scope (such as --local) narrows it.
func (r *Repo) getConfig(ctx context.Context, call *RepoArgs, key string, scope ...string) (value string, err error) {
	var cmd []string

	cmd = append([]string{"config"}, scope...)
	cmd = append(cmd, "--get", "--", key)
	value, err = r.run(ctx, call, runArgs{Command: cmd})
	if exitCode(err) == 1 {
		err = NewErr(ErrConfigNotSet, "key", key, err)
		goto end
	}
	if err != nil {
		err = NewErr(ErrGetConfig, "key", key, err)
		goto end
	}
end:
	return value, err
}

// ignoreConfigNotSet returns nil for an ErrConfigNotSet error, and err
// otherwise.
func ignoreConfigNotSet(err error) error {
	if errors.Is(err, ErrConfigNotSet) {
		err = nil
	}
	return err
}

// newRepoConfig returns config with identity's user.name and user.email
// added, identity taking precedence. Both may be nil.
func newRepoConfig(config map[string]string, identity *dt.Mailbox) (out map[string]string) {
	out = make(map[string]string, len(config)+2)
	maps.Copy(out, config)
	if identity != nil {
		out["user.name"] = string(identity.DisplayName)
		out["user.email"] = string(identity.Email)
	}
	return out
}

// sortedKeys returns m's keys in sorted order.
func sortedKeys(m map[string]string) []string {
	return slices.Sorted(maps.Keys(m))
}

// checkIdentity refuses a non-nil identity with an empty DisplayName, which
// field names (Identity, Author, Committer).
func checkIdentity(field string, identity *dt.Mailbox) (err error) {
	if identity == nil {
		goto end
	}
	if strings.TrimSpace(string(identity.DisplayName)) == "" {
		err = NewErr(ErrIdentityNoDisplayName, "field", field, "identity", identity.String())
	}
end:
	return err
}
