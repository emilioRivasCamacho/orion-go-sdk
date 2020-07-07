package http2

import (
	"crypto/tls"
	"io/ioutil"
	"log"

	"github.com/gig/orion-go-sdk/env"
)

func TLSConfig() *tls.Config {
	skipVerification := env.Truthy("SKIP_TRAEFIK_VERIFICATION_CACERT")

	crtFilename := env.Get("ORION_DEFAULT_SSL_CERT", "")
	keyFilename := env.Get("ORION_DEFAULT_SSL_KEY", "")

	if crtFilename == "" || keyFilename == "" {
		log.Fatal("Missing ORION_DEFAULT_SSL_CERT or ORION_DEFAULT_SSL_KEY ")
	}

	crt, err := ioutil.ReadFile(crtFilename)
	if err != nil {
		log.Fatal(err)
	}

	key, err := ioutil.ReadFile(keyFilename)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		log.Fatal(err)
	}

	return &tls.Config{
		InsecureSkipVerify: skipVerification,
		Certificates:       []tls.Certificate{cert},
		ServerName:         "localhost",
	}
}
