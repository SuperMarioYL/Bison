{{/*
Expand the name of the chart.
*/}}
{{- define "bison.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "bison.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "bison.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "bison.labels" -}}
helm.sh/chart: {{ include "bison.chart" . }}
{{ include "bison.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "bison.selectorLabels" -}}
app.kubernetes.io/name: {{ include "bison.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "bison.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "bison.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
API Server full name
*/}}
{{- define "bison.apiServer.fullname" -}}
{{- printf "%s-api" (include "bison.fullname" .) }}
{{- end }}

{{/*
Web UI full name
*/}}
{{- define "bison.webUI.fullname" -}}
{{- printf "%s-web" (include "bison.fullname" .) }}
{{- end }}

{{/*
Get image registry
*/}}
{{- define "bison.imageRegistry" -}}
{{- if .Values.global.imageRegistry }}
{{- printf "%s/" .Values.global.imageRegistry }}
{{- else }}
{{- "" }}
{{- end }}
{{- end }}

{{/*
Auth secret name
*/}}
{{- define "bison.authSecretName" -}}
{{- if .Values.auth.admin.existingSecret }}
{{- .Values.auth.admin.existingSecret }}
{{- else }}
{{- printf "%s-auth" (include "bison.fullname" .) }}
{{- end }}
{{- end }}
