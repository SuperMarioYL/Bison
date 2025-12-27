# Bison - GPU 资源计费平台
# 基于 Capsule + OpenCost 架构

# ==================== 配置 ====================
REGISTRY ?= docker.io
REPO ?= bison
VERSION ?= latest
HELM_RELEASE ?= bison
NAMESPACE ?= bison-system

DOCKER_PLATFORMS ?= linux/amd64,linux/arm64
BINARY_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

API_IMAGE = $(REGISTRY)/$(REPO)/api-server:$(VERSION)
WEB_IMAGE = $(REGISTRY)/$(REPO)/web-ui:$(VERSION)
DIST_DIR = dist

.PHONY: help
help: ## 显示帮助
	@echo "Usage: make <target>"
	@echo ""
	@echo "本地开发:"
	@echo "  dev-api          本地运行 API 服务器"
	@echo "  dev-web          本地运行 Web UI"
	@echo "  dev-docs         本地运行文档站点 (http://localhost:3001)"
	@echo "  dev              同时运行 API 和 Web (需要 tmux)"
	@echo "  install-deps     安装开发依赖"
	@echo ""
	@echo "文档:"
	@echo "  dev-docs         本地运行文档站点开发服务器"
	@echo "  build-docs       构建生产版本文档"
	@echo "  serve-docs       本地预览构建后的文档"
	@echo "  clean-docs       清理文档构建产物"
	@echo ""
	@echo "测试和检查:"
	@echo "  test             运行所有测试"
	@echo "  lint             运行所有 linter"
	@echo ""
	@echo "构建:"
	@echo "  build            构建 Docker 镜像 (当前平台)"
	@echo "  build-local      使用 buildx 构建并加载到本地"
	@echo "  build-multiarch  构建多平台镜像"
	@echo "  build-binary     构建二进制文件"
	@echo ""
	@echo "发布:"
	@echo "  push             推送镜像到仓库"
	@echo "  release          构建并推送多平台镜像"
	@echo ""
	@echo "部署 - 测试环境 (NodePort):"
	@echo "  deploy-dev-env   安装测试环境依赖 (Capsule + Prometheus + OpenCost)"
	@echo ""
	@echo "部署 - 生产环境 (ClusterIP):"
	@echo "  deploy-prod-env  安装生产环境依赖 (Capsule + Prometheus + OpenCost)"
	@echo "  deploy           部署 Bison 到 Kubernetes"
	@echo "  undeploy         卸载 Bison"
	@echo ""
	@echo "Helm:"
	@echo "  helm-lint        验证 Helm chart"
	@echo "  helm-template    渲染 Helm 模板"
	@echo ""
	@echo "其他:"
	@echo "  status           查看部署状态"
	@echo "  clean            清理构建产物"

# ==================== 本地开发 ====================

.PHONY: dev-api
dev-api: ## 本地运行 API 服务器
	cd api-server && go run cmd/main.go

.PHONY: dev-web
dev-web: ## 本地运行 Web UI
	cd web-ui && npm run dev

.PHONY: dev
dev: ## 同时运行 API 和 Web (需要 tmux)
	@if command -v tmux &> /dev/null; then \
		tmux new-session -d -s bison-dev 'make dev-api' && \
		tmux split-window -h 'make dev-web' && \
		tmux attach -t bison-dev; \
	else \
		echo "tmux not found. Run 'make dev-api' and 'make dev-web' in separate terminals."; \
	fi

.PHONY: install-deps
install-deps: ## 安装开发依赖
	cd api-server && go mod download
	cd web-ui && npm install
	cd website && npm install

.PHONY: tidy
tidy: ## 整理 Go 依赖
	cd api-server && go mod tidy

# ==================== 文档 ====================

.PHONY: dev-docs
dev-docs: ## 本地运行文档站点开发服务器
	@echo "Starting Docusaurus development server..."
	@echo "访问: http://localhost:3001"
	cd website && npm run start

.PHONY: build-docs
build-docs: ## 构建生产版本文档
	@echo "Building Docusaurus site for production..."
	cd website && npm run build
	@echo "Build complete! Files in website/build/"

.PHONY: serve-docs
serve-docs: ## 本地预览构建后的文档
	@echo "Serving built documentation..."
	@echo "访问: http://localhost:3001"
	cd website && npm run serve

.PHONY: clean-docs
clean-docs: ## 清理文档构建产物
	rm -rf website/build
	rm -rf website/.docusaurus

# ==================== 测试和检查 ====================

.PHONY: test
test: test-api test-web ## 运行所有测试

.PHONY: test-api
test-api:
	cd api-server && go test ./...

.PHONY: test-web
test-web:
	cd web-ui && npm run test || true

.PHONY: lint
lint: lint-api lint-web ## 运行所有 linter

.PHONY: lint-api
lint-api:
	cd api-server && go vet ./...

.PHONY: lint-web
lint-web:
	cd web-ui && npm run lint

# ==================== 构建 Docker 镜像 ====================

.PHONY: build
build: build-api build-web ## 构建所有镜像 (当前平台)

.PHONY: build-api
build-api:
	docker build -t $(API_IMAGE) -f api-server/Dockerfile api-server

.PHONY: build-web
build-web:
	docker build -t $(WEB_IMAGE) -f web-ui/Dockerfile web-ui

.PHONY: buildx-setup
buildx-setup:
	@if ! docker buildx inspect bison-builder > /dev/null 2>&1; then \
		docker buildx create --name bison-builder --driver docker-container --bootstrap --use; \
	else \
		docker buildx use bison-builder; \
	fi

.PHONY: build-local
build-local: buildx-setup ## 使用 buildx 构建并加载到本地
	docker buildx build --load -t $(API_IMAGE) -f api-server/Dockerfile api-server
	docker buildx build --load -t $(WEB_IMAGE) -f web-ui/Dockerfile web-ui

.PHONY: build-multiarch
build-multiarch: buildx-setup ## 构建多平台镜像 (仅缓存)
	docker buildx build --platform $(DOCKER_PLATFORMS) -t $(API_IMAGE) -f api-server/Dockerfile api-server
	docker buildx build --platform $(DOCKER_PLATFORMS) -t $(WEB_IMAGE) -f web-ui/Dockerfile web-ui

# ==================== 构建二进制文件 ====================

.PHONY: build-binary
build-binary: ## 构建所有平台二进制文件
	@mkdir -p $(DIST_DIR)
	@for platform in $(BINARY_PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		output=$(DIST_DIR)/api-server-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then output=$$output.exe; fi; \
		echo "Building api-server for $$os/$$arch..."; \
		cd api-server && CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags="-w -s" -o ../$$output ./cmd/main.go && cd ..; \
	done

.PHONY: build-binary-local
build-binary-local: ## 构建当前平台二进制文件
	@mkdir -p $(DIST_DIR)
	cd api-server && go build -ldflags="-w -s" -o ../$(DIST_DIR)/api-server ./cmd/main.go

# ==================== 推送和发布 ====================

.PHONY: push
push: ## 推送镜像到仓库
	docker push $(API_IMAGE)
	docker push $(WEB_IMAGE)

.PHONY: release
release: buildx-setup ## 构建并推送多平台镜像
	docker buildx build --platform $(DOCKER_PLATFORMS) -t $(API_IMAGE) -f api-server/Dockerfile api-server --push
	docker buildx build --platform $(DOCKER_PLATFORMS) -t $(WEB_IMAGE) -f web-ui/Dockerfile web-ui --push

# ==================== 部署 - 测试环境 (NodePort) ====================

.PHONY: deploy-dev-env
deploy-dev-env: deploy-capsule deploy-dev-prometheus deploy-dev-opencost ## 安装测试环境依赖 (带 NodePort)
	@echo ""
	@echo "=========================================="
	@echo "  测试环境依赖安装完成"
	@echo "=========================================="
	@echo "  Prometheus: http://<node-ip>:30090"
	@echo "  Grafana:    http://<node-ip>:30080 (admin/admin123)"
	@echo "  OpenCost:   http://<node-ip>:30009"
	@echo ""

.PHONY: deploy-dev-prometheus
deploy-dev-prometheus: ## 安装 Prometheus (NodePort)
	@echo "Installing Prometheus (dev mode with NodePort)..."
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
	helm repo update
	helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
		--namespace monitoring \
		--create-namespace \
		--set prometheus.service.type=NodePort \
		--set prometheus.service.nodePort=30090 \
		--set grafana.service.type=NodePort \
		--set grafana.service.nodePort=30080 \
		--set grafana.adminPassword=admin123

.PHONY: deploy-dev-opencost
deploy-dev-opencost: ## 安装 OpenCost (NodePort)
	@echo "Installing OpenCost (dev mode with NodePort)..."
	helm repo add opencost https://opencost.github.io/opencost-helm-chart || true
	helm repo update
	helm upgrade --install opencost opencost/opencost \
		--namespace opencost \
		--create-namespace \
		--set opencost.prometheus.internal.serviceName=prometheus-kube-prometheus-prometheus \
		--set opencost.prometheus.internal.namespaceName=monitoring \
		--set opencost.prometheus.internal.port=9090 \
		--set opencost.ui.enabled=true \
		--set service.type=NodePort
	@sleep 3
	kubectl patch svc opencost -n opencost --type='json' \
		-p='[{"op":"replace","path":"/spec/ports/0/nodePort","value":30009}]' || true

# ==================== 部署 - 生产环境 (ClusterIP) ====================

.PHONY: deploy-prod-env
deploy-prod-env: deploy-capsule deploy-prometheus deploy-opencost ## 安装生产环境依赖 (ClusterIP)
	@echo ""
	@echo "=========================================="
	@echo "  生产环境依赖安装完成"
	@echo "=========================================="
	@echo "  使用 kubectl port-forward 或 Ingress 访问服务"
	@echo ""

.PHONY: deploy-capsule
deploy-capsule: ## 安装 Capsule
	@echo "Installing Capsule..."
	helm repo add projectcapsule https://projectcapsule.github.io/charts || true
	helm repo update
	helm upgrade --install capsule projectcapsule/capsule \
		--namespace capsule-system \
		--create-namespace
	@echo "Waiting for Capsule..."
	kubectl wait --for=condition=available --timeout=120s deployment/capsule-controller-manager -n capsule-system || true

.PHONY: deploy-prometheus
deploy-prometheus: ## 安装 Prometheus (ClusterIP)
	@echo "Installing Prometheus..."
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
	helm repo update
	helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
		--namespace monitoring \
		--create-namespace

.PHONY: deploy-opencost
deploy-opencost: ## 安装 OpenCost (ClusterIP)
	@echo "Installing OpenCost..."
	helm repo add opencost https://opencost.github.io/opencost-helm-chart || true
	helm repo update
	helm upgrade --install opencost opencost/opencost \
		--namespace opencost \
		--create-namespace \
		--set opencost.prometheus.internal.serviceName=prometheus-kube-prometheus-prometheus \
		--set opencost.prometheus.internal.namespaceName=monitoring \
		--set opencost.prometheus.internal.port=9090 \
		--set opencost.ui.enabled=true

.PHONY: undeploy-env
undeploy-env: ## 卸载所有依赖
	helm uninstall capsule -n capsule-system || true
	helm uninstall opencost -n opencost || true
	helm uninstall prometheus -n monitoring || true

# ==================== 部署 Bison ====================

.PHONY: deploy
deploy: ## 部署 Bison
	helm upgrade --install $(HELM_RELEASE) ./deploy/charts/bison \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set apiServer.image.repository=$(REGISTRY)/$(REPO)/api-server \
		--set apiServer.image.tag=$(VERSION) \
		--set webUI.image.repository=$(REGISTRY)/$(REPO)/web-ui \
		--set webUI.image.tag=$(VERSION)

.PHONY: deploy-with-auth
deploy-with-auth: ## 部署 Bison (启用认证)
	helm upgrade --install $(HELM_RELEASE) ./deploy/charts/bison \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set auth.enabled=true \
		--set auth.admin.password=$$(openssl rand -base64 12) \
		--set apiServer.image.repository=$(REGISTRY)/$(REPO)/api-server \
		--set apiServer.image.tag=$(VERSION) \
		--set webUI.image.repository=$(REGISTRY)/$(REPO)/web-ui \
		--set webUI.image.tag=$(VERSION)

.PHONY: undeploy
undeploy: ## 卸载 Bison
	helm uninstall $(HELM_RELEASE) --namespace $(NAMESPACE) || true

# ==================== Helm ====================

.PHONY: helm-lint
helm-lint: ## 验证 Helm chart
	helm lint ./deploy/charts/bison

.PHONY: helm-template
helm-template: ## 渲染 Helm 模板
	helm template $(HELM_RELEASE) ./deploy/charts/bison --namespace $(NAMESPACE)

.PHONY: helm-package
helm-package: ## 打包 Helm chart
	helm package ./deploy/charts/bison

# ==================== 数据管理 ====================

.PHONY: backup-data
backup-data: ## 备份 ConfigMap 数据
	@mkdir -p backups
	kubectl get configmap -n $(NAMESPACE) -l app.kubernetes.io/name=bison -o yaml > backups/bison-configmaps-$$(date +%Y%m%d-%H%M%S).yaml
	@echo "Backup saved to backups/"

.PHONY: show-balances
show-balances: ## 显示团队余额
	kubectl get configmap -n $(NAMESPACE) bison-team-balances -o jsonpath='{.data}' | jq . || echo "No balances found"

# ==================== 清理 ====================

.PHONY: clean
clean: ## 清理构建产物
	rm -rf $(DIST_DIR)
	rm -rf web-ui/dist
	rm -rf website/build
	rm -rf website/.docusaurus
	rm -f *.tgz

.PHONY: clean-all
clean-all: clean ## 清理所有 (包括 node_modules 和 Docker)
	rm -rf web-ui/node_modules
	rm -rf website/node_modules
	docker rmi $(API_IMAGE) || true
	docker rmi $(WEB_IMAGE) || true
	docker buildx rm bison-builder || true

# ==================== 状态和信息 ====================

.PHONY: status
status: ## 查看部署状态
	@echo "=== Capsule ==="
	@kubectl get deployment -n capsule-system 2>/dev/null || echo "Not installed"
	@echo ""
	@echo "=== Prometheus ==="
	@kubectl get deployment -n monitoring 2>/dev/null | head -3 || echo "Not installed"
	@echo ""
	@echo "=== OpenCost ==="
	@kubectl get deployment -n opencost 2>/dev/null || echo "Not installed"
	@echo ""
	@echo "=== Bison ==="
	@kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=bison 2>/dev/null || echo "Not deployed"

.PHONY: info
info: ## 显示项目信息
	@echo "=== Bison GPU 资源计费平台 ==="
	@echo "Version:    $(VERSION)"
	@echo "Registry:   $(REGISTRY)/$(REPO)"
	@echo "Namespace:  $(NAMESPACE)"
	@echo ""
	@echo "Images:"
	@echo "  API:  $(API_IMAGE)"
	@echo "  Web:  $(WEB_IMAGE)"
