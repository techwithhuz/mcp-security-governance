{{/*
Expand the name of the chart.
*/}}
{{- define "mcp-governance.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "mcp-governance.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart label
*/}}
{{- define "mcp-governance.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mcp-governance.labels" -}}
helm.sh/chart: {{ include "mcp-governance.chart" . }}
app.kubernetes.io/name: mcp-governance
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Controller labels
*/}}
{{- define "mcp-governance.controllerLabels" -}}
{{ include "mcp-governance.labels" . }}
app.kubernetes.io/component: controller
{{- end }}

{{/*
Dashboard labels
*/}}
{{- define "mcp-governance.dashboardLabels" -}}
{{ include "mcp-governance.labels" . }}
app.kubernetes.io/component: dashboard
{{- end }}

{{/*
Controller selector labels
*/}}
{{- define "mcp-governance.controllerSelectorLabels" -}}
app.kubernetes.io/name: mcp-governance
app.kubernetes.io/component: controller
{{- end }}

{{/*
Dashboard selector labels
*/}}
{{- define "mcp-governance.dashboardSelectorLabels" -}}
app.kubernetes.io/name: mcp-governance
app.kubernetes.io/component: dashboard
{{- end }}

{{/*
Service account name
*/}}
{{- define "mcp-governance.serviceAccountName" -}}
{{- if .Values.controller.serviceAccount.create }}
{{- default "mcp-governance-controller" .Values.controller.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.controller.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Controller image
*/}}
{{- define "mcp-governance.controllerImage" -}}
{{- printf "%s:%s" .Values.controller.image.repository .Values.controller.image.tag }}
{{- end }}

{{/*
Dashboard image
*/}}
{{- define "mcp-governance.dashboardImage" -}}
{{- printf "%s:%s" .Values.dashboard.image.repository .Values.dashboard.image.tag }}
{{- end }}

{{/*
Controller in-cluster API URL.
Uses dashboard.apiUrl if set explicitly, otherwise builds from the controller
service name, release namespace, and port.
*/}}
{{- define "mcp-governance.controllerApiUrl" -}}
{{- if .Values.dashboard.apiUrl }}
{{- .Values.dashboard.apiUrl }}
{{- else }}
{{- printf "http://mcp-governance-controller.%s.svc.cluster.local:%v" .Release.Namespace (.Values.controller.port | default 8090) }}
{{- end }}
{{- end }}
