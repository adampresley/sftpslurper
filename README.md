# SFTP Slurper

SFTP Slurper is a lightweight, configurable SFTP server designed for local testing of SFTP connections and file uploads. It provides a simple way to simulate an SFTP server environment without the need for complex setup, making it ideal for development and testing scenarios.

## Purpose

This application allows developers to:
- Test SFTP client implementations
- Verify file upload functionality
- Debug SFTP connection issues
- Simulate an SFTP server locally without complex configuration

All uploaded files are stored in a configurable local directory, making it easy to inspect and validate the uploaded content.

## Features

- Simple authentication with predefined username and password
- Customizable listening address and port
- Support for standard SFTP operations (put, get, list, delete)

## Configuration Options

SFTP Slurper can be configured using command-line flags or environment variables:

| Option | Flag | Environment Variable | Default Value | Description |
|--------|------|----------------------|---------------|-------------|
| Web Host | `-h` | `HOST` | `localhost:8080` | Address and port to listen on for the Web interface |
| SFTP Address | `-sftph` | `SFTP_HOST` | `localhost:2200` | Address and port to listen on for the SFTP server |

## Installation

### Prerequisites

- Go 1.24 or higher

### Building from Source

```bash
git clone https://github.com/adampresley/sftpslurper.git
cd sftpslurper/cmd/sftpslurper
go build
```

### Basic Usage

Run the server with default settings:

```bash
./sftpslurper
```

This will start an SFTP server on localhost:2200 with the username testuser and password password. Files will be uploaded to the ./uploads directory. To view the web interface visit `http://localhost:8080`.

### Custom Configuration

Using command-line flags:

`./sftpslurper -h=0.0.0.0:8080 -sftph=0.0.0.0:2222

Using environment variables:

`HOST=0.0.0.0:8080 SFTP_HOST=0.0.0.0:2222 ./sftpslurper`

#### Note About *nix Systems
_On most Unix-like systems (Linux, MacOS), trying to bind to a port lower than 1000 will require administrative privileges._

### Connecting to the Server

You can connect to the server using any SFTP client. For example, using the command-line sftp client. The user name and password are:

- **User name**: `user`
- **Password**: `password`

`sftp -P 22 user@localhost`

Or with custom port:

`sftp -P 2222 user@localhost`

Stopping the Server

The server can be stopped by sending an interrupt signal (Ctrl+C).

## Example Operations

Once connected, you can perform standard SFTP operations:

```
# Upload a file
put localfile.txt remotefile.txt

# Download a file
get remotefile.txt downloadedfile.txt

# List files
ls

# Change directory
cd some-directory

# Remove a file
rm file-to-delete.txt
```

## Security Considerations

SFTP Slurper is designed for local development and testing purposes only. It includes:

- Support for various SSH ciphers and key exchange methods for compatibility
- Basic password authentication

However, it is **not recommended** for production use as it:

- Generates ephemeral RSA keys on startup
- Uses simple password authentication
- May not implement all security best practices required for production environments

**USE AT YOUR OWN RISK!**

## License

MIT License
