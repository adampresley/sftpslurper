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

- Simple authentication with configurable username and password
- Customizable listening address and port
- Configurable upload directory
- Support for standard SFTP operations (put, get, list, delete)
- Comprehensive logging of connection attempts and operations

## Configuration Options

SFTP Slurper can be configured using command-line flags or environment variables:

| Option | Flag | Environment Variable | Default Value | Description |
|--------|------|----------------------|---------------|-------------|
| Address | `--address` | `ADDRESS` | `localhost:22` | Address and port to listen on |
| Upload Folder | `--uploadpath` | `UPLOAD_PATH` | `./uploads` | Directory where uploaded files are stored |
| Username | `-u` | `FTP_USER` | `testuser` | Username for SFTP authentication |
| Password | `-p` | `FTP_PASSWORD` | `password` | Password for SFTP authentication |

## Installation

### Prerequisites

- Go 1.18 or higher

### Building from Source

```bash
git clone https://github.com/adampresley/sftpslurper.git
cd sftpslurper
go build
```

### Basic Usage

Run the server with default settings:

```bash
./sftpslurper
```

This will start an SFTP server on localhost:22 with the username testuser and password password. Files will be uploaded to the ./uploads directory.

### Custom Configuration

Using command-line flags:

`./sftpslurper --address=0.0.0.0:2222 --uploadpath=/tmp/sftp-uploads -u myuser -p mysecretpassword`

Using environment variables:

`ADDRESS=0.0.0.0:2222 UPLOAD_PATH=/tmp/sftp-uploads FTP_USER=myuser FTP_PASSWORD=mysecretpassword ./sftpslurper`

### Connecting to the Server

You can connect to the server using any SFTP client. For example, using the command-line sftp client:

`sftp -P 22 testuser@localhost`

Or with custom port:

`sftp -P 2222 myuser@localhost`

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

## License

MIT License
