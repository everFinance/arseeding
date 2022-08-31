package rawdb

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/everFinance/arseeding/schema"
	"net"
	"net/url"
	"strings"
)

const (
	ForeverLandEndpoint = "https://endpoint.4everland.co"
)

type S3DB struct {
	downloader   s3manager.Downloader
	uploader     s3manager.Uploader
	s3Api        s3iface.S3API
	bucketPrefix string
}

func NewS3DB(accKey, secretKey, region, bktPrefix, endpoint string) (*S3DB, error) {
	mySession := session.Must(session.NewSession())
	cred := credentials.NewStaticCredentials(accKey, secretKey, "")
	cfgs := aws.NewConfig().WithRegion(region).WithCredentials(cred)
	if endpoint != "" {
		cfgs.WithEndpoint(endpoint) // inject endpoint
		// if endpoint is an IP address, use path-style addressing.
		if u, err := url.Parse(endpoint); err == nil {
			if net.ParseIP(u.Hostname()) != nil {
				cfgs.S3ForcePathStyle = aws.Bool(true)
			}
		}
	}
	s3Api := s3.New(mySession, cfgs)
	err := createS3Bucket(s3Api, bktPrefix)
	if err != nil {
		return nil, err
	}

	log.Info("run with s3 success")
	return &S3DB{
		downloader: s3manager.Downloader{
			S3: s3Api,
		},
		uploader: s3manager.Uploader{
			S3: s3Api,
		},
		s3Api:        s3Api,
		bucketPrefix: bktPrefix,
	}, nil
}

func (s *S3DB) Put(bucket, key string, value []byte) (err error) {
	bkt := getS3Bucket(s.bucketPrefix, bucket)
	uploadInfo := &s3manager.UploadInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
		Body:   bytes.NewReader(value),
	}
	_, err = s.uploader.Upload(uploadInfo)
	return
}

func (s *S3DB) Get(bucket, key string) (data []byte, err error) {
	bkt := getS3Bucket(s.bucketPrefix, bucket)
	downloadInfo := &s3.GetObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	}
	buf := aws.NewWriteAtBuffer([]byte{})
	n, err := s.downloader.Download(buf, downloadInfo)
	if n == 0 {
		return nil, schema.ErrNotExist
	}
	data = buf.Bytes()
	return
}

func (s *S3DB) GetAllKey(bucket string) (keys []string, err error) {
	bkt := getS3Bucket(s.bucketPrefix, bucket)
	resp, err := s.s3Api.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bkt)})
	if err != nil {
		return
	}
	keys = make([]string, 0)
	for _, item := range resp.Contents {
		keys = append(keys, *item.Key)
	}
	if len(keys) == 0 {
		err = schema.ErrNotExist
	}
	return
}

func (s *S3DB) Delete(bucket, key string) (err error) {
	bkt := getS3Bucket(s.bucketPrefix, bucket)
	_, err = s.s3Api.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(bkt), Key: aws.String(key)})
	return
}

func (s *S3DB) Close() (err error) {
	return
}

func createS3Bucket(svc s3iface.S3API, prefix string) error {
	bucketNames := []string{
		schema.ChunkBucket,
		schema.TxDataEndOffSetBucket,
		schema.TxMetaBucket,
		schema.ConstantsBucket,
		schema.TaskIdPendingPoolBucket,
		schema.TaskBucket,
		schema.BundleItemBinary,
		schema.BundleItemMeta,
		schema.BundleWaitParseArIdBucket,
		schema.BundleArIdToItemIdsBucket,
	}
	for _, bucketName := range bucketNames {
		s3Bkt := getS3Bucket(prefix, bucketName) // s3 bucket name only accept lower case
		_, err := svc.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(s3Bkt)})
		if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			return err
		}
	}
	return nil
}

func getS3Bucket(prefix, bktName string) string {
	return strings.ToLower(prefix + "-" + bktName)
}
