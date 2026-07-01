package ws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Legacy CLI/binary names kept for backwards-compatible invocation and self-update.
const (
	cliNameWS   = "ws"
	cliNameKuba = "kuba"
	installURL  = "https://withsecrets.com/installation"
)

var deprecationShown = false

// CLIName returns the command name based on how the binary was invoked.
func CLIName() string {
	name := filepath.Base(os.Args[0])
	name = strings.TrimSuffix(name, ".exe")
	if name == cliNameKuba {
		return cliNameKuba
	}
	return cliNameWS
}

// BinaryAssetName returns the release asset prefix for self-update.
func BinaryAssetName() string {
	if CLIName() == cliNameKuba {
		return cliNameKuba
	}
	return cliNameWS
}

func maybeShowDeprecationNotice() {
	if CLIName() != cliNameKuba || deprecationShown {
		return
	}
	deprecationShown = true
	fmt.Fprintf(os.Stderr, "Note: the kuba command was renamed to ws (withsecrets). See %s\n", installURL)
}
