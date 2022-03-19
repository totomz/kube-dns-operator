package controllers

import (
	"context"
	_ "github.com/joho/godotenv/autoload"
	"github.com/totomz/kube-dns-operator/api/v1alpha1"
	"os"
	"testing"
)

func TestUpsertCNAMERoute53(t *testing.T) {
	t.Skip("Integration test - manual ony, no resource cleanup")
	ctx := context.Background()
	record := v1alpha1.Route53Record{
		AwsSecrets: v1alpha1.AwsSecret{},
		Name:       "uuihfsdso.kube-operator-test.test.my-ideas.it",
		Type:       "CNAME",
		ZoneId:     os.Getenv("ZONE_ID"),
		ResourceRecords: []string{
			"kubeapp.dc-pilotto.my-ideas.it",
		},
		Ttl:     300,
		Comment: "INTEGTEST -- kube-dns-operator",
	}

	err := UpsertRoute53(ctx, record, "UPSERT", os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_ACCESS_SECRET"))
	if err != nil {
		t.Error(err)
	}
}

func TestUpsertARoute53(t *testing.T) {
	t.Skip("Integration test - manual ony, no resource cleanup")
	ctx := context.Background()
	record := v1alpha1.Route53Record{
		AwsSecrets: v1alpha1.AwsSecret{},
		Name:       "uuihfsdso-a.kube-operator-test.test.my-ideas.it",
		Type:       "A",
		ZoneId:     os.Getenv("ZONE_ID"),
		ResourceRecords: []string{
			"151.100.152.223",
		},
		Ttl:     300,
		Comment: "INTEGTEST -- kube-dns-operator",
	}

	err := UpsertRoute53(ctx, record, "UPSERT", os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_ACCESS_SECRET"))
	if err != nil {
		t.Error(err)
	}

}
