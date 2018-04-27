package client

/*
Copyright 2017-2018 Crunchy Data Solutions.

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

import (
	"fmt"
	"reflect"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE backups).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const backupCRDName = crv1.PgbackupResourcePlural + "." + crv1.GroupName

// PgbackupCreateCustomResourceDefinition creates the CRD for pgbackups
func PgbackupCreateCustomResourceDefinition(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: backupCRDName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crv1.GroupName,
			Version: crv1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crv1.PgbackupResourcePlural,
				Kind:   reflect.TypeOf(crv1.Pgbackup{}).Name(),
			},
		},
	}
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(backupCRDName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					fmt.Printf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(backupCRDName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}
	return crd, nil
}

// WaitForPgbackupInstanceProcessed polls till a backup is processed
func WaitForPgbackupInstanceProcessed(exampleClient *rest.RESTClient, name string) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		var backup crv1.Pgbackup
		err := exampleClient.Get().
			Resource(crv1.PgbackupResourcePlural).
			Namespace(apiv1.NamespaceDefault).
			Name(name).
			Do().Into(&backup)

		if err == nil && backup.Status.State == crv1.PgbackupStateProcessed {
			return true, nil
		}

		return false, err
	})
}
