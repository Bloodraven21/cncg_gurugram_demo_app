{{/*
Expand the name of the chart.
*/}}
{{- define "o11y-starter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
Truncate at 63 chars because some Kubernetes name fields are limited.
*/}}
{{- define "o11y-starter.fullname" -}}
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
Create chart label value.
*/}}
{{- define "o11y-starter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "o11y-starter.labels" -}}
helm.sh/chart: {{ include "o11y-starter.chart" . }}
{{ include "o11y-starter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels used by the Deployment and Service.
*/}}
{{- define "o11y-starter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "o11y-starter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name — uses explicit name, or generates one, or uses "default".
*/}}
{{- define "o11y-starter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "o11y-starter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Gateway resource name — uses explicit name or falls back to fullname.
*/}}
{{- define "o11y-starter.gatewayName" -}}
{{- default (include "o11y-starter.fullname" .) .Values.gateway.name }}
{{- end }}

{{/*
Name of the Gateway the HTTPRoute parentRef points to.
When gateway.create is false, uses gateway.existingGatewayName.
*/}}
{{- define "o11y-starter.httpRouteParentRef" -}}
{{- if .Values.gateway.create }}
{{- include "o11y-starter.gatewayName" . }}
{{- else }}
{{- required "gateway.existingGatewayName is required when gateway.create is false" .Values.gateway.existingGatewayName }}
{{- end }}
{{- end }}

{{/*
Name of the Kubernetes Secret the Deployment's envFrom references.
For native secrets this is the fullname; for external secrets it uses
secret.external.targetName when set (ESO writes the synced Secret there).
*/}}
{{- define "o11y-starter.secretName" -}}
{{- if and (eq .Values.secret.type "external") .Values.secret.external.targetName }}
{{- .Values.secret.external.targetName }}
{{- else }}
{{- include "o11y-starter.fullname" . }}
{{- end }}
{{- end }}
