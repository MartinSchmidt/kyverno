package kube

import (
	"encoding/json"

	datautils "github.com/kyverno/kyverno/pkg/utils/data"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RedactSecret masks keys of data and metadata.annotation fields of Secrets.
func RedactSecret(resource *unstructured.Unstructured) (unstructured.Unstructured, error) {
	var secret *corev1.Secret
	data, err := json.Marshal(resource.Object)
	if err != nil {
		return *resource, err
	}
	err = json.Unmarshal(data, &secret)
	if err != nil {
		return *resource, errors.Wrap(err, "unable to convert object to secret")
	}
	stringSecret := struct {
		Data map[string]string `json:"string_data"`
		*corev1.Secret
	}{
		Data:   make(map[string]string),
		Secret: secret,
	}
	for key := range secret.Data {
		secret.Data[key] = []byte("**REDACTED**")
		stringSecret.Data[key] = string(secret.Data[key])
	}
	for key := range secret.Annotations {
		secret.Annotations[key] = "**REDACTED**"
	}
	updateSecret := map[string]interface{}{}
	raw, err := json.Marshal(stringSecret)
	if err != nil {
		return *resource, nil
	}
	err = json.Unmarshal(raw, &updateSecret)
	if err != nil {
		return *resource, errors.Wrap(err, "unable to convert object from secret")
	}
	if secret.Data != nil {
		v := updateSecret["string_data"].(map[string]interface{})
		err = unstructured.SetNestedMap(resource.Object, v, "data")
		if err != nil {
			return *resource, errors.Wrap(err, "failed to set secret.data")
		}
	}
	if secret.Annotations != nil {
		metadata, err := datautils.ToMap(resource.Object["metadata"])
		if err != nil {
			return *resource, errors.Wrap(err, "unable to convert metadata to map")
		}
		updatedMeta := updateSecret["metadata"].(map[string]interface{})
		if err != nil {
			return *resource, errors.Wrap(err, "unable to convert object from secret")
		}
		err = unstructured.SetNestedMap(metadata, updatedMeta["annotations"].(map[string]interface{}), "annotations")
		if err != nil {
			return *resource, errors.Wrap(err, "failed to set secret.annotations")
		}
	}
	return *resource, nil
}
