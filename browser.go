package main

import (
	"io"

	"github.com/cli/go-gh/v2/pkg/browser"
)

// launcher honors the same browser settings as gh (GH_BROWSER, gh config's
// browser, BROWSER) before falling back to the OS default. Its output is
// discarded so a chatty launcher can't scribble over the picker.
var launcher = browser.New("", io.Discard, io.Discard)

func openURL(url string) error {
	return launcher.Browse(url)
}
