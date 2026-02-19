package lspproxy

import (
	"bufio"
	"fmt"
	"io"
	"log"
)

// processServerToEditor continuously reads from the language server,
// intercepts the initialize response to force Full Sync, and forwards to the editor.
func (p *Proxy) processServerToEditor() {
	reader := bufio.NewReader(p.serverOut)

	for {

		payload, err := readLSPMessage(reader)
		if err != nil {
			if err == io.EOF {
				// The server closed the connection
				return
			}
			log.Fatalf("[%s] Fatal error reading header from server: %v", componentName, err)
		}

		// Intercept and optionally rewrite the payload
		modifiedPayload := p.forceFullSync(payload)

		header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(modifiedPayload))
		if _, err := p.editorOut.Write([]byte(header)); err != nil {
			log.Printf("[%s] Error writing header to editor: %v", componentName, err)
			return
		}

		if _, err := p.editorOut.Write(modifiedPayload); err != nil {
			log.Printf("[%s] Error writing payload to editor: %v", componentName, err)
			return
		}
	}
}
