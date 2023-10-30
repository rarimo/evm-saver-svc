package metadata

import (
	"fmt"
	"strings"
)

// StripIPFSProtocol - strips IPFS URL protocol and returns URL in HTTP format with given gateway
func StripIPFSProtocol(imageSrc string, ipfsGateway string) string {
	if strings.HasPrefix(imageSrc, "ipfs") {
		return fmt.Sprintf("%s/%s", ipfsGateway, strings.ReplaceAll(imageSrc, "ipfs://", ""))
	}
	return imageSrc
}

// StripArweaveGateway - strips Arweave URL protocol and returns URL in HTTP format
func StripArweaveGateway(imageSrc string) string {
	if strings.HasPrefix(imageSrc, "ar") {
		return fmt.Sprintf("%s/%s", "https://arweave.net", strings.ReplaceAll(imageSrc, "ar://", ""))
	}
	return imageSrc
}

var publicIpfsGateways = []string{
	"https://ipfs.io/ipfs/",
	"https://gateway.pinata.cloud/ipfs/",
	"https://cloudflare-ipfs.com/ipfs/",
	"https://gateway.ipfs.io/ipfs/",
	"https://dweb.link/ipfs/",
	"https://hardbin.com/ipfs/",
	"https://ipfs.fleek.co/ipfs/",
	"https://jorropo.net/ipfs/",
	"https://ipfs.eth.aragon.network/ipfs/",
	"https://cloudflare-ipfs.com/ipfs/",
	"https://storry.tv/ipfs/",
	"https://ipfs.telos.miami/ipfs/",
	"https://via0.com/ipfs/",
	"https://ipfs.infura.io/ipfs/",
	"https://infura-ipfs.io/ipfs/",
	"https://ipfs.mihir.ch/ipfs/",
	"https://nftstorage.link/ipfs/",
	"https://cf-ipfs.com/ipfs/",
	"https://gateway.pinata.cloud/ipfs/",
	"https://ipfs.azurewebsites.net/ipfs/",
	"https://permaweb.eu.org/ipfs/",
}

var httpPrefix = "http"
var httpsPrefix = "https"

// StripIPFSGateway - strips public IPFS gateways URLs, and returns URL in IPFS protocol format
func StripIPFSGateway(uri string) string {
	if !strings.HasPrefix(uri, httpPrefix) {
		return uri
	}

	sourceUrl := uri

	if !strings.HasPrefix(uri, httpsPrefix) {
		sourceUrl = fmt.Sprintf("%s://", httpsPrefix) + strings.ReplaceAll(uri, fmt.Sprintf("%s://", httpPrefix), "")
	}

	for _, ipfsGatewayToReplace := range publicIpfsGateways {
		if strings.HasPrefix(sourceUrl, ipfsGatewayToReplace) {
			return "ipfs://" + strings.ReplaceAll(sourceUrl, ipfsGatewayToReplace, "")
		}
	}

	return uri
}
