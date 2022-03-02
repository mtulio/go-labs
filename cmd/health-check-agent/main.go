package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	flag "github.com/spf13/pflag"
)

type Profiles map[string]*tls.Config

var (
	profiles      Profiles
	watch         bool    = true
	url           *string = flag.String("url", "", "URL to check.")
	tlsProfile    *string = flag.String("tls-profile", "default", "TLS Profile.")
	watchInterval *uint8  = flag.Uint8("watch-interval", 10, "Interval to perform checks to URL.")
	watchCount    *uint8  = flag.Uint8("watch-count", 1, "Total of checks.")
	disableKA     *bool   = flag.Bool("no-keep-alive", false, "Total of checks.")
	dbTlsLogKeys  *string = flag.String("debug-tls-keys-log-file", "", "TLS Keys file")
)

func init() {
	// TLS client profiles
	profiles = make(map[string]*tls.Config)

	profiles["default"] = &tls.Config{}
	profiles["insecure"] = &tls.Config{
		InsecureSkipVerify: true,
	}
	profiles["tls12"] = &tls.Config{
		MaxVersion: tls.VersionTLS12,
	}
	profiles["tls13"] = &tls.Config{
		MaxVersion: tls.VersionTLS13,
	}
	profiles["tls12i"] = &tls.Config{
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS12,
	}
	profiles["tls13i"] = &tls.Config{
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS13,
	}

	flag.Parse()
	fmt.Println(*url)
	if *url == "" {
		log.Println("URL must be set: --url")
		os.Exit(1)
	}
	if _, ok := profiles[*tlsProfile]; !ok {
		log.Println("--profile not found.")
		log.Printf("Valie profiles are: %+v", profiles)
		os.Exit(1)
	}

}

func main() {
	var watchCounter uint8 = 1
	for watch {
		if *watchCount == watchCounter {
			watch = false
		}
		if (watchCounter != 1) && (*watchCount > 1) {
			time.Sleep(time.Duration(*watchInterval) * time.Second)

		}
		watchCounter++

		t := http.DefaultTransport.(*http.Transport).Clone()
		if *disableKA {
			t.DisableKeepAlives = true
			t.MaxIdleConnsPerHost = -1
		}
		t.TLSClientConfig = profiles[*tlsProfile]
		if *dbTlsLogKeys != "" {
			tlsKeysFD, err := os.OpenFile(*dbTlsLogKeys, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				panic(err)
			}
			defer tlsKeysFD.Close()
			t.TLSClientConfig.KeyLogWriter = tlsKeysFD
		}
		c := &http.Client{Transport: t}

		// http.DefaultTransport.(*http.Transport).TLSClientConfig = profiles[*tlsProfile]
		resp, err := c.Get(*url)
		if err != nil {
			log.Println(fmt.Sprintf("ERROR received from server. Delaying 10s: %s", err))
			continue
		}
		log.Println(resp)
	}
}
