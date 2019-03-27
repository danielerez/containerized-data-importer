/*
Copyright 2018 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"reflect"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crdv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
)

// createVolumeSnapshotCRD creates CustomResourceDefinition
func createVolumeSnapshotClassCRD() *extv1beta1.CustomResourceDefinition {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdv1alpha1.VolumeSnapshotClassResourcePlural + "." + crdv1alpha1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crdv1alpha1.GroupName,
			Version: crdv1alpha1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.ClusterScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crdv1alpha1.VolumeSnapshotClassResourcePlural,
				Kind:   reflect.TypeOf(crdv1alpha1.VolumeSnapshotClass{}).Name(),
			},
		},
	}
	return crd
}

func createVolumeSnapshotContentCRD() *extv1beta1.CustomResourceDefinition {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdv1alpha1.VolumeSnapshotContentResourcePlural + "." + crdv1alpha1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crdv1alpha1.GroupName,
			Version: crdv1alpha1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.ClusterScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crdv1alpha1.VolumeSnapshotContentResourcePlural,
				Kind:   reflect.TypeOf(crdv1alpha1.VolumeSnapshotContent{}).Name(),
			},
		},
	}
	return crd
}

func createVolumeSnapshotCRD() *extv1beta1.CustomResourceDefinition {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdv1alpha1.VolumeSnapshotResourcePlural + "." + crdv1alpha1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crdv1alpha1.GroupName,
			Version: crdv1alpha1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crdv1alpha1.VolumeSnapshotResourcePlural,
				Kind:   reflect.TypeOf(crdv1alpha1.VolumeSnapshot{}).Name(),
			},
		},
	}
	return crd
}
