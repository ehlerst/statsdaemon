package route

import (
	"github.com/raintank/schema"
	"github.com/ugorji/go/codec"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/aws"
	"errors"
	"os"
)

type GAASPublisher interface {
	Publish(metrics []schema.MetricData, stream string, partitionStrategy PartitionStrategy) error
}

type KinesisPublisher struct {
	KinesisClient *kinesis.Kinesis
}

var (
	ErrInvalidMetrics = errors.New("Invalid metrics!")
)

type PartitionStrategy string

const (
	BySeries PartitionStrategy = "bySeries"
	ByOrg PartitionStrategy = "byOrg"
)

func (KP KinesisPublisher) Publish(metrics []schema.MetricData, stream string, partitionStrategy PartitionStrategy) error {
	if len(metrics) == 0 {
		return ErrInvalidMetrics
	}

	// assumes all metrics correspond to one series
	// TODO add logic to support multiple series
	pKey := string(metrics[0].KeyBySeries(nil))

	if partitionStrategy == ByOrg {
		pKey = string(metrics[0].KeyByOrgId(nil))
	}
	// TODO break up data in batches less than 1MB
	msgpkMetrics := make([]byte, 0, 64)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&msgpkMetrics, handle)
	err := encoder.Encode(metrics)

	if err != nil {
		return err
	}

	if KP.KinesisClient == nil {
		sess, err := session.NewSession()

		if err != nil {
			return err
		}

		KP.KinesisClient = kinesis.New(sess, aws.NewConfig().WithRegion(os.Getenv("AWS_REGION")))
	}

	svc := KP.KinesisClient

	// TODO add retry mechanism
	putInput := &kinesis.PutRecordInput{
		Data:         msgpkMetrics,
		PartitionKey: &pKey,
		StreamName:   &stream,
	}

	_ , err = svc.PutRecord(putInput)

	return err
}