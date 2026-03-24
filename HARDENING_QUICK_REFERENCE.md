# Hardening Checks - Quick Reference

## Deploy Sample Resources

### GovernanceEvaluation
```bash
kubectl apply -f deploy/samples/governance-evaluation.yaml
```

Verify:
```bash
kubectl get governanceevaluation enterprise-governance-eval -o yaml | grep -A 20 "^status:"
```

### Test Workloads
```bash
kubectl apply -f deploy/samples/test-hardening-workloads.yaml
```

Verify discovery:
```bash
kubectl logs -n mcp-governance -l app.kubernetes.io/component=controller | grep "Found:"
```

Expected output:
```
[discovery] Found: ... 25 workloads, 2 networkpolicies
```

## Run Tests

### Unit Tests (All Hardening Checks)
```bash
cd controller
go test -v ./pkg/evaluator -run TestCheckHardenedDeployment
```

**Expected**: All 13 tests PASS ✅

### Run Specific Test
```bash
go test -v ./pkg/evaluator -run TestCheckHardenedDeployment_ContainerRunAsRoot
```

### Run with Coverage
```bash
go test -coverage ./pkg/evaluator -run TestCheckHardenedDeployment
```

### Benchmark
```bash
go test -bench=BenchmarkCheckHardenedDeployment ./pkg/evaluator -benchtime=3s -benchmem
```

**Expected Performance**: ~88 microseconds/op

## Monitor GovernanceEvaluation

### Watch for Updates
```bash
kubectl get governanceevaluation enterprise-governance-eval -w
```

### Get Full Status
```bash
kubectl get governanceevaluation enterprise-governance-eval -o jsonpath='{.status}' | jq .
```

### Count Findings by Severity
```bash
kubectl get governanceevaluation enterprise-governance-eval -o jsonpath='{.status.findings[*].severity}' | tr ' ' '\n' | sort | uniq -c
```

### List Finding IDs
```bash
kubectl get governanceevaluation enterprise-governance-eval -o jsonpath='{.status.findings[*].id}' | tr ' ' '\n' | sort
```

## HDN Findings Breakdown

| Code | Check | Severity | Remediation |
|------|-------|----------|-------------|
| HDN-000 | No workloads discovered | High | Deploy workloads as Deployments/StatefulSets |
| HDN-001 | Container runs as root | Critical | Add `runAsNonRoot: true` to securityContext |
| HDN-002 | Root filesystem writable | High | Set `readOnlyRootFilesystem: true` |
| HDN-003 | Privilege escalation allowed | High | Set `allowPrivilegeEscalation: false` |
| HDN-004 | Capabilities not dropped | Medium | Add `capabilities.drop: [ALL]` |
| HDN-005 | No seccomp profile | Medium | Set `seccompProfile.type: RuntimeDefault` |
| HDN-006 | :latest or untagged image | Medium | Use specific version tags (e.g., `nginx:1.27-alpine`) |
| HDN-007 | No NetworkPolicy in namespace | Critical | Create NetworkPolicy with ingress/egress rules |
| HDN-008 | Plaintext secrets in env | High | Use Vault or External Secrets Operator |
| HDN-009 | No Vault/ESO injection | Medium | Add Vault/ESO annotations to pod spec |
| HDN-010 | No image signature verification | Medium | Add `imageSignatureVerified: "true"` annotation |

## Troubleshooting

### Controller Not Detecting Workloads
```bash
# Check discovery logs
kubectl logs -n mcp-governance -l app.kubernetes.io/component=controller --since=1m | grep -i "workload\|discover"

# Verify RBAC permissions
kubectl get clusterrole/mcp-governance-controller-role -o yaml | grep -A 10 "deployments\|statefulsets"
```

### GovernanceEvaluation Not Updating
```bash
# Ensure sample resource exists
kubectl get governanceevaluation enterprise-governance-eval

# Check policy reference
kubectl get governanceevaluation enterprise-governance-eval -o jsonpath='{.spec.policyRef}'

# Should return: enterprise-mcp-policy
```

### No HDN Findings Generated
```bash
# Verify policy has hardening enabled
kubectl get mcpgovernancepolicy enterprise-mcp-policy -o jsonpath='{.spec.requireHardenedDeployment}'

# Should return: true

# Check if test namespace is excluded
kubectl get mcpgovernancepolicy enterprise-mcp-policy -o jsonpath='{.spec.excludeNamespaces}' | grep test-hardening

# Should return nothing (not excluded)
```

## Performance Tuning

### Scan Interval
```bash
kubectl patch mcpgovernancepolicy enterprise-mcp-policy \
  --type merge \
  -p '{"spec":{"scanInterval":"10m"}}'
```

### Disable Hardening Checks (If Needed)
```bash
kubectl patch mcpgovernancepolicy enterprise-mcp-policy \
  --type merge \
  -p '{"spec":{"requireHardenedDeployment":false}}'
```

## Example Hardening Remediation

### Transform Vulnerable Deployment to Hardened

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        vault.hashicorp.com/agent-inject: "true"
        vault.hashicorp.com/role: "app-role"
        imageSignatureVerified: "true"
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 3000
        fsGroup: 2000
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: app
        image: myapp:1.2.3  # ← Specific version, not :latest
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        env:
        - name: CONFIG_PATH
          value: /etc/app
        # ← No plaintext secrets, use Vault instead
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: app-config
          mountPath: /etc/app
          readOnly: true
      volumes:
      - name: tmp
        emptyDir: {}
      - name: app-config
        configMap:
          name: app-config
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: my-app-policy
spec:
  podSelector:
    matchLabels:
      app: my-app
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          role: frontend
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: UDP
      port: 53  # DNS
  - to:
    - podSelector:
        matchLabels:
          role: backend
    ports:
    - protocol: TCP
      port: 5432  # Database
```

## API Endpoints

### Get Hardening Findings
```bash
curl http://localhost:8090/api/governance/findings | jq '.findings[] | select(.category == "Hardening")'
```

### Get Score Breakdown
```bash
curl http://localhost:8090/api/governance/score | jq '.categories[] | select(.category == "Hardened Deployment")'
```

### Health Check
```bash
curl http://localhost:8090/api/health | jq .
```

## Related Files

- **Implementation**: `controller/pkg/evaluator/evaluator.go` (lines 1120-1260)
- **Discovery**: `controller/pkg/discovery/discovery.go` (lines 802-990)
- **Tests**: `controller/pkg/evaluator/evaluator_hardening_test.go`
- **Test Workloads**: `deploy/samples/test-hardening-workloads.yaml`
- **Sample Evaluation**: `deploy/samples/governance-evaluation.yaml`
- **Documentation**: `TIER1_HARDENING_COMPLETION.md`

