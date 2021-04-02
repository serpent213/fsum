// port of "chia farm summary"
// https://github.com/Chia-Network/chia-blockchain/blob/1.0.3/src/cmds/farm.py

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/bits"
	"net/http"
	"os"
	"path/filepath"

  // "github.com/davecgh/go-spew/spew"
	"github.com/tidwall/gjson"
)

const (
	RPC_NODE_HOST      = "localhost"
	RPC_NODE_PORT      = 8555
	RPC_WALLET_HOST    = "localhost"
	RPC_WALLET_PORT    = 9256
	RPC_FARMER_HOST    = "localhost"
	RPC_FARMER_PORT    = 8559
	RPC_HARVESTER_HOST = "localhost"
	RPC_HARVESTER_PORT = 8560
)

func tlsJSONRequest(hostname string, port int, certFile string, keyFile string, method string) gjson.Result {
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

func nodeRPC(method string) gjson.Result {
	// find client cert
	user_home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}
	certFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/full_node/private_full_node.crt")
	keyFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/full_node/private_full_node.key")

	// perform RPC request
	return tlsJSONRequest(RPC_NODE_HOST, RPC_NODE_PORT, certFile, keyFile, method)
}

func walletRPC(method string) gjson.Result {
	// find client cert
	user_home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}
	certFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/wallet/private_wallet.crt")
	keyFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/wallet/private_wallet.key")

	// perform RPC request
	return tlsJSONRequest(RPC_WALLET_HOST, RPC_WALLET_PORT, certFile, keyFile, method)
}

func farmerRPC(method string) gjson.Result {
	// find client cert
	user_home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}
	certFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/farmer/private_farmer.crt")
	keyFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/farmer/private_farmer.key")

	// perform RPC request
	return tlsJSONRequest(RPC_FARMER_HOST, RPC_FARMER_PORT, certFile, keyFile, method)
}

func isFarmerRunning() bool {
	farmerConnections := farmerRPC("get_connections")
  connCount := farmerConnections.Get("connections.#").Int()
  log.Printf("Farmer connections: %d\n", connCount)
  return connCount > 0
}

func harvesterRPC(method string) gjson.Result {
	// find client cert
	user_home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}
	certFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/harvester/private_harvester.crt")
	keyFile := filepath.Join(user_home, ".chia/mainnet/config/ssl/harvester/private_harvester.key")

	// perform RPC request
	return tlsJSONRequest(RPC_HARVESTER_HOST, RPC_HARVESTER_PORT, certFile, keyFile, method)
}

func humanBytes(bytes uint64) string {
  if bytes < 1024 {
    return fmt.Sprintf("%d bytes", bytes)
  }

  base := uint(bits.Len64(bytes) / 10)
  val := float64(bytes) / float64(uint64(1 << (base * 10)))

  return fmt.Sprintf("%.3f %ciB", val, " KMGTPE"[base])
}

func main() {
	// perform RPC requests
	nodeBlockchainState := nodeRPC("get_blockchain_state")
	log.Printf("Blockchain state: %s\n", nodeBlockchainState.Get("@this|@pretty"))
	walletFarmedAmount := walletRPC("get_farmed_amount")
	log.Printf("Farmed amount: %s\n", walletFarmedAmount.Get("@this|@pretty"))
	// farmerConnections := FarmerRPC("get_connections")
	// log.Printf("Farmer connections: %s\n", farmerConnections.Get("@this|@pretty"))
  farmerRunning := isFarmerRunning()
	harvesterPlots := harvesterRPC("get_plots")
	// log.Printf("Harvester plots: %s\n", harvesterPlots.Get("@this|@pretty"))

  // calculations
	farmedAmountMojo := walletFarmedAmount.Get("farmed_amount").Uint()
  farmedAmountXCH := float64(farmedAmountMojo) / math.Pow(10, 12)
  var total_plot_size uint64
  for _, psize := range harvesterPlots.Get("plots.#.file_size").Array() {
    total_plot_size += psize.Uint()
  }

  fmt.Printf("Farmer running: %v\n", farmerRunning)
	fmt.Printf("Farmed Amount: %d Mojo (%f XCH)\n", farmedAmountMojo, farmedAmountXCH)
  fmt.Printf("Number of plots: %d\n", harvesterPlots.Get("plots.#").Int())
  fmt.Printf("Total plot size: %s\n", humanBytes(total_plot_size))
  fmt.Printf("Estimated network space: %s\n", humanBytes(nodeBlockchainState.Get("blockchain_state.space").Uint()))

  // display expected time to win

}
