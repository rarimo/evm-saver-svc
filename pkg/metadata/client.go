package metadata

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rarimo/evm-saver-svc/pkg/ipfs"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

const maxBodySize = 10 << 20 // 10 MB

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	client  HttpClient
	ipfs    ipfs.Gateway
	timeout time.Duration
}

func NewClient(client HttpClient, ipfs ipfs.Gateway, timeout time.Duration) *Client {
	return &Client{
		client:  client,
		ipfs:    ipfs,
		timeout: timeout,
	}
}

type unmarshalerFrom interface {
	UnmarshalFrom(json.RawMessage) error
}

type uriPopulator interface {
	PopulateURI(uri string)
}

func (c *Client) LoadMetadata(ctx context.Context, uri string, out interface{}) error {
	if pu, ok := out.(uriPopulator); ok {
		pu.PopulateURI(uri)
	}

	metadataPayload, err := c.getMetadataPayload(ctx, uri)
	if err != nil {
		return errors.Wrap(err, "failed to get metadata payload")
	}

	if u, ok := out.(unmarshalerFrom); ok {
		return u.UnmarshalFrom(metadataPayload)
	}

	err = json.Unmarshal(metadataPayload, out)
	return errors.Wrap(err, "failed to unmarshal metadata payload")
}

func (c *Client) getMetadataPayload(ctx context.Context, uri string) (json.RawMessage, error) {
	uri = StripIPFSGateway(uri)
	uri = StripArweaveGateway(uri)

	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse uri")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	payload, err := c.load(ctx, parsedURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load metadata")
	}

	return sanitizeJSON(payload), nil
}

func (c *Client) load(ctx context.Context, url *url.URL) ([]byte, error) {
	fields := logan.F{
		"schema": url.Scheme,
		"uri":    url.String(),
	}

	switch url.Scheme {
	case "http", "https":
		return c.getHttpMeta(ctx, *url)
	case "ipfs":
		return c.ipfs.Get(ctx, url.Host+url.Path)
	case "data":
		return parseData(*url)
	case "":
		return nil, errors.New("empty url schema")
	default:
		return nil, errors.From(errors.New("unexpected schema"), fields)
	}
}

func (c *Client) DownloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	imageURL = StripIPFSGateway(imageURL)
	imageURL = StripArweaveGateway(imageURL)

	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse uri")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.load(ctx, parsedURL)
}

func (c *Client) getHttpMeta(ctx context.Context, metaURL url.URL) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metaURL.String(), nil)
	if err != nil {
		panic(errors.Wrap(err, "failed to create request"))
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request")
	}

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		reader := io.LimitReader(resp.Body, maxBodySize)
		result, err := io.ReadAll(reader)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read body")
		}

		return result, nil
	case http.StatusForbidden, http.StatusUnauthorized:
		return nil, errors.From(errors.New("not allowed to access url"), logan.F{
			"url": metaURL.String(),
		})
	case http.StatusNotFound:
		return nil, errors.From(errors.New("not found resources url pointing to"), logan.F{
			"url": metaURL.String(),
		})
	default:
		return nil, errors.From(errors.New("unexpected status code"), logan.F{
			"status_code": resp.StatusCode,
		})
	}
}

func parseData(u url.URL) ([]byte, error) {
	mimeTypeData := strings.Split(u.Opaque, ";")
	if len(mimeTypeData) != 2 {
		return nil, errors.New("unexpected number of data url parts")
	}

	mimeType := mimeTypeData[0]
	if mimeType != "application/json" {
		return nil, errors.From(errors.New("unexpected mime type"), logan.F{
			"mime_type": mimeType,
		})
	}

	dataPart := mimeTypeData[1]
	encodingPayload := strings.Split(dataPart, ",")
	if len(encodingPayload) != 2 {
		return nil, errors.New("unexpected format of data url's payload")
	}

	encoding := encodingPayload[0]
	if encoding != "base64" {
		return nil, errors.New("expected base64 encoding")
	}

	payload, err := base64.StdEncoding.DecodeString(encodingPayload[1])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode payload")
	}

	return payload, nil
}
