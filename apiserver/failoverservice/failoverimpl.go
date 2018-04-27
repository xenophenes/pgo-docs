package failoverservice

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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
	"errors"
	log "github.com/Sirupsen/logrus"
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/apiserver"
	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/labels"
)

//  CreateFailover ...
// pgo failover mycluster
// pgo failover all
// pgo failover --selector=name=mycluster
func CreateFailover(request *msgs.CreateFailoverRequest) msgs.CreateFailoverResponse {
	var err error
	resp := msgs.CreateFailoverResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)

	if request.Target != "" {
		_, err = validateDeploymentName(request.Target)
		if err != nil {
			resp.Status.Code = msgs.Error
			resp.Status.Msg = err.Error()
			return resp
		}
	}

	//get the clusters list
	//var cluster *crv1.Pgcluster
	_, err = validateClusterName(request.ClusterName)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debug("create failover called for " + request.ClusterName)

	// Create a pgtask
	spec := crv1.PgtaskSpec{}
	spec.Name = request.ClusterName + "-failover"
	spec.TaskType = crv1.PgtaskFailover
	spec.Parameters = request.ClusterName
	labels := make(map[string]string)
	labels["target"] = request.Target

	newInstance := &crv1.Pgtask{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   spec.Name,
			Labels: labels,
		},
		Spec: spec,
	}

	err = kubeapi.Createpgtask(apiserver.RESTClient,
		newInstance, apiserver.Namespace)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	resp.Results = append(resp.Results, "created Pgtask (failover) for cluster "+request.ClusterName)

	return resp
}

//  QueryFailover ...
// pgo failover mycluster --query
func QueryFailover(request *msgs.CreateFailoverRequest) msgs.CreateFailoverResponse {
	var err error
	resp := msgs.CreateFailoverResponse{}
	resp.Status.Code = msgs.Ok
	resp.Status.Msg = ""
	resp.Results = make([]string, 0)
	resp.Targets = make([]string, 0)

	//var deployment *v1beta1.Deployment

	//get the clusters list
	//var cluster *crv1.Pgcluster
	_, err = validateClusterName(request.ClusterName)
	if err != nil {
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debug("query failover called for " + request.ClusterName)

	//get failover targets for this cluster
	//deployments with --selector=replica=true,pg-cluster=ClusterName

	selector := "replica=true,pg-cluster=" + request.ClusterName

	deployments, err := kubeapi.GetDeployments(apiserver.Clientset, selector, apiserver.Namespace)
	if kerrors.IsNotFound(err) {
		log.Debug("no replicas found ")
		resp.Status.Msg = "no replicas found for " + request.ClusterName
		return resp
	} else if err != nil {
		log.Error("error getting deployments " + err.Error())
		resp.Status.Code = msgs.Error
		resp.Status.Msg = err.Error()
		return resp
	}

	log.Debugf("deps len %d\n", len(deployments.Items))
	for _, dep := range deployments.Items {
		log.Debug("found " + dep.Name)
		resp.Targets = append(resp.Targets, dep.Name)
	}

	//resp.Results = append(resp.Results, "")

	return resp
}

func validateClusterName(clusterName string) (*crv1.Pgcluster, error) {
	cluster := crv1.Pgcluster{}
	found, err := kubeapi.Getpgcluster(apiserver.RESTClient,
		&cluster, clusterName, apiserver.Namespace)
	if !found {
		return &cluster, errors.New("no cluster found named " + clusterName)
	}

	return &cluster, err
}

func validateDeploymentName(deployName string) (*v1beta1.Deployment, error) {

	deployment, found, err := kubeapi.GetDeployment(apiserver.Clientset, deployName, apiserver.Namespace)
	if !found {
		return deployment, errors.New("no target found named " + deployName)
	}

	return deployment, err

}
