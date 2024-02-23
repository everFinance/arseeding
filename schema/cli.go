package schema

type Config struct {
	RollupKeyPath  string `yaml:"rollupKeyPath"`
	Pay            string `yaml:"pay"`
	ArNode         string `yaml:"arNode"`
	Mysql          string `yaml:"mysql"`
	Port           string `yaml:"port"`
	Manifest       bool   `yaml:"manifest"`
	NoFee          bool   `yaml:"noFee"`
	BundleInterval int    `yaml:"bundleInterval"`
	Tags           string `yaml:"tags"`

	BoltDir   string    `yaml:"boltDir"`
	S3KV      S3KV      `yaml:"s3KV"`
	AliyunKV  AliyunKV  `yaml:"aliyunKV"`
	MongoDBKV MongoDBKV `yaml:"mongoDBKV"`

	Kafka Kafka `yaml:"kafka"`
}

type S3KV struct {
	UseS3     bool   `yaml:"useS3"`
	User4Ever bool   `yaml:"user4Ever"`
	AccKey    string `yaml:"accKey"`
	SecretKey string `yaml:"secretKey"`
	Prefix    string `yaml:"prefix"`
	Region    string `yaml:"region"`
	Endpoint  string `yaml:"endpoint"`
}

type AliyunKV struct {
	UseAliyun bool   `yaml:"useAliyun"`
	Endpoint  string `yaml:"endpoint"`
	AccKey    string `yaml:"accKey"`
	SecretKey string `yaml:"secretKey"`
	Prefix    string `yaml:"prefix"`
}

type MongoDBKV struct {
	UseMongoDB bool   `yaml:"useMongoDB"`
	Uri        string `yaml:"uri"`
}

type Kafka struct {
	Start bool   `yaml:"start"`
	Uri   string `yaml:"uri"`
}
