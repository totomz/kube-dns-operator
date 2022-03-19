/*
Copyright 2022 Tommaso Doninelli.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
RUN  make generate manifests to update the CRD manifests
*/

// AwsSecret holds an AWS IAM AccessKey and SecretAccessKey
type AwsSecret struct {
	// SecretName Name of the secret holding the AWS credentials
	SecretName string `json:"secretName"`
	// SecretNamespace The namespace containing the secret
	// Leave it empty to use the operator namespace
	SecretNamespace string `json:"secretNamespace"`
	// AccessKeyIDKey The key that holds the AWS Access Key ID within the secret
	AccessKeyIDKey string `json:"accessKeyIDKey"`
	// SecretAccessKeyKey The key that holds the AWS Secret Access Key within the secret
	SecretAccessKeyKey string `json:"secretAccessKeyKey"`
}

type Route53Record struct {
	// IAM Access Key to use to interact with AWS
	AwsSecrets AwsSecret `json:"awsSecrets"`
	// Name Fully Qualified Domain Name
	Name string `json:"name"`
	// Type One of CNAME, A
	Type string `json:"type"`
	// ZoneId AWS Route53 ZoneID
	ZoneId string `json:"zoneId"`
	// ResourceRecords List of DNS target
	ResourceRecords []string `json:"resourceRecords"`
	// Ttl time To live in seconds
	Ttl int64 `json:"ttl"`
	// Comment optional comment
	Comment string `json:"comment"`
}

// DnsRecordSpec defines the desired state of DnsRecord
type DnsRecordSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	Route53Records Route53Record `json:"Route53Records"`
}

// DnsRecordStatus defines the observed state of DnsRecord
type DnsRecordStatus struct {
	Status     string             `json:"status"`
	ChangeId   string             `json:"changeId"`
	Conditions []metav1.Condition `json:"conditions"`
}

// DnsRecord is the Schema for the dnsrecords API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type DnsRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DnsRecordSpec   `json:"spec,omitempty"`
	Status DnsRecordStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DnsRecordList contains a list of DnsRecord
type DnsRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DnsRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DnsRecord{}, &DnsRecordList{})
}
