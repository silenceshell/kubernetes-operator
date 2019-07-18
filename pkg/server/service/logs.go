package service

import (
	"bufio"
	"fmt"
	"io"

	"github.com/gosoon/glog"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	jobLabelKey = "job-name"
)

func (s *service) GetClusterOperationLogs(region string, namespace string, name string) (string, error) {
	clusterClientset := s.opt.KubernetesClusterClientset
	var logs string
	kubernetesCluster, err := clusterClientset.EcsV1().KubernetesClusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("get kubernetesCluster %s/%s failed with:%v", namespace, name, err)
		return logs, err
	}

	jobName := kubernetesCluster.Status.JobName
	if jobName == "" {
		return logs, errors.New("no opreation logs")
	}

	kubeClientset := s.opt.KubeClientset
	// get job's pod by labelSelector
	labelSelector := fmt.Sprintf("%v=%v", jobLabelKey, jobName)
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	podList, err := kubeClientset.CoreV1().Pods(namespace).List(listOptions)
	if err != nil {
		glog.Errorf("list [%v=%v] lable pod failed with:%v", err)
		return logs, errors.Errorf("get job %v logs failed with:%v", jobName, err)
	}

	if len(podList.Items) == 0 {
		return logs, errors.Errorf("get job %v logs failed with:%v", jobName, err)
	}

	// get the pod logs
	podName := podList.Items[0].Name
	logOptions := &v1.PodLogOptions{}
	req := kubeClientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	readCloser, err := req.Stream()
	if err != nil {
		return logs, errors.Errorf("get job %v logs failed with:%v", jobName, err)
	}
	defer readCloser.Close()

	r := bufio.NewReader(readCloser)
	for {
		bytes, err := r.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return logs, errors.Errorf("get job %v logs failed with:%v", jobName, err)
			}
			break
		}
		logs += string(bytes)
	}
	return logs, nil
}
