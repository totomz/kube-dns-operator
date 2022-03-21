# kube-dns-operator

Create a `DnsRecord` resource, and the operator will create a correspondent DNS record in your dns server
```yaml
apiVersion: net.beekube.cloud/v1alpha1
kind: DnsRecord
metadata:
  name: www-blog-alpha
  namespace: ziotest
spec:
  # Create a record in AWS Route53
  Route53Records:
    awsSecrets:
      secretNamespace: ziotest 
      secretName: my-ideas-aws-dns
      accessKeyIDKey: access-key-id
      secretAccessKeyKey: secret-access-key
    zoneId: "<ZoneId>"
    # Any valid .Type, like CNAME, A, TXT
    type: "CNAME" 
    # The FQDN 
    name: "www-demo394.my-ideas.it"
    # The resource records (check the AWS docs! CNAME allows only 1 element)
    resourceRecords:
      - kubeapp.dc-pilotto.my-ideas.it
    comment: "This is cio"
    ttl: 300
```

## Supported DNS

### AWS Route53
Create a secret with an AIM user that have access to Route53
```
kubectl create secret generic my-ideas-aws-dns \
  --from-literal=secret-access-key="<secret key>" \
  --from-literal=access-key-id="<AKIAzzzzz>" \
  --namespace="default"
```