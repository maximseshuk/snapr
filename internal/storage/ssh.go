package storage

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

const sshDefaultPort = 22

func dialSSH(storage pkgconfig.StorageConfig) (*ssh.Client, error) {
	username := storage.Username
	if username == "" {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("ssh username not set and current user lookup failed: %w", err)
		}
		username = u.Username
	}

	auths, err := buildSSHAuth(storage)
	if err != nil {
		return nil, err
	}
	if len(auths) == 0 {
		return nil, fmt.Errorf("ssh: no auth methods configured (provide privateKey or password)")
	}

	hostKeyCallback, err := buildHostKeyCallback(storage)
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User:            username,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	port := storage.Port
	if port == 0 {
		port = sshDefaultPort
	}
	addr := net.JoinHostPort(storage.Host, fmt.Sprintf("%d", port))

	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return client, nil
}

func buildSSHAuth(storage pkgconfig.StorageConfig) ([]ssh.AuthMethod, error) {
	var auths []ssh.AuthMethod

	if storage.PrivateKey != "" {
		keyPath := expandHome(storage.PrivateKey)
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key %s: %w", keyPath, err)
		}

		var signer ssh.Signer
		if storage.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(storage.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key %s: %w", keyPath, err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}

	if storage.Password != "" {
		auths = append(auths, ssh.Password(storage.Password))
	}

	return auths, nil
}

// strictHostKey=false disables host-key checking — unsafe for production.
func buildHostKeyCallback(storage pkgconfig.StorageConfig) (ssh.HostKeyCallback, error) {
	if storage.StrictHostKey != nil && !*storage.StrictHostKey {
		return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec // explicit user opt-in via StrictHostKey=false
	}

	khPath := storage.KnownHosts
	if khPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("locate user home for known_hosts: %w", err)
		}
		khPath = filepath.Join(home, ".ssh", "known_hosts")
	}
	khPath = expandHome(khPath)

	if _, err := os.Stat(khPath); err != nil {
		return nil, fmt.Errorf("known_hosts %s not found (set strictHostKey: false to skip): %w", khPath, err)
	}

	cb, err := knownhosts.New(khPath)
	if err != nil {
		return nil, fmt.Errorf("parse known_hosts %s: %w", khPath, err)
	}
	return cb, nil
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) == 1 {
		return home
	}
	if path[1] == '/' {
		return filepath.Join(home, path[2:])
	}
	return path
}
