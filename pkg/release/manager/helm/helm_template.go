package helm

import (
	"WarpCloud/walm/pkg/release"
	"k8s.io/helm/pkg/chart/loader"
	"github.com/sirupsen/logrus"
	"bytes"
	"WarpCloud/walm/pkg/k8s/client"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/api/extensions/v1beta1"
	"encoding/json"
	"k8s.io/api/core/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	"WarpCloud/walm/pkg/k8s/adaptor"
	"WarpCloud/walm/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (hc *HelmClient) DryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, isSystem bool, chartFiles []*loader.BufferedFile) ([]map[string]interface{}, error) {
	release, err := hc.doInstallUpgradeRelease(namespace, releaseRequest, isSystem, chartFiles, true)
	if err != nil {
		logrus.Errorf("failed to dry run install release : %s", err.Error())
		return nil, err
	}
	logrus.Debugf("release manifest : %s", release.Manifest)
	resources, err := client.GetKubeClient(namespace).BuildUnstructured(namespace, bytes.NewBufferString(release.Manifest))
	if err != nil {
		logrus.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}

	results := []map[string]interface{}{}
	for _, resource := range resources {
		results = append(results, resource.Object.(*unstructured.Unstructured).Object)
	}

	return results, nil
}

func (hc *HelmClient) ComputeResourcesByDryRunRelease(namespace string, releaseRequest *release.ReleaseRequestV2, isSystem bool, chartFiles []*loader.BufferedFile) (*release.ReleaseResources, error) {
	r, err := hc.doInstallUpgradeRelease(namespace, releaseRequest, isSystem, chartFiles, true)
	if err != nil {
		logrus.Errorf("failed to dry run install release : %s", err.Error())
		return nil, err
	}
	logrus.Debugf("release manifest : %s", r.Manifest)
	resources, err := client.GetKubeClient(namespace).BuildUnstructured(namespace, bytes.NewBufferString(r.Manifest))
	if err != nil {
		logrus.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}

	result := &release.ReleaseResources{}
	for _, resource := range resources {
		unstructured := resource.Object.(*unstructured.Unstructured)
		switch unstructured.GetKind() {
		case "Deployment":
			releaseResourceDeployment, err := buildReleaseResourceDeployment(unstructured)
			if err != nil {
				logrus.Errorf("failed to build release resource deployment %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Deployments = append(result.Deployments, releaseResourceDeployment)
		case "StatefulSet":
			releaseResourceStatefulSet, err := buildReleaseResourceStatefulSet(unstructured)
			if err != nil {
				logrus.Errorf("failed to build release resource stateful set %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.StatefulSets = append(result.StatefulSets, releaseResourceStatefulSet)
		case "DaemonSet":
			releaseResourceDaemonSet, err := buildReleaseResourceDaemonSet(unstructured)
			if err != nil {
				logrus.Errorf("failed to build release resource daemon set %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.DaemonSets = append(result.DaemonSets, releaseResourceDaemonSet)
		case "Job":
			releaseResourceJob, err := buildReleaseResourceJob(unstructured)
			if err != nil {
				logrus.Errorf("failed to build release resource job %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Jobs = append(result.Jobs, releaseResourceJob)
		case "PersistentVolumeClaim":
			pvc, err := buildReleaseResourcePvc(unstructured)
			if err != nil {
				logrus.Errorf("failed to build release resource pvc %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Pvcs = append(result.Pvcs, pvc)
		default:
		}
	}

	return result, nil
}

func buildReleaseResourceDeployment(resource *unstructured.Unstructured) (*release.ReleaseResourceDeployment, error) {
	deployment := &v1beta1.Deployment{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		logrus.Errorf("failed to marshal deployment %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, deployment)
	if err != nil {
		logrus.Errorf("failed to unmarshal deployment %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResourceDeployment := &release.ReleaseResourceDeployment{
		Replicas: *deployment.Spec.Replicas,
	}

	releaseResourceDeployment.ReleaseResourceBase, err = buildReleaseResourceBase(resource, deployment.Spec.Template, nil)
	if err != nil {
		logrus.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResourceDeployment, nil
}

func buildReleaseResourceStatefulSet(resource *unstructured.Unstructured) (*release.ReleaseResourceStatefulSet, error) {
	statefulSet := &appsv1beta1.StatefulSet{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		logrus.Errorf("failed to marshal statefulSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, statefulSet)
	if err != nil {
		logrus.Errorf("failed to unmarshal statefulSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceStatefulSet{
		Replicas: *statefulSet.Spec.Replicas,
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, statefulSet.Spec.Template, statefulSet.Spec.VolumeClaimTemplates)
	if err != nil {
		logrus.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourceDaemonSet(resource *unstructured.Unstructured) (*release.ReleaseResourceDaemonSet, error) {
	daemonSet := &extv1beta1.DaemonSet{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		logrus.Errorf("failed to marshal daemonSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, daemonSet)
	if err != nil {
		logrus.Errorf("failed to unmarshal daemonSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceDaemonSet{
		NodeSelector: daemonSet.Spec.Template.Spec.NodeSelector,
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, daemonSet.Spec.Template, nil)
	if err != nil {
		logrus.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourceJob(resource *unstructured.Unstructured) (*release.ReleaseResourceJob, error) {
	job := &batchv1.Job{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		logrus.Errorf("failed to marshal job %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, job)
	if err != nil {
		logrus.Errorf("failed to unmarshal job %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceJob{}
	if job.Spec.Parallelism != nil {
		releaseResource.Parallelism = *job.Spec.Parallelism
	}
	if job.Spec.Completions != nil {
		releaseResource.Completions = *job.Spec.Completions
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, job.Spec.Template, nil)
	if err != nil {
		logrus.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourcePvc(resource *unstructured.Unstructured) (*release.ReleaseResourceStorage, error) {
	pvc := &v1.PersistentVolumeClaim{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		logrus.Errorf("failed to marshal pvc %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, pvc)
	if err != nil {
		logrus.Errorf("failed to unmarshal pvc %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	return buildPvcStorage(*pvc), nil
}

func buildReleaseResourceBase(r *unstructured.Unstructured, podTemplateSpec v1.PodTemplateSpec, pvcs []v1.PersistentVolumeClaim) (releaseResource release.ReleaseResourceBase, err error) {
	releaseResource = release.ReleaseResourceBase{
		Name:        r.GetName(),
		PodRequests: &release.ReleaseResourcePod{},
		PodLimits:   &release.ReleaseResourcePod{},
	}

	podRequests, podLimits := adaptor.GetPodRequestsAndLimits(podTemplateSpec.Spec)
	if quantity, ok := podRequests[v1.ResourceCPU]; ok {
		releaseResource.PodRequests.Cpu = float64(quantity.MilliValue()) / util.K8sResourceCpuScale
	}
	if quantity, ok := podRequests[v1.ResourceMemory]; ok {
		releaseResource.PodRequests.Memory = quantity.Value() / util.K8sResourceMemoryScale
	}
	if quantity, ok := podLimits[v1.ResourceCPU]; ok {
		releaseResource.PodLimits.Cpu = float64(quantity.MilliValue()) / util.K8sResourceCpuScale
	}
	if quantity, ok := podLimits[v1.ResourceMemory]; ok {
		releaseResource.PodLimits.Memory = quantity.Value() / util.K8sResourceMemoryScale
	}

	releaseResource.PodRequests.Storage = buildTosDiskStorage(r.Object)
	releaseResource.PodRequests.Storage = append(releaseResource.PodRequests.Storage, buildPvcStorages(pvcs)...)
	return
}

func buildTosDiskStorage(object map[string]interface{}) (tosDiskStorages []*release.ReleaseResourceStorage) {
	tosDiskStorages = []*release.ReleaseResourceStorage{}
	type TosDiskVolumeSource struct {
		Name        string        `json:"name" description:"tos disk name"`
		StorageType string        `json:"storageType" description:"tos disk storageType"`
		Capability  v1.Capability `json:"capability" description:"tos disk capability"`
	}

	volumes, found, err := unstructured.NestedSlice(object, "spec", "template", "spec", "volumes")
	if !found || err != nil {
		logrus.Warn("failed to find pod volumes")
		return
	}

	for _, volume := range volumes {
		if volumeMap, ok := volume.(map[string]interface{}); ok {
			if tosDisk, ok1 := volumeMap["tosDisk"]; ok1 {
				tosDiskBytes, err := json.Marshal(tosDisk)
				if err != nil {
					logrus.Warnf("failed to marshal tosDisk : %s", err.Error())
					continue
				}
				tosDiskVolumeSource := &TosDiskVolumeSource{}
				err = json.Unmarshal(tosDiskBytes, tosDiskVolumeSource)
				if err != nil {
					logrus.Warnf("failed to unmarshal tosDisk : %s", err.Error())
					continue
				}

				quantity, err := resource.ParseQuantity(string(tosDiskVolumeSource.Capability))
				if err != nil {
					logrus.Warnf("failed to parse quantity: %s", err.Error())
					continue
				}

				tosDiskStorages = append(tosDiskStorages, &release.ReleaseResourceStorage{
					Name:         tosDiskVolumeSource.Name,
					Type:         release.TosDiskPodStorageType,
					Size:         quantity.Value() / util.K8sResourceStorageScale,
					StorageClass: tosDiskVolumeSource.StorageType,
				})
			}
		}
	}
	return
}

func buildPvcStorages(pvcs []v1.PersistentVolumeClaim) (pvcStorages []*release.ReleaseResourceStorage) {
	pvcStorages = []*release.ReleaseResourceStorage{}
	for _, pvc := range pvcs {
		pvcStorages = append(pvcStorages, buildPvcStorage(pvc))
	}
	return
}

func buildPvcStorage(pvc v1.PersistentVolumeClaim) *release.ReleaseResourceStorage {
	pvcStorage := &release.ReleaseResourceStorage{
		Name: pvc.Name,
		Type: release.PvcPodStorageType,
	}
	quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	pvcStorage.Size = quantity.Value() / util.K8sResourceStorageScale
	if pvc.Spec.StorageClassName != nil {
		pvcStorage.StorageClass = *pvc.Spec.StorageClassName
	} else if len(pvc.Annotations) > 0 {
		pvcStorage.StorageClass = pvc.Annotations["volume.beta.kubernetes.io/storage-class"]
	}
	return pvcStorage
}
