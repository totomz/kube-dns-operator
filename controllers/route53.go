package controllers

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/totomz/kube-dns-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

var (
	ActionUpsert = "UPSERT"
	ActionDelete = "DELETE"
)

func (r *DnsRecordReconciler) getAwsCred(ctx context.Context, secretNs, secretName, accessKeyIdKey, accessSecretKeyKey string) (string, string, error) {
	logger := log.FromContext(ctx)
	accessId, errSecret := r.GetSecret(ctx, secretNs, secretName, accessKeyIdKey)
	if errSecret != nil {
		logger.Error(errSecret, "can't get the aws access key")
		return "", "", errSecret
	}
	accessSecret, errSecret := r.GetSecret(ctx, secretNs, secretName, accessSecretKeyKey)
	if errSecret != nil {
		logger.Error(errSecret, "can't get the aws secret")
		return "", "", errSecret
	}

	return accessId, accessSecret, nil
}

func (r *DnsRecordReconciler) ReconcileRoute53(ctx context.Context, ns string, record v1alpha1.Route53Record) error {
	secretNs := record.AwsSecrets.SecretNamespace
	if secretNs == "" {
		secretNs = ns
	}

	accessId, accessSecret, err := r.getAwsCred(ctx, secretNs, record.AwsSecrets.SecretName, record.AwsSecrets.AccessKeyIDKey, record.AwsSecrets.SecretAccessKeyKey)
	if err != nil {
		return err
	}

	return UpsertRoute53(ctx, record, ActionUpsert, accessId, accessSecret)
}

func (r *DnsRecordReconciler) FinalizeAwsRoute53(ctx context.Context, ns string, record v1alpha1.Route53Record) error {
	secretNs := record.AwsSecrets.SecretNamespace
	if secretNs == "" {
		secretNs = ns
	}

	accessId, accessSecret, err := r.getAwsCred(ctx, secretNs, record.AwsSecrets.SecretName, record.AwsSecrets.AccessKeyIDKey, record.AwsSecrets.SecretAccessKeyKey)
	if err != nil {
		return err
	}

	return UpsertRoute53(ctx, record, ActionDelete, accessId, accessSecret)
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

func UpsertRoute53(ctx context.Context, record v1alpha1.Route53Record, action, accessId, accessSecret string) error {
	logger := log.FromContext(ctx)
	cfg, errConfig := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessId, accessSecret, "")),
		config.WithRegion("eu-west-1"))
	if errConfig != nil {
		logger.Error(errConfig, "can't get aws configuration")
		return errConfig
	}

	logger.Info(fmt.Sprintf("%s dns record", action))

	var rr []types.ResourceRecord
	for _, r := range record.ResourceRecords {
		rr = append(rr, types.ResourceRecord{Value: aws.String(r)})
	}

	params := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(record.ZoneId),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeAction(action),
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
		if action == "DELETE" && strings.Contains(errUpsert.Error(), "StatusCode: 400") {
			logger.Error(errUpsert, "Record not found? Considering the reconcilitaion completed")
			return nil
		}

		logger.Error(errUpsert, "failed aws api call :(")
		return errUpsert
	}

	logger.Info("change committed", "changeId", output.ChangeInfo.Id)

	return nil
}
