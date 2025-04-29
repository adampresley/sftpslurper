package sftp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"

	"github.com/adampresley/sftpslurper/cmd/sftpslurper/internal/configuration"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func StartServer(config *configuration.Config, shutdownCtx context.Context) {
	go func() {
		var (
			err      error
			hostKey  ssh.Signer
			listener net.Listener
		)

		// Generate an ephemeral host key.
		if hostKey, err = generateHostKey(); err != nil {
			slog.Error("Error generating host key", "error", err)
			os.Exit(1)
		}

		// Create the SSH server configuration with a PasswordCallback.
		sshConfig := &ssh.ServerConfig{
			PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
				slog.Info("user is logging in...", "user", c.User())

				if c.User() == configuration.SftpUserName && string(password) == configuration.SftpPassword {
					return nil, nil
				}

				return nil, fmt.Errorf("password rejected for %q", c.User())
			},
		}

		// Specify the allowed ciphers.
		// TODO: This should probably be done a different way.

		sshConfig.Config = ssh.Config{
			Ciphers: []string{ // Modern, secure ciphers
				"chacha20-poly1305@openssh.com",
				"aes128-gcm@openssh.com",
				"aes256-gcm@openssh.com",
				"aes128-ctr",
				"aes192-ctr",
				"aes256-ctr",

				// Older ciphers kept for compatibility
				"3des-cbc",
				"blowfish-cbc",
				"aes128-cbc",
				"aes192-cbc",
				"aes256-cbc",
			},
			KeyExchanges: []string{
				"curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256",
				"ecdh-sha2-nistp384",
				"ecdh-sha2-nistp521",
				"diffie-hellman-group14-sha1",
				"diffie-hellman-group1-sha1",
			},
			MACs: []string{
				"hmac-sha2-256-etm@openssh.com",
				"hmac-sha2-256",
				"hmac-sha1",
				"hmac-sha1-96",
			},
		}

		// Add the generated host key to the server configuration.
		sshConfig.AddHostKey(hostKey)

		if listener, err = net.Listen("tcp", config.SftpHost); err != nil {
			slog.Error("failed to start SFTP server", "host", config.SftpHost, "error", err)
			os.Exit(1)
		}

		defer listener.Close()
		slog.Info("SFTP server started", "host", config.SftpHost)

		// Listen for incoming SSH connections and handle them concurrently.
		for {
			select {
			case <-shutdownCtx.Done():
				break

			default:
				nConn, err := listener.Accept()
				if err != nil {
					log.Printf("failed to accept incoming connection: %v", err)
					continue
				}

				// Handle each connection in a separate goroutine
				go handleConnection(nConn, sshConfig, configuration.UploadFolder)
			}
		}

		slog.Info("SFTP server stopped")
	}()
}

func handleConnection(nConn net.Conn, sshConfig *ssh.ServerConfig, uploadFolder string) {
	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, sshConfig)

	if err != nil {
		slog.Error("failed to establish SSH connection", "error", err)
		nConn.Close()
		return
	}

	slog.Info("new SSH connection", "remote_addr", sshConn.RemoteAddr(), "client_version", sshConn.ClientVersion())

	// Discard all global requests
	go ssh.DiscardRequests(reqs)

	// Handle all channels
	go handleChannels(chans, uploadFolder)
}

func handleChannels(chans <-chan ssh.NewChannel, uploadFolder string) {
	for newChannel := range chans {
		// Only accept session channels.
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()

		if err != nil {
			slog.Error("could not accept channel", "error", err)
			continue
		}

		// Handle session requests in a separate goroutine
		go handleSessionRequests(channel, requests, uploadFolder)
	}
}

func handleSessionRequests(channel ssh.Channel, requests <-chan *ssh.Request, uploadFolder string) {
	defer channel.Close()

	for req := range requests {
		slog.Info("received request", "type", req.Type)

		switch req.Type {
		case "subsystem":
			if len(req.Payload) >= 4 && string(req.Payload[4:]) == "sftp" {
				slog.Info("SFTP subsystem requested")

				// Reply true to the subsystem request
				if req.WantReply {
					req.Reply(true, nil)
				}

				handler := &Handler{RootPath: uploadFolder}

				// Create SFTP server
				server := sftp.NewRequestServer(channel, sftp.Handlers{
					FilePut:  handler,
					FileGet:  handler,
					FileCmd:  handler,
					FileList: handler,
				})

				slog.Info("starting SFTP server")

				if err := server.Serve(); err == io.EOF {
					slog.Error("SFTP session ended")
					server.Close()
					return
				} else if err != nil {
					slog.Error("SFTP server error", "error", err)
					return
				}
			} else {
				slog.Error("unknown subssytem request", "payload", string(req.Payload))

				if req.WantReply {
					req.Reply(false, nil)
				}
			}

		default:
			// Reply false to other requests
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// generateHostKey creates an ephemeral RSA host key.
func generateHostKey() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %v", err)
	}
	return signer, nil
}
