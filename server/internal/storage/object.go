package storage

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
	"pregnancy-tracker/server/internal/config"
)

type Client struct {
	cfg    *config.Config
	cos    *cos.Client
	prefix string
}

func New(cfg *config.Config) (*Client, error) {
	c := &Client{cfg: cfg, prefix: cfg.COSPathPrefix}
	if cfg.COSBucketURL != "" && cfg.COSSecretID != "" && cfg.COSSecretKey.String() != "" {
		u, err := url.Parse(cfg.COSBucketURL)
		if err != nil {
			return nil, err
		}
		b := &cos.BaseURL{BucketURL: u}
		co := cos.NewClient(b, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  cfg.COSSecretID,
				SecretKey: cfg.COSSecretKey.String(),
			},
		})
		c.cos = co
		return c, nil
	}
	if err := os.MkdirAll(cfg.LocalUploadDir, 0o755); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) Put(ctx context.Context, key string, r io.Reader, contentLength int64, contentType string) (publicURL string, err error) {
	key = strings.TrimLeft(key, "/")
	if c.prefix != "" {
		key = c.prefix + "/" + key
	}
	if c.cos != nil {
		_, err := c.cos.Object.Put(ctx, key, r, &cos.ObjectPutOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
				ContentType:   contentType,
				ContentLength: contentLength,
			},
		})
		if err != nil {
			return "", err
		}
		base := strings.TrimRight(c.cfg.COSBucketURL, "/")
		return base + "/" + key, nil
	}
	full := filepath.Join(c.cfg.LocalUploadDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(full)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	rel := "/files/" + key
	return c.cfg.PublicBaseURL + rel, nil
}

func (c *Client) LocalPathForKey(key string) string {
	key = strings.TrimLeft(key, "/")
	if c.prefix != "" {
		key = c.prefix + "/" + key
	}
	return filepath.Join(c.cfg.LocalUploadDir, filepath.FromSlash(key))
}
