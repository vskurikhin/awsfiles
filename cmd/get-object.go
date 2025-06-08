package cmd

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	Address         string `mapstructure:"address"`
	Bucket          string `mapstructure:"bucket"`
	BufferSize      int    `mapstructure:"buffer_size"`
	Key             string `mapstructure:"key"`
	S3Host          string `mapstructure:"s3_host"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

var zeroTime = new(time.Time)

func getObject(cfg Config) {
	keys := getKeys(cfg)
	addr, s3host := getAddressS3Host(cfg)
	httpclient := getHTTPClient(cfg, addr)
	customResolver := getCustomResolver(s3host)
	creeds := credentials.NewStaticCredentialsProvider(keys.AccessKeyID, keys.SecretAccessKey, "")
	client := s3.NewFromConfig(aws.Config{
		Credentials:                 creeds,
		HTTPClient:                  httpclient,
		EndpointResolverWithOptions: customResolver,
	})
	slog.Info("Getting object", "client", "prepared")
	res, err := clientGetObject(cfg, client)
	if err != nil {
		slog.Error("Get object failed", "err", err)
		return
	}
	hasher := md5.New()
	b := make([]byte, cfg.BufferSize)
	bytesWritten := 0
	fmt.Println()
	for {
		i, err := res.Body.Read(b)
		if err != nil && err != io.EOF {
			fmt.Printf("\nerror: %v\n", err.Error())
			break
		} else if err == io.EOF {
			bytesWritten += i
			hasher.Write(b[:i])
			break
		} else {
			bytesWritten += i
			hasher.Write(b[:i])
		}
		fmt.Printf("read bytes: %d, total read bytes %d\r", i, bytesWritten)
	}
	fmt.Println()
	slog.Info("Get object", "result", fmt.Sprintf("total read bytes: %d", bytesWritten))
	value := hex.EncodeToString(hasher.Sum(nil))
	slog.Info("Get object", "result", fmt.Sprintf("md5sum: %s", value))
}

func clientGetObject(cfg Config, client *s3.Client) (*s3.GetObjectOutput, error) {
	return client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cfg.Key),
	})
}

func getCustomResolver(s3host string) aws.EndpointResolverWithOptionsFunc {
	return func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if region == "" {
			region = "us-east-1"
		}
		if service == s3.ServiceID {
			return aws.Endpoint{URL: s3host, HostnameImmutable: true, SigningRegion: region}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	}
}

func getHTTPClient(cfg Config, address string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:     false,
			IdleConnTimeout:       0,
			TLSHandshakeTimeout:   0,
			ResponseHeaderTimeout: 0,
			ExpectContinueTimeout: 0,
			WriteBufferSize:       cfg.BufferSize,
			ReadBufferSize:        cfg.BufferSize,
			DialContext:           dialContextFunc(cfg, address),
			TLSClientConfig:       createTLSClientConfig(cfg),
		},
		Timeout: 0,
	}
}

func createTLSClientConfig(_ Config) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}
}

func dialContextFunc(_ Config, address string) func(ctx context.Context, network string, addr string) (net.Conn, error) {
	return func(ctx context.Context, network string, addr string) (net.Conn, error) {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return nil, err
		}
		err = conn.SetDeadline(*zeroTime)
		return conn, err
	}
}

func getAddressS3Host(cfg Config) (string, string) {
	return cfg.Address, cfg.S3Host
}

func getKeys(cfg Config) aws.Credentials {
	return aws.Credentials{
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
	}
}
