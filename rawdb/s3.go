package rawdb

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/everFinance/arseeding/schema"
	"io"
	"net"
	"net/url"
	"os"
	"reflect"
	"strings"
)

const (
	ForeverLandEndpoint = "https://endpoint.4everland.co"
	S3Type              = "s3"
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

func (s *S3DB) Type() string {
	return S3Type
}
func (s *S3DB) Put(bucket, key string, value interface{}) (err error) {
	bkt := getS3Bucket(s.bucketPrefix, bucket)

	uploadInfo := &s3manager.UploadInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	}
	if _, ok := value.([]byte); ok {
		uploadInfo.Body = bytes.NewReader(value.([]byte))
	} else if _, ok := value.(io.Reader); ok {
		uploadInfo.Body = value.(io.Reader)
	} else {
		return fmt.Errorf("unknown data type: %s, db: s3 db", reflect.TypeOf(value))
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

func (s *S3DB) GetStream(bucket, key string) (data *os.File, err error) {
	bkt := getS3Bucket(s.bucketPrefix, bucket)
	downloadInfo := &s3.GetObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	}
	data, err = os.CreateTemp("./tmpFile", "s3-")
	if err != nil {
		return
	}

	n, err := s.downloader.Download(data, downloadInfo)
	if n == 0 { // if key not exist, need delete temp file
		data.Close()
		os.Remove(data.Name())
		return nil, schema.ErrNotExist
	}
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

func (s *S3DB) Exist(bucket, key string) bool {
	bkt := getS3Bucket(s.bucketPrefix, bucket)
	_, err := s.s3Api.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	})
	return err == nil
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
		schema.StatisticBucket,
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
