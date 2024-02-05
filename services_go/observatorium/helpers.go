package observatorium

import (
	"sort"

	kghelpers "github.com/observatorium/observatorium/configuration_go/kubegen/helpers"
	"github.com/observatorium/observatorium/configuration_go/schemas/thanos/objstore"
	objstore3 "github.com/observatorium/observatorium/configuration_go/schemas/thanos/objstore/s3"
	templatev1 "github.com/openshift/api/template/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

// postProcessServiceMonitor updates the service monitor to work with the app-sre prometheus.
func postProcessServiceMonitor(serviceMonitor *monv1.ServiceMonitor, namespaceSelector string) {
	serviceMonitor.ObjectMeta.Namespace = monitoringNamespace
	serviceMonitor.Spec.NamespaceSelector.MatchNames = []string{namespaceSelector}
	serviceMonitor.ObjectMeta.Labels["prometheus"] = "app-sre"
	// Prefix the service monitor name with the namespace to avoid conflicts.
	serviceMonitor.ObjectMeta.Name = namespaceSelector + "-" + serviceMonitor.ObjectMeta.Name
}

// deleteObjStoreEnv deletes the objstore env var from the list of env vars.
// This env var is included by default by the observatorium config for each thanos component.
func deleteObjStoreEnv(objStoreEnv []corev1.EnvVar) []corev1.EnvVar {
	for i, env := range objStoreEnv {
		if env.Name == "OBJSTORE_CONFIG" {
			return append(objStoreEnv[:i], objStoreEnv[i+1:]...)
		}
	}

	return objStoreEnv
}

// objStoreEnvVars returns the env vars required for the objstore config.
// Base env vars are taken from the s3 secret generated by app-interface.
// The objstore config env var is generated by aggregating the other env vars.
func objStoreEnvVars(objstoreSecret string) []corev1.EnvVar {
	objStoreCfg, err := yaml.Marshal(objstore.BucketConfig{
		Type: objstore.S3,
		Config: objstore3.Config{
			Bucket:   "$(OBJ_STORE_BUCKET)",
			Endpoint: "$(OBJ_STORE_ENDPOINT)",
			Region:   "$(OBJ_STORE_REGION)",
		},
	})
	if err != nil {
		panic(err)
	}

	return []corev1.EnvVar{
		kghelpers.NewEnvFromSecret("AWS_ACCESS_KEY_ID", objstoreSecret, "aws_access_key_id"),
		kghelpers.NewEnvFromSecret("AWS_SECRET_ACCESS_KEY", objstoreSecret, "aws_secret_access_key"),
		kghelpers.NewEnvFromSecret("OBJ_STORE_BUCKET", objstoreSecret, "bucket"),
		kghelpers.NewEnvFromSecret("OBJ_STORE_REGION", objstoreSecret, "aws_region"),
		kghelpers.NewEnvFromSecret("OBJ_STORE_ENDPOINT", objstoreSecret, "endpoint"),
		{
			Name:  "OBJSTORE_CONFIG",
			Value: string(objStoreCfg),
		},
	}
}

func addQuayPullSecret(sa *corev1.ServiceAccount) {
	sa.ImagePullSecrets = append(sa.ImagePullSecrets, corev1.LocalObjectReference{
		Name: "quay.io",
	})
}

func sortTemplateParams(params []templatev1.Parameter) []templatev1.Parameter {
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	return params
}

func executeIfNotNil[T any](f func(T), param T) {
	if f != nil {
		f(param)
	}
}
