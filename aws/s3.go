package aws

import (
	"fmt"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/errwrap"

	"github.com/yuuki1/droot/log"
)

const uploadPartSize = 64 * 1024 * 1024 // 64MB part size
const downloadPartSize = uploadPartSize

type S3Client struct {
	svc s3iface.S3API
}

func NewS3Client() *S3Client {
	var svc s3iface.S3API
	if log.IsDebug {
		svc = s3.New(session.New(), aws.NewConfig().WithLogLevel(aws.LogDebug))
	} else {
		svc = s3.New(session.New())
	}
	return &S3Client{svc: svc}
}

func (clt *S3Client) ExistsBucket(bucket string) (bool, error) {
	_, err := clt.svc.ListObjects(&s3.ListObjectsInput{
		Bucket: &bucket,
	})
	if err != nil {
		return false, errwrap.Wrapf(fmt.Sprintf("Failed to list s3 objects %s: {{err}}", bucket), err)
	}
	return true, nil
}

func (clt *S3Client) Upload(s3Url *url.URL, reader io.Reader) (string, error) {
	bucket, object := s3Url.Host, s3Url.Path

	ok, err := clt.ExistsBucket(bucket)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("No such bucket: %s", bucket)
	}

	uploader := s3manager.NewUploaderWithClient(clt.svc)
	upOutput, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucket,
		Key:    &object,
		Body:   reader,
	}, func(u *s3manager.Uploader) {
		u.PartSize = uploadPartSize
	})
	if err != nil {
		return "", errwrap.Wrapf("Failed to upload s3: {{err}}", err)
	}

	return upOutput.Location, nil
}

func (clt *S3Client) Download(s3Url *url.URL, writer io.WriterAt) (int64, error) {
	bucket, object := s3Url.Host, s3Url.Path

	ok, err := clt.ExistsBucket(bucket)
	if err != nil {
		return -1, err
	}
	if !ok {
		return -1, fmt.Errorf("No such bucket: %s", bucket)
	}

	downloader := s3manager.NewDownloaderWithClient(clt.svc)
	nBytes, err := downloader.Download(writer, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &object,
	}, func(d *s3manager.Downloader) {
		d.PartSize = downloadPartSize
	})
	if err != nil {
		return -1, errwrap.Wrapf("Failed to download s3: {{err}}", err)
	}

	return nBytes, nil
}
