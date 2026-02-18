package lspproxy

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

// readLSPMessage reads the HTTP-like headers to find the Content-Length,
// then reads and returns the exact JSON payload.
func readLSPMessage(reader *bufio.Reader) ([]byte, error) {
	var contentLength int

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err // Usually io.EOF when the connection closes
		}

		line = strings.TrimSpace(line)

		// An empty line marks the end of the headers
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				contentLength, err = strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("invalid Content-Length value '%s': %w", val, err)
				}
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing or zero Content-Length header")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("failed to read full payload of size %d: %w", contentLength, err)
	}

	return payload, nil
}

// forwardToServer cleanly re-serializes the header and sends the exact payload to the language server.
func (p *Proxy) forwardToServer(payload []byte) {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))

	if _, err := p.serverIn.Write([]byte(header)); err != nil {
		log.Printf("Error writing header to server: %v", err)
		return
	}

	if _, err := p.serverIn.Write(payload); err != nil {
		log.Printf("Error writing payload to server: %v", err)
	}
}
