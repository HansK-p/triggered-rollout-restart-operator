# triggered-rollout-restart-operator
Operator responsible for gracefully restart pods when depended resources change

This oprator is based on the [Operator SDK](https://github.com/operator-framework/operator-sdk) and is used on order to make sure that pods are gracefully restarted after secrets have been updated by [Kubernetes Replicator](https://github.com/mittwald/kubernetes-replicator) or similar software.

The idea is to create a CRD in the same namespace as the secret and the pods to be restarted with a content like:
<pre>
apiVersion: reload-restart-trigger.k8s.faith/v1alpha1
kind: ResourceReloadRestartTrigger
metadata:
  name: nginx
  namespace: website
spec:
  triggers:
  - name: nginx-config
    kind: ConfigMap
  - name: cert-tls
    kind: Secret
  targets:
  - kind: Deployment
    name: nginx
</pre>

A docker image will be published at irregular intervals as k8sfaith/triggered-rollout-restart-operator.

Suggested RBAC rules when deploying the Operator are. See the deployment folder for files needed for a deployment.
