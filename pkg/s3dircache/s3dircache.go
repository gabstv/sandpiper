package s3dircache

import (
	"bytes"
	"context"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"golang.org/x/crypto/acme/autocert"
)

// C s3 cache
type C struct {
	AwsID     string
	AwsSecret string
	Region    string
	Bucket    string
	Folder    string
	s3        *s3.S3
	s3u       *s3manager.Uploader
}

func (c *C) setup() {
	if c.s3 != nil {
		return
	}
	creds := credentials.NewStaticCredentials(c.AwsID, c.AwsSecret, "")
	awss := session.New(&aws.Config{
		Credentials: creds,
		Region:      &c.Region,
	})
	c.s3 = s3.New(awss)
	c.s3u = s3manager.NewUploader(awss)
}

// Get returns a certificate data for the specified key.
// If there's no such key, Get returns ErrCacheMiss.
func (c *C) Get(ctx context.Context, key string) ([]byte, error) {
	c.setup()
	k := key
	if c.Folder != "" {
		k = path.Join(c.Folder, key)
	}
	outp, err := c.s3.GetObject(&s3.GetObjectInput{
		Bucket: &c.Bucket,
		Key:    &k,
	})
	if err != nil {
		return nil, autocert.ErrCacheMiss
	}
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, outp.Body); err != nil {
		return nil, err
	}
	outp.Body.Close()
	return buf.Bytes(), nil
}

// Put stores the data in the cache under the specified key.
// Underlying implementations may use any data storage format,
// as long as the reverse operation, Get, results in the original data.
func (c *C) Put(ctx context.Context, key string, data []byte) error {
	c.setup()
	k := key
	if c.Folder != "" {
		k = path.Join(c.Folder, key)
	}
	buf := bytes.NewBuffer(data)
	_, err := c.s3u.Upload(&s3manager.UploadInput{
		ACL:    aws.String("private"),
		Body:   aws.ReadSeekCloser(buf),
		Bucket: &c.Bucket,
		Key:    &k,
	})
	return err
}

// Delete removes a certificate data from the cache under the specified key.
// If there's no such key in the cache, Delete returns nil.
func (c *C) Delete(ctx context.Context, key string) error {
	c.setup()
	k := key
	if c.Folder != "" {
		k = path.Join(c.Folder, key)
	}
	_, err := c.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &c.Bucket,
		Key:    &k,
	})
	return err
}
