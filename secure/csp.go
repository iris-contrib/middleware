package secure

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/kataras/iris/v12"
)

const cspNonceKey string = "iris.secure.nonce"

// CSPNonce returns the nonce value associated with the present request. If no nonce has been generated it returns an empty string.
func CSPNonce(ctx iris.Context) string {
	return ctx.Values().GetString(cspNonceKey)
}

func cspRandNonce() string {
	var buf [cspNonceSize]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic("CSP Nonce rand.Reader failed" + err.Error())
	}

	return base64.RawStdEncoding.EncodeToString(buf[:])
}
