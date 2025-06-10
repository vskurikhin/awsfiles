package object

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/vskurikhin/awsfiles/internal/config"
)

var zeroTime = new(time.Time)

func GetObject(cfg config.Config) {
	keys := getKeys(cfg)
	httpclient := getHTTPClient(cfg)
	customResolver := getCustomResolver(cfg.S3Host)
	creeds := credentials.NewStaticCredentialsProvider(keys.AccessKeyID, keys.SecretAccessKey, "")
	client := s3.NewFromConfig(aws.Config{
		Credentials:                 creeds,
		HTTPClient:                  httpclient,
		EndpointResolverWithOptions: customResolver,
	})
	if cfg.Verbose {
		slog.Info("Getting object", "client", "prepared")
	}
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
		if cfg.Verbose {
			fmt.Printf("read bytes: %d, total read bytes %d\r", i, bytesWritten)
		}
	}
	fmt.Println()
	slog.Info("Get object", "result", fmt.Sprintf("total read bytes: %d", bytesWritten))
	value := hex.EncodeToString(hasher.Sum(nil))
	slog.Info("Get object", "result", fmt.Sprintf("md5sum: %s", value))
}

func clientGetObject(cfg config.Config, client *s3.Client) (*s3.GetObjectOutput, error) {
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

func getHTTPClient(cfg config.Config) *http.Client {
	var tlsCfg *tls.Config
	if cfg.Ssl() {
		tlsCfg = createTLSClientConfig(cfg)
	}
	transport := &http.Transport{
		DisableKeepAlives:     false,
		IdleConnTimeout:       0,
		TLSHandshakeTimeout:   0,
		ResponseHeaderTimeout: 0,
		ExpectContinueTimeout: 0,
		WriteBufferSize:       cfg.BufferSize,
		ReadBufferSize:        cfg.BufferSize,
		TLSClientConfig:       tlsCfg,
		DialContext:           dialContextFunc(cfg, tlsCfg),
	}
	return &http.Client{
		Transport: transport,
		Timeout:   0,
	}
}

func createTLSClientConfig(cfg config.Config) *tls.Config {
	certs := x509.NewCertPool()

	if cfg.CAFile != "" {
		pemData, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			slog.Error("Read certificate failed", "err", err)
		}
		certs.AppendCertsFromPEM(pemData)
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		RootCAs:            certs,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		NextProtos: []string{
			"http/1.1",
			"h2",
		},
	}
}

func dialContextFunc(cfg config.Config, tlsCfg *tls.Config) func(ctx context.Context, network string, addr string) (net.Conn, error) {
	if cfg.Ssl() {
		return func(ctx context.Context, network string, addr string) (net.Conn, error) {
			tlsCfg.ServerName = cfg.ServerName
			conn, err := tls.Dial("tcp", cfg.Address, tlsCfg)
			if err != nil {
				return nil, err
			}
			if cfg.Debug {
				slog.Debug(
					"connected",
					"ServerName", conn.ConnectionState().ServerName,
					"NegotiatedProtocol", conn.ConnectionState().NegotiatedProtocol,
					"HandshakeComplete", conn.ConnectionState().HandshakeComplete)
			}
			err = conn.SetDeadline(*zeroTime)
			return conn, err
		}
	}
	return func(ctx context.Context, network string, addr string) (net.Conn, error) {
		conn, err := net.Dial("tcp", cfg.Address)
		if err != nil {
			fmt.Println("dial connection failed", "err", err)
			return nil, err
		}
		err = conn.SetDeadline(*zeroTime)
		return conn, err
	}
}

func getKeys(cfg config.Config) aws.Credentials {
	return aws.Credentials{
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
	}
}
