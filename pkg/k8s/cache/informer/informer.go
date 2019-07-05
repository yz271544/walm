package informer

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"time"
	listv1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/listers/core/v1"
	batchv1 "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/listers/apps/v1beta1"
	storagev1 "k8s.io/client-go/listers/storage/v1"
	releaseconfigexternalversions "transwarp/release-config/pkg/client/informers/externalversions"
	releaseconfigv1beta1 "transwarp/release-config/pkg/client/listers/transwarp/v1beta1"
	releaseconfigclientset "transwarp/release-config/pkg/client/clientset/versioned"
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/sirupsen/logrus"
	"WarpCloud/walm/pkg/k8s/converter"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/release"
	"k8s.io/apimachinery/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"WarpCloud/walm/pkg/k8s/utils"
	"k8s.io/client-go/tools/cache"
)

type Informer struct {
	factory                     informers.SharedInformerFactory
	deploymentLister            listv1beta1.DeploymentLister
	configMapLister             v1.ConfigMapLister
	daemonSetLister             listv1beta1.DaemonSetLister
	ingressLister               listv1beta1.IngressLister
	jobLister                   batchv1.JobLister
	podLister                   v1.PodLister
	secretLister                v1.SecretLister
	serviceLister               v1.ServiceLister
	statefulSetLister           v1beta1.StatefulSetLister
	nodeLister                  v1.NodeLister
	namespaceLister             v1.NamespaceLister
	resourceQuotaLister         v1.ResourceQuotaLister
	persistentVolumeClaimLister v1.PersistentVolumeClaimLister
	storageClassLister          storagev1.StorageClassLister
	endpointsLister             v1.EndpointsLister
	limitRangeLister            v1.LimitRangeLister

	releaseConifgFactory releaseconfigexternalversions.SharedInformerFactory
	releaseConfigLister  releaseconfigv1beta1.ReleaseConfigLister
}

func (informer *Informer) AddReleaseConfigHandler(OnAdd func(obj interface{}), OnUpdate func(oldObj, newObj interface{}), OnDelete func(obj interface{})) {
	handlerFuncs := &cache.ResourceEventHandlerFuncs{
		AddFunc:    OnAdd,
		UpdateFunc: OnUpdate,
		DeleteFunc: OnDelete,
	}
	informer.releaseConifgFactory.Transwarp().V1beta1().ReleaseConfigs().Informer().AddEventHandler(handlerFuncs)
}

func (informer *Informer) ListPersistentVolumeClaims(namespace string, labelSelectorStr string) ([]*k8s.PersistentVolumeClaim, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		logrus.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}
	resources, err := informer.persistentVolumeClaimLister.PersistentVolumeClaims(namespace).List(selector)
	if err != nil {
		logrus.Errorf("failed to list pvcs in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	pvcs := []*k8s.PersistentVolumeClaim{}
	for _, resource := range resources {
		pvc, err := converter.ConvertPvcFromK8s(resource)
		if err != nil {
			logrus.Errorf("failed to convert release config %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		pvcs = append(pvcs, pvc)
	}
	return pvcs, nil
}

func (informer *Informer) ListReleaseConfigs(namespace, labelSelectorStr string) ([]*k8s.ReleaseConfig, error) {
	selector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		logrus.Errorf("failed to parse label string %s : %s", labelSelectorStr, err.Error())
		return nil, err
	}
	resources, err := informer.releaseConfigLister.ReleaseConfigs(namespace).List(selector)
	if err != nil {
		logrus.Errorf("failed to list release configs in namespace %s : %s", namespace, err.Error())
		return nil, err
	}

	releaseConfigs := []*k8s.ReleaseConfig{}
	for _, resource := range resources {
		releaseConfig, err := converter.ConvertReleaseConfigFromK8s(resource)
		if err != nil {
			logrus.Errorf("failed to convert release config %s/%s: %s", resource.Namespace, resource.Name, err.Error())
			return nil, err
		}
		releaseConfigs = append(releaseConfigs, releaseConfig)
	}
	return releaseConfigs, nil
}

func (informer *Informer) listPods(namespace string, labelSelector *metav1.LabelSelector) ([]*corev1.Pod, error) {
	selector, err := utils.ConvertLabelSelectorToSelector(labelSelector)
	if err != nil {
		logrus.Errorf("failed to convert label selector : %s", err.Error())
		return nil, err
	}
	pods, err := informer.podLister.Pods(namespace).List(selector)
	if err != nil {
		logrus.Errorf("failed to list pods : %s", err.Error())
		return nil, err
	}
	return pods, nil
}

func (informer *Informer) getEndpoints(namespace, name string) (*corev1.Endpoints, error) {
	endpoints, err := informer.endpointsLister.Endpoints(namespace).Get(name)
	if err != nil {
		logrus.Errorf("failed to get endpoints : %s", err.Error())
		return nil, err
	}

	return endpoints, nil
}

func (informer *Informer) GetResourceSet(releaseResourceMetas []release.ReleaseResourceMeta) (resourceSet *k8s.ResourceSet, err error) {
	resourceSet = k8s.NewResourceSet()
	for _, resourceMeta := range releaseResourceMetas {
		resource, err := informer.GetResource(resourceMeta.Kind, resourceMeta.Namespace, resourceMeta.Name)
		// if resource is not found , do not return error, add it into resource set, so resource should not be nil
		if err != nil && !errorModel.IsNotFoundError(err) {
			return nil, err
		}
		resource.AddToResourceSet(resourceSet)
	}
	return
}

func (informer *Informer) GetResource(kind k8s.ResourceKind, namespace, name string) (k8s.Resource, error) {
	switch kind {
	case k8s.ReleaseConfigKind:
		return informer.getReleaseConfig(namespace, name)
	case k8s.ConfigMapKind:
		return informer.getConfigMap(namespace, name)
	case k8s.PersistentVolumeClaimKind:
		return informer.getPvc(namespace, name)
	case k8s.DaemonSetKind:
		return informer.getDaemonSet(namespace, name)
	case k8s.DeploymentKind:
		return informer.getDeployment(namespace, name)
	case k8s.ServiceKind:
		return informer.getService(namespace, name)
	case k8s.StatefulSetKind:
		return informer.getStatefulSet(namespace, name)
	case k8s.JobKind:
		return informer.getJob(namespace, name)
	case k8s.IngressKind:
		return informer.getIngress(namespace, name)
	case k8s.SecretKind:
		return informer.getSecret(namespace, name)
	default:
		return &k8s.DefaultResource{Meta: k8s.NewMeta(kind, namespace, name, k8s.NewState("Unknown", "NotSupportedKind", "Can not get this resource"))}, nil
	}
}

func (informer *Informer) start(stopCh <-chan struct{}) {
	informer.factory.Start(stopCh)
	informer.releaseConifgFactory.Start(stopCh)
}

func (informer *Informer) waitForCacheSync(stopCh <-chan struct{}) {
	informer.factory.WaitForCacheSync(stopCh)
	informer.releaseConifgFactory.WaitForCacheSync(stopCh)
}

func NewInformer(client *kubernetes.Clientset, releaseConfigClient *releaseconfigclientset.Clientset, resyncPeriod time.Duration, stopCh <-chan struct{}) (*Informer) {
	informer := &Informer{}
	informer.factory = informers.NewSharedInformerFactory(client, resyncPeriod)
	informer.deploymentLister = informer.factory.Extensions().V1beta1().Deployments().Lister()
	informer.configMapLister = informer.factory.Core().V1().ConfigMaps().Lister()
	informer.daemonSetLister = informer.factory.Extensions().V1beta1().DaemonSets().Lister()
	informer.ingressLister = informer.factory.Extensions().V1beta1().Ingresses().Lister()
	informer.jobLister = informer.factory.Batch().V1().Jobs().Lister()
	informer.podLister = informer.factory.Core().V1().Pods().Lister()
	informer.secretLister = informer.factory.Core().V1().Secrets().Lister()
	informer.serviceLister = informer.factory.Core().V1().Services().Lister()
	informer.statefulSetLister = informer.factory.Apps().V1beta1().StatefulSets().Lister()
	informer.nodeLister = informer.factory.Core().V1().Nodes().Lister()
	informer.namespaceLister = informer.factory.Core().V1().Namespaces().Lister()
	informer.resourceQuotaLister = informer.factory.Core().V1().ResourceQuotas().Lister()
	informer.persistentVolumeClaimLister = informer.factory.Core().V1().PersistentVolumeClaims().Lister()
	informer.storageClassLister = informer.factory.Storage().V1().StorageClasses().Lister()
	informer.endpointsLister = informer.factory.Core().V1().Endpoints().Lister()
	informer.limitRangeLister = informer.factory.Core().V1().LimitRanges().Lister()

	informer.releaseConifgFactory = releaseconfigexternalversions.NewSharedInformerFactory(releaseConfigClient, resyncPeriod)
	informer.releaseConfigLister = informer.releaseConifgFactory.Transwarp().V1beta1().ReleaseConfigs().Lister()

	informer.start(stopCh)
	informer.waitForCacheSync(stopCh)
	return informer
}
