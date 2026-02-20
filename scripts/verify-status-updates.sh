#!/bin/bash
# verify-status-updates.sh
# Verifies that the governance controller is successfully patching catalog status fields

set -e

NAMESPACE="${NAMESPACE:-agentregistry}"
TIMEOUT="${TIMEOUT:-30}"

echo "🔍 Governance Controller Status Update Verification"
echo "=================================================="
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl."
    exit 1
fi

# Check if there are any MCPServerCatalog resources
echo "📋 Checking for MCPServerCatalog resources in namespace: $NAMESPACE"
CATALOG_COUNT=$(kubectl get mcpservercatalog -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

if [ "$CATALOG_COUNT" -eq 0 ]; then
    echo "⚠️  No MCPServerCatalog resources found in namespace $NAMESPACE"
    echo ""
    echo "Available namespaces with MCPServerCatalog:"
    kubectl get mcpservercatalog -A | head -5
    exit 1
fi

echo "✅ Found $CATALOG_COUNT MCPServerCatalog resources"
echo ""

# Check controller deployment
echo "📊 Checking Governance Controller Status"
CONTROLLER_NS="${CONTROLLER_NS:-default}"
CONTROLLER_NAME="mcp-governance-controller"

POD_STATUS=$(kubectl get pods -n "$CONTROLLER_NS" -l app.kubernetes.io/name=mcp-governance \
    -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "NOT_FOUND")

if [ "$POD_STATUS" != "Running" ]; then
    echo "⚠️  Controller pod not running. Status: $POD_STATUS"
    echo ""
    echo "Check controller logs:"
    echo "  kubectl logs -n $CONTROLLER_NS -l app.kubernetes.io/name=mcp-governance --tail=20"
    exit 1
fi

echo "✅ Controller pod is Running"
echo ""

# Check for recent patches in logs
echo "📝 Checking for recent status patches in controller logs..."
PATCH_COUNT=$(kubectl logs -n "$CONTROLLER_NS" -l app.kubernetes.io/name=mcp-governance \
    --tail=100 2>/dev/null | grep -c "Successfully patched" || echo "0")

if [ "$PATCH_COUNT" -gt 0 ]; then
    echo "✅ Found $PATCH_COUNT successful status patches in recent logs"
else
    echo "⚠️  No recent patches found in logs. Controller may be waiting for events."
fi

echo ""

# Check actual status fields
echo "🔎 Verifying .status.publisher fields:"
echo ""

VERIFIED_COUNT=0
EMPTY_COUNT=0

CATALOG_NAMES=$(kubectl get mcpservercatalog -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

if [ -z "$CATALOG_NAMES" ]; then
    echo "⚠️  No MCPServerCatalog resources found"
else
    for CATALOG_NAME in $CATALOG_NAMES; do
        SCORE=$(kubectl get mcpservercatalog "$CATALOG_NAME" -n "$NAMESPACE" \
            -o jsonpath='{.status.publisher.score}' 2>/dev/null || echo "")
        GRADE=$(kubectl get mcpservercatalog "$CATALOG_NAME" -n "$NAMESPACE" \
            -o jsonpath='{.status.publisher.grade}' 2>/dev/null || echo "")
        
        if [ -n "$SCORE" ] && [ -n "$GRADE" ]; then
            echo "  ✅ $CATALOG_NAME — Score: $SCORE, Grade: $GRADE"
            VERIFIED_COUNT=$((VERIFIED_COUNT + 1))
        else
            echo "  ⚠️  $CATALOG_NAME — Status not patched (missing score/grade)"
            EMPTY_COUNT=$((EMPTY_COUNT + 1))
        fi
    done
fi

echo ""
echo "Summary:"
TOTAL=$((VERIFIED_COUNT + EMPTY_COUNT))
echo "  Patched: $VERIFIED_COUNT/$TOTAL"
echo "  Pending: $EMPTY_COUNT/$TOTAL"
echo ""

# Check for RBAC errors
echo "🔐 Checking for RBAC errors..."
RBAC_ERRORS=$(kubectl logs -n "$CONTROLLER_NS" -l app.kubernetes.io/name=mcp-governance \
    --tail=200 2>/dev/null | grep -c "forbidden\|cannot patch" || true)

if [ "$RBAC_ERRORS" -gt 0 ]; then
    echo "❌ Found $RBAC_ERRORS RBAC errors. Controller lacks permissions."
    echo ""
    echo "Fix by updating ClusterRole:"
    echo "  kubectl patch clusterrole mcp-governance-controller --type='json' -p='[{\"op\": \"add\", \"path\": \"/rules/-\", \"value\": {\"apiGroups\": [\"agentregistry.dev\"], \"resources\": [\"mcpservercatalogs/status\", \"agentcatalogs/status\", \"skillcatalogs/status\", \"modelcatalogs/status\"], \"verbs\": [\"get\", \"patch\", \"update\"]}}]'"
    exit 1
fi

echo "✅ No RBAC errors detected"
echo ""

# Display sample status
if [ "$VERIFIED_COUNT" -gt 0 ]; then
    echo "📌 Sample catalog status:"
    FIRST_CATALOG=$(kubectl get mcpservercatalog -n "$NAMESPACE" \
        -o jsonpath='{.items[0].metadata.name}')
    
    echo ""
    echo "  Name: $FIRST_CATALOG"
    kubectl get mcpservercatalog "$FIRST_CATALOG" -n "$NAMESPACE" \
        -o jsonpath='{.status.publisher}' | jq . 2>/dev/null || \
        kubectl get mcpservercatalog "$FIRST_CATALOG" -n "$NAMESPACE" \
        -o jsonpath='{.status.publisher}'
    echo ""
fi

# Final status
echo "✨ Verification Complete!"
echo ""

TOTAL=$((VERIFIED_COUNT + EMPTY_COUNT))
if [ "$TOTAL" -gt 0 ] && [ "$VERIFIED_COUNT" -eq "$TOTAL" ]; then
    echo "✅ All catalog resources have governance scores patched!"
    exit 0
elif [ "$VERIFIED_COUNT" -gt 0 ]; then
    echo "⚠️  Some catalogs are still pending patches. Check controller logs for progress."
    exit 0
else
    echo "⚠️  No catalogs have been patched yet. This is normal on first deployment."
    echo "    Wait for the controller to process catalog changes."
    exit 0
fi
