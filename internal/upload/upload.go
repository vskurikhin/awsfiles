package upload

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/vskurikhin/awsfiles/internal/config"
)

const (
	partMiBs = 5
)

var zeroTime = new(time.Time)

func Upload(cfg config.Config) {
	now := time.Now().UnixNano()
	source := rand.NewSource(now)
	r := rand.New(source)
	lr := io.LimitReader(r, int64(cfg.Size))
	h := md5.New()
	md5r := md5Reader{h, lr}
	ctx := context.Background()
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
		slog.Info("Upload", "client", "prepared")
	}
	_, err := clientUploaderUpload(ctx, cfg, client, &md5r)
	if err != nil {
		slog.Error("Upload failed", "err", err)
		return
	}
	err = s3.
		NewObjectExistsWaiter(client).
		Wait(
			ctx,
			&s3.HeadObjectInput{
				Bucket: aws.String(cfg.Bucket),
				Key:    aws.String(cfg.Key),
			}, time.Minute,
		)
	slog.Info("Upload", "result", fmt.Sprintf("total write bytes: %d", cfg.Size))
	value := hex.EncodeToString(md5r.H.Sum(nil))
	slog.Info("Upload", "result", fmt.Sprintf("md5sum: %s", value))
	if err != nil {
		log.Printf("Failed attempt to wait for object %s to exist.\n", cfg.Key)
	}
}

var _ io.Reader = (*md5Reader)(nil)

type md5Reader struct {
	H hash.Hash
	R io.Reader // underlying reader
}

func (m *md5Reader) Read(p []byte) (n int, err error) {
	n, err = m.R.Read(p)
	m.H.Write(p[:n])
	return n, err
}

func clientUploaderUpload(ctx context.Context, cfg config.Config, client *s3.Client, reader io.Reader) (*manager.UploadOutput, error) {
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
	})
	return uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(cfg.Key),
		Body:   reader,
	})
}

func getCustomResolver(s3host string) aws.EndpointResolverWithOptionsFunc {
	return func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if region == "" {
			region = "auto"
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
