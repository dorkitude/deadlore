package cli

import (
	"fmt"
	"io"
)

func writeFallbackNotice(output io.Writer, fallback bool) {
	if !fallback {
		return
	}
	writeBox(output, "Source fallback", []string{"The Deadlock Wiki did not provide this data; showing the Deadlock.io fallback."})
	fmt.Fprintln(output)
}
