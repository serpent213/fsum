package main

import (
  "bytes"
  "crypto/tls"
  "fmt"
  "io/ioutil"
  "log"
  "math"
  "net/http"
  "os"
  "path/filepath"

  "github.com/tidwall/gjson"
)

const (
  RPC_WALLET_PORT = 9256
)

func TLSJSONRequest(hostname string, port int, certFile string, keyFile string, method string) gjson.Result {
  // load cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalln(err)
	}

	// setup HTTP client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	tr := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: tr}

	// talk to RPC service
	empty_json := bytes.NewBufferString("{}")
  resp, err := client.Post(fmt.Sprintf("https://%s:%d/%s", hostname, port, method), "application/json", empty_json)
	if err != nil {
		log.Fatalf("Error accessing RPC: %#v\n", err)
	}
	defer resp.Body.Close()

	// evaluate response
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading response body: %#v\n", err)
		}
    validJSON := gjson.ValidBytes(body)
    if !validJSON {
			log.Fatalln("Response is not valid JSON")
    }
    return gjson.ParseBytes(body)
	} else {
    log.Fatalln("RPC server returned error")
  }

  return gjson.Parse("{}")
}

func main() {
	// find client cert
	user_home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}
	certFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/wallet/private_wallet.crt")
	keyFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/wallet/private_wallet.key")

  // perform RPC request
  result := TLSJSONRequest("localhost", RPC_WALLET_PORT, certFile, keyFile, "get_farmed_amount")
  // log.Printf("Result: %+v\n", result)

  // display results
  farmedAmount := result.Get("farmed_amount").Uint()
  log.Printf("Farmed Amount: %d Mojo (%f XCH)\n", farmedAmount, float64(farmedAmount) / math.Pow(10, 12))
}
