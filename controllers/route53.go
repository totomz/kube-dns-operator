package controllers

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/totomz/kube-dns-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *DnsRecordReconciler) ReconcileRoute53(ctx context.Context, ns string, record v1alpha1.Route53Record, status v1alpha1.DnsRecordStatus) error {
	logger := log.FromContext(ctx)
	logger.Info("upsert resource for ", "name", record.Name)

	secretNs := record.AwsSecrets.SecretNamespace
	if secretNs == "" {
		secretNs = ns
	}

	accessId, errSecret := r.GetSecret(ctx, secretNs, record.AwsSecrets.SecretName, record.AwsSecrets.AccessKeyIDKey)
	if errSecret != nil {
		logger.Error(errSecret, "can't get the aws access key")
		return errSecret
	}
	accessSecret, errSecret := r.GetSecret(ctx, secretNs, record.AwsSecrets.SecretName, record.AwsSecrets.SecretAccessKeyKey)
	if errSecret != nil {
		logger.Error(errSecret, "can't get the aws secret")
		return errSecret
	}

	return UpsertRoute53(ctx, record, accessId, accessSecret)

}

func GetChangeStatus53(ctx context.Context, changeId, accessId, accessSecret string) (*route53.GetChangeOutput, error) {
	logger := log.FromContext(ctx)
	cfg, errConfig := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessId, accessSecret, "")))
	if errConfig != nil {
		logger.Error(errConfig, "can't get aws configuration")
		return nil, errConfig
	}
	svc := route53.NewFromConfig(cfg)

	return svc.GetChange(ctx, &route53.GetChangeInput{Id: aws.String(changeId)})
}

func UpsertRoute53(ctx context.Context, record v1alpha1.Route53Record, accessId, accessSecret string) error {
	logger := log.FromContext(ctx)
	cfg, errConfig := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessId, accessSecret, "")))
	if errConfig != nil {
		logger.Error(errConfig, "can't get aws configuration")
		return errConfig
	}

	var rr []types.ResourceRecord
	for _, r := range record.ResourceRecords {
		rr = append(rr, types.ResourceRecord{Value: aws.String(r)})
	}

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(record.ZoneId),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: "UPSERT",
					ResourceRecordSet: &types.ResourceRecordSet{
						TTL:             aws.Int64(300),
						Name:            aws.String(record.Name),
						Type:            types.RRType(record.Type),
						ResourceRecords: rr,
					},
				},
			},
			Comment: aws.String(record.Comment),
		},
	}

	svc := route53.NewFromConfig(cfg)
	output, errUpsert := svc.ChangeResourceRecordSets(ctx, params)
	if errUpsert != nil {
		logger.Error(errUpsert, "failed aws api call :(")
		return errUpsert
	}

	logger.Info("change committed", "changeId", output.ChangeInfo.Id)

	return nil
}
