package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/adampresley/adamgokit/waiter"
	"github.com/app-nerds/configinator"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Config struct {
	Address      string `flag:"address" env:"ADDRESS" default:"localhost:22" description:"Address to listen on"`
	UploadFolder string `flag:"uploadpath" env:"UPLOAD_PATH" default:"./uploads" description:"Folder to store uploaded files"`
	User         string `flag:"u" env:"FTP_USER" default:"testuser" description:"Username for SFTP"`
	Password     string `flag:"p" env:"FTP_PASSWORD" default:"password" description:"Password for SFTP"`
}

func main() {
	var (
		err      error
		hostKey  ssh.Signer
		listener net.Listener
	)

	/*
	 * First get our configuration information.
	 */
	appConfig := Config{}
	configinator.Behold(&appConfig)

	// Generate an ephemeral host key.
	if hostKey, err = generateHostKey(); err != nil {
		log.Fatalf("Error generating host key: %v", err)
	}

	// Create the SSH server configuration with a PasswordCallback.
	sshConfig := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			log.Printf("Configured user: '%s', user received: '%s', configured password: '%s', password received: '%s'", appConfig.User, c.User(), appConfig.Password, string(password))
			if c.User() == appConfig.User && string(password) == appConfig.Password {
				return nil, nil
			}

			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}

	// Specify the allowed ciphers.
	// availableCiphers := ssh.s

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

	if listener, err = net.Listen("tcp", appConfig.Address); err != nil {
		log.Fatalf("failed to listen on %s: %v", appConfig.Address, err)
	}

	defer listener.Close()
	log.Printf("listening on %s", appConfig.Address)

	quit := waiter.Wait()

	// Listen for incoming SSH connections and handle them concurrently.
	for {
		select {
		case <-quit:
			break

		default:
			nConn, err := listener.Accept()
			if err != nil {
				log.Printf("failed to accept incoming connection: %v", err)
				continue
			}

			// Handle each connection in a separate goroutine
			go handleConnection(nConn, sshConfig, appConfig.UploadFolder)
		}
	}

	log.Printf("server stopped")
}

func handleConnection(nConn net.Conn, sshConfig *ssh.ServerConfig, uploadFolder string) {
	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, sshConfig)
	if err != nil {
		log.Printf("failed to establish SSH connection: %v", err)
		nConn.Close()
		return
	}

	log.Printf("new SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

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
			log.Printf("could not accept channel: %v", err)
			continue
		}

		// Handle session requests in a separate goroutine
		go handleSessionRequests(channel, requests, uploadFolder)
	}
}

func handleSessionRequests(channel ssh.Channel, requests <-chan *ssh.Request, uploadFolder string) {
	defer channel.Close()

	for req := range requests {
		log.Printf("Received request: %s", req.Type)

		switch req.Type {
		case "subsystem":
			if len(req.Payload) >= 4 && string(req.Payload[4:]) == "sftp" {
				log.Printf("SFTP subsystem requested")

				// Reply true to the subsystem request
				if req.WantReply {
					req.Reply(true, nil)
				}

				handler := &SFTPHandler{RootPath: uploadFolder}

				// Create SFTP server
				server := sftp.NewRequestServer(channel, sftp.Handlers{
					FilePut:  handler,
					FileGet:  handler,
					FileCmd:  handler,
					FileList: handler,
				})

				log.Printf("Starting SFTP server")
				if err := server.Serve(); err == io.EOF {
					log.Printf("SFTP session ended")
					server.Close()
					return
				} else if err != nil {
					log.Printf("SFTP server error: %v", err)
					return
				}
			} else {
				log.Printf("Unknown subsystem: %s", req.Payload)
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
