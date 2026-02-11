.PHONY: all build-controller build-dashboard load-images deploy clean dev-api dev-dashboard test helm-install helm-install-samples helm-upgrade helm-uninstall helm-template

CLUSTER_NAME := mcp-governance
CONTROLLER_IMAGE := mcp-governance-controller:latest
DASHBOARD_IMAGE := mcp-governance-dashboard:latest
HELM_RELEASE := mcp-governance
HELM_CHART := ./charts/mcp-governance

# =====================
# BUILD
# =====================

all: build-controller build-dashboard load-images deploy

test:
	@echo "🧪 Running controller tests..."
	cd controller && go test ./... -v -count=1

build-controller:
	@echo "🔨 Building controller image..."
	cd controller && podman build -t $(CONTROLLER_IMAGE) .

build-dashboard:
	@echo "🔨 Building dashboard image..."
	cd dashboard && podman build -t $(DASHBOARD_IMAGE) .

# =====================
# KIND
# =====================

load-images:
	@echo "📦 Loading images into Kind cluster..."
	kind load docker-image $(CONTROLLER_IMAGE) --name $(CLUSTER_NAME)
	kind load docker-image $(DASHBOARD_IMAGE) --name $(CLUSTER_NAME)

create-cluster:
	@echo "🏗️ Creating Kind cluster..."
	kind create cluster --config kind-config.yaml
	kubectl create namespace mcp-governance --dry-run=client -o yaml | kubectl apply -f -
	kubectl create namespace mcp-system --dry-run=client -o yaml | kubectl apply -f -
	kubectl create namespace agents --dry-run=client -o yaml | kubectl apply -f -

delete-cluster:
	kind delete cluster --name $(CLUSTER_NAME)

# =====================
# DEPLOY
# =====================

deploy: deploy-crds deploy-app

deploy-crds:
	@echo "📋 Applying CRDs..."
	kubectl apply -f deploy/crds/governance-crds.yaml

deploy-app:
	@echo "🚀 Deploying controller and dashboard..."
	kubectl apply -f deploy/k8s/deployment.yaml

deploy-samples:
	@echo "📝 Applying sample resources..."
	kubectl apply -f deploy/samples/governance-policy.yaml
	kubectl apply -f deploy/samples/demo-resources.yaml 2>/dev/null || true

undeploy:
	kubectl delete -f deploy/k8s/deployment.yaml --ignore-not-found
	kubectl delete -f deploy/crds/governance-crds.yaml --ignore-not-found

# =====================
# DEV (local)
# =====================

dev-api:
	@echo "🔧 Starting API server locally..."
	cd controller && go run cmd/api/main.go

dev-dashboard:
	@echo "🎨 Starting dashboard locally..."
	cd dashboard && npm run dev

dev: 
	@echo "Starting both API and dashboard..."
	@make dev-api &
	@make dev-dashboard

# =====================
# STATUS
# =====================

status:
	@echo "📊 Cluster Status:"
	@kubectl get nodes
	@echo ""
	@echo "📦 Deployments:"
	@kubectl get deployments -n mcp-governance
	@echo ""
	@echo "🔌 Services:"
	@kubectl get services -n mcp-governance
	@echo ""
	@echo "📋 Pods:"
	@kubectl get pods -n mcp-governance
	@echo ""
	@echo "🔍 CRDs:"
	@kubectl get crds | grep governance || echo "No governance CRDs found"

logs-controller:
	kubectl logs -f -n mcp-governance -l app.kubernetes.io/component=controller

logs-dashboard:
	kubectl logs -f -n mcp-governance -l app.kubernetes.io/component=dashboard

# =====================
# CLEAN
# =====================

clean:
	podman rmi $(CONTROLLER_IMAGE) $(DASHBOARD_IMAGE) 2>/dev/null || true

# =====================
# HELM
# =====================

helm-install:
	@echo "⎈ Installing MCP Governance via Helm..."
	helm install $(HELM_RELEASE) $(HELM_CHART) --create-namespace

helm-install-samples:
	@echo "⎈ Installing MCP Governance via Helm (with samples)..."
	helm install $(HELM_RELEASE) $(HELM_CHART) --create-namespace --set samples.install=true

helm-upgrade:
	@echo "⎈ Upgrading MCP Governance Helm release..."
	helm upgrade $(HELM_RELEASE) $(HELM_CHART)

helm-uninstall:
	@echo "⎈ Uninstalling MCP Governance Helm release..."
	helm uninstall $(HELM_RELEASE)

helm-template:
	@echo "⎈ Rendering Helm templates..."
	helm template $(HELM_RELEASE) $(HELM_CHART)
