# PDF Guide to Code Implementation Mapping

## Document Reference
**Source**: A Practical Guide for Secure MCP Server Development v1.0 (February 2026)

---

## Section 6: Secure Deployment & Updates (Page 11)

### PDF Requirement: "Supply Chain Controls"

**Direct Quote from PDF (Page 11):**
> "Supply Chain Controls: Version pin and scan all dependencies at build time, use signed container images, monitor for new CVEs, and maintain AIBOMs for all builds to ensure provenance and detect tampering."

#### Key Components:
1. **Version pin** - Pin all dependencies to specific versions
2. **Scan all dependencies** - Use SCA tools during build
3. **Use signed container images** - Verify container image signatures
4. **Monitor for new CVEs** - Continuous vulnerability tracking
5. **Maintain AIBOMs** - AI Bill of Materials for provenance

---

## Code Implementation: Image Version Pinning

### PDF Guidance on Container Images
The PDF emphasizes:
- **"Version pin"** container images
- Use **"signed container images"**
- Maintain **provenance** and detect tampering

### Implementation in Codebase

#### File: `controller/pkg/evaluator/evaluator.go`

**HDN-006 Finding: Container Image Tag Validation**

```go
// HDN-006: :latest or untagged image
if w.HasLatestTag {
    findings = append(findings, Finding{
        ID:          fmt.Sprintf("HDN-006-%s", w.Name),
        Severity:    SeverityMedium,
        Category:    CategoryHardening,
        Title:       fmt.Sprintf("%s '%s' uses :latest or untagged container image", w.Kind, w.Name),
        Description: fmt.Sprintf("%s '%s/%s' has one or more containers using the :latest tag or no tag at all: %v", w.Kind, w.Namespace, w.Name, w.ImageNames),
        Impact:      "Untagged images are non-deterministic. A registry push can silently replace a running image, introducing malicious code.",
        Remediation: "Pin all container images to a specific immutable digest (sha256:...) or a versioned tag. Never use :latest in production.",
        ResourceRef: ref,
        Namespace:   w.Namespace,
    })
}
```

**Lines**: ~1203-1213 in `evaluator.go`

### Direct PDF-to-Code Alignment

| PDF Requirement | Code Implementation | Location |
|---|---|---|
| "Version pin...container images" | HDN-006 detection: Flags `:latest` tags and untagged images | `evaluator.go` line 1203 |
| **Remediation**: "Pin all container images to a specific immutable digest (sha256:...) or a versioned tag" | Finding remediation text explicitly recommends SHA256 digest pinning | `evaluator.go` line 1210 |
| **Impact**: "Untagged images are non-deterministic" | Finding impact explains the non-deterministic risk | `evaluator.go` line 1208 |

---

## Why Version Pinning Matters (Per PDF Context)

### Current Vulnerability Landscape (Page 5)

The PDF identifies **"Dynamic Tool Instability ("Rug Pulls")"**:

> "Given the lack of strict versioning for tool descriptions and the dynamic nature of loading them, there is a risk of 'rug pulls.' A previously trusted tool definition can be swapped or modified in real-time to introduce malicious behavior, bypassing initial security checks that occurred before the change."

### Application to Container Images

The same principle applies to container images:
- **Without version pinning**: An image reference `:latest` can be silently updated in the registry
- **With version pinning**: An immutable digest (sha256:...) ensures deterministic, reproducible deployments
- **Security impact**: Prevents "rug pull" scenarios where a trusted container image is replaced with malicious code

---

## Related PDF Sections

### Section 6 (Page 11) - Containerize and Harden
> "Deploy the MCP server in a minimal, hardened container. Run the container as a non-root user and drop all unnecessary container packages, dependencies, or Linux capabilities to limit residual attack surface."

**Related Code Checks**:
- HDN-001: Containers run as root
- HDN-004: Capabilities not dropped
- HDN-002: No read-only root filesystem

### Section 7 (Page 12) - Cryptographic Integrity
> "Use cryptographic signing and version pinning for all tools, dependencies, and registry manifests to ensure their integrity and prevent tampering."

**Related Code Checks**:
- HDN-010: No image signature annotation (Cosign/Sigstore)
- HDN-006: Image version pinning (covered above)

---

## Summary

**PDF Document Emphasis**: Organizations must implement **version pinning** for container images as part of their supply chain control strategy to prevent "rug pull" style attacks where previously trusted container images are silently replaced with malicious versions.

**Code Implementation**: The `HDN-006` check in `evaluator.go` (lines 1203-1213) directly enforces this PDF guidance by:
1. Detecting any use of `:latest` tags or untagged images
2. Marking this as a Medium-severity hardening violation
3. Providing explicit remediation: "Pin all container images to a specific immutable digest (sha256:...) or a versioned tag. Never use :latest in production."

This aligns perfectly with the PDF's supply chain control requirements for secure MCP server deployment.
