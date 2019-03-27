package controller

import (
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	csiclientset "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	csisnapshotv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions/core/v1alpha1"
	snapshotsinformers "github.com/kubernetes-csi/external-snapshotter/pkg/client/informers/externalversions/volumesnapshot/v1alpha1"
	listers "kubevirt.io/containerized-data-importer/pkg/client/listers/core/v1alpha1"
	snapshotslisters "github.com/kubernetes-csi/external-snapshotter/pkg/client/listers/volumesnapshot/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/common"
	expectations "kubevirt.io/containerized-data-importer/pkg/expectations"
)

const (
	//AnnSmartCloneRequest sets our expected annotation for a CloneRequest
	AnnSmartCloneRequest = "k8s.io/SmartCloneRequest"
)

// SmartCloneController represents the CDI Clone Controller
type SmartCloneController struct {
	Controller
	cdiClientSet clientset.Interface
	csiClientSet csiclientset.Interface

	snapshotsLister  snapshotslisters.VolumeSnapshotLister
	dataVolumeLister listers.DataVolumeLister
	snapshotsSynced  cache.InformerSynced
	podExpectations  *expectations.UIDTrackingControllerExpectations
}

// NewSmartCloneController sets up a Smart Clone Controller, and returns a pointer to
// to the newly created Controller
func NewSmartCloneController(client kubernetes.Interface,
	cdiClientSet clientset.Interface,
	csiClientSet csiclientset.Interface,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	podInformer coreinformers.PodInformer,
	snapshotInformer snapshotsinformers.VolumeSnapshotInformer,
	dataVolumeInformer informers.DataVolumeInformer,
	image string,
	pullPolicy string,
	verbose string) *SmartCloneController {
	c := &SmartCloneController{
		Controller:       *NewController(client, pvcInformer, podInformer, image, pullPolicy, verbose),
		cdiClientSet:     cdiClientSet,
		csiClientSet:     csiClientSet,
		snapshotsLister:  snapshotInformer.Lister(),
		dataVolumeLister: dataVolumeInformer.Lister(),
		podExpectations:  expectations.NewUIDTrackingControllerExpectations(expectations.NewControllerExpectations()),
	}

	// Set up an event handler for when VolumeSnapshot resources change
	snapshotInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueVolumeSnapshot,
		UpdateFunc: func(old, new interface{}) {
			c.enqueueVolumeSnapshot(new)
		},
		DeleteFunc: c.enqueueVolumeSnapshot,
	})
	c.snapshotInformer = snapshotInformer.Informer()

	// Set up an event handler for when PVC resources change
	// handleObject function ensures we filter PVCs not created by this controller
	pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueuePvc,
		UpdateFunc: func(old, new interface{}) {
			c.enqueuePvc(new)
		},
		DeleteFunc: c.enqueuePvc,
	})

	return c
}

func (c *SmartCloneController) initializeExpectations(pvcKey string) error {
	return c.podExpectations.SetExpectations(pvcKey, 0, 0)
}

func (c *SmartCloneController) raisePodCreate(pvcKey string) {
	c.podExpectations.RaiseExpectations(pvcKey, 1, 0)
}

//ProcessNextItem ...
func (c *SmartCloneController) ProcessNextItem() bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	ns, name, err := cache.SplitMetaNamespaceKey(key.(string))
	if err != nil {
		runtime.HandleError(errors.Errorf("invalid resource key: %s", key))
		return false
	}

	pvc, err := c.pvcLister.PersistentVolumeClaims(ns).Get(name)
	snapshot, err := c.snapshotsLister.VolumeSnapshots(ns).Get(name)

	if pvc != nil && pvc.Status.Phase == corev1.ClaimBound {
		klog.V(3).Infof("!!! smart-clone ProcessNextItem pvc.Name: %s", pvc.Name)
		klog.V(3).Infof("!!! smart-clone ProcessNextItem pvc.ObjectMeta.Annotations: %s", pvc.ObjectMeta.Annotations[AnnSmartCloneRequest])
		if pvc.ObjectMeta.Annotations[AnnSmartCloneRequest] == "true" {
			snapshotName := pvc.Spec.DataSource.Name
			snapshotToDelete, err := c.snapshotsLister.VolumeSnapshots(ns).Get(snapshotName)
			if snapshotToDelete != nil {
				klog.V(3).Infof("!!! smart-clone ProcessNextItem snapshotName: %s", snapshotName)
				err = c.csiClientSet.VolumesnapshotV1alpha1().VolumeSnapshots(ns).Delete(snapshotName, &metav1.DeleteOptions{})
				if err != nil {
					klog.Errorf("error deleting snapshot for smart-clone %q: %v", key, err)
					return true
				}
				klog.V(3).Infof("!!! smart-clone ProcessNextItem snapshot deleted: %s", snapshotName)
				anno := map[string]string{}
				anno[AnnPodPhase] = string(corev1.PodSucceeded)
				lab := map[string]string{}
				if !checkIfLabelExists(pvc, common.CDILabelKey, common.CDILabelValue) {
					lab = map[string]string{common.CDILabelKey: common.CDILabelValue}
				}
				pvc, err = updatePVC(c.clientset, pvc, anno, lab)
				if err != nil {
					klog.Errorf("could not update pvc %q annotation and/or label", err)
					return false
				}
			}
		}
	} else if snapshot != nil {
		klog.V(3).Infof("!!! smart-clone ProcessNextItem key: %s", key)
		err := c.syncSnapshot(key.(string))
		if err != nil {
			klog.Errorf("error processing pvc %q: %v", key, err)
			return true
		}
	}
	return c.forgetKey(key, fmt.Sprintf("ProcessNextSnapshotItem: processing pvc %q completed", key))
}

func (c *SmartCloneController) syncSnapshot(key string) error {
	klog.V(3).Infof("!!! smart-clone syncSnapshot key: %s", key)
	snapshot, exists, err := c.snapshotFromKey(key)
	if err != nil {
		return err
	} else if !exists {
		c.podExpectations.DeleteExpectations(key)
	}

	if snapshot == nil {
		return nil
	}

	snapshotReadyToUse := snapshot.Status.ReadyToUse
	klog.V(3).Infof("Snapshot \"%s/%s\" - ReadyToUse: %t", snapshot.Namespace, snapshot.Name, snapshotReadyToUse)
	if !snapshotReadyToUse {
		return nil
	}
	return c.processSnapshotItem(snapshot)
}

// Create the cloning source and target pods based the pvc. The pvc is checked (again) to ensure that we are not already
// processing this pvc, which would result in multiple pods for the same pvc.
func (c *SmartCloneController) processSnapshotItem(snapshot *csisnapshotv1.VolumeSnapshot) error {
	// anno := map[string]string{}

	snapshotKey, err := cache.MetaNamespaceKeyFunc(snapshot)
	klog.V(3).Infof("!!! smart-clone processSnapshotItem snapshotKey: %s", snapshotKey)
	if err != nil {
		return err
	}

	// expectations prevent us from creating multiple pods. An expectation forces
	// us to observe a pod's creation in the cache.
	needsSync := c.podExpectations.SatisfiedExpectations(snapshotKey)

	// make sure not to reprocess a PVC that has already completed successfully,
	// even if the pod no longer exists
	phase, exists := snapshot.ObjectMeta.Annotations[AnnPodPhase]
	if exists && (phase == string(corev1.PodSucceeded)) {
		needsSync = false
	}

	if needsSync {
		err := c.initializeExpectations(snapshotKey)
		if err != nil {
			return err
		}
	}

	dataVolume, err := c.dataVolumeLister.DataVolumes(snapshot.Namespace).Get(snapshot.Name)
	klog.V(3).Infof("!!! smart-clone processSnapshotItem dataVolume name: %s", dataVolume.Name)
	if err != nil {
		return err
	}

	newPvc := newPvcFromSnapshot(snapshot, dataVolume)
	if newPvc == nil {
		klog.Errorf("error creating new pvc from snapshot object")
		return nil
	}
	c.podExpectations.ExpectCreations(snapshotKey, 1)
	pvc, err := c.clientset.CoreV1().PersistentVolumeClaims(snapshot.Namespace).Create(newPvc)
	if err != nil {
		c.podExpectations.CreationObserved(snapshotKey)
		return err
	}
	klog.V(3).Infof("!!! smart-clone processSnapshotItem pvc name: %s", pvc.Name)

	return nil
}

// return a VolumeSnapshot pointer based on the passed-in work queue key.
func (c *SmartCloneController) snapshotFromKey(key interface{}) (*csisnapshotv1.VolumeSnapshot, bool, error) {
	obj, exists, err := c.objFromKey(c.snapshotInformer, key)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get pvc object from key")
	} else if !exists {
		return nil, false, nil
	}

	snapshot, ok := obj.(*csisnapshotv1.VolumeSnapshot)
	if !ok {
		return nil, false, errors.New("Object not of type *v1.PersistentVolumeClaim")
	}
	return snapshot, true, nil
}

//Run is being called from cdi-controller (cmd)
func (c *SmartCloneController) Run(threadiness int, stopCh <-chan struct{}) error {
	c.Controller.run(threadiness, stopCh, c)
	return nil
}

func (c *SmartCloneController) runSnapshotWorkers() {
	for c.ProcessNextItem() {
		// empty
	}
}

// enqueueVolumeSnapshot takes a VolumeSnapshots resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DataVolume.
func (c *SmartCloneController) enqueueVolumeSnapshot(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *SmartCloneController) enqueuePvc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func newPvcFromSnapshot(snapshot *csisnapshotv1.VolumeSnapshot, dataVolume *cdiv1.DataVolume) *corev1.PersistentVolumeClaim {
	labels := map[string]string{
		"cdi-controller": snapshot.Name,
		"app":            "containerized-data-importer",
	}
	ownerRef := metav1.GetControllerOf(snapshot)
	if ownerRef == nil {
		return nil
	}
	annotations := make(map[string]string)
	annotations[AnnSmartCloneRequest] = "true"
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            snapshot.Name,
			Namespace:       snapshot.Namespace,
			Labels:          labels,
			Annotations:     annotations,
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			DataSource: &corev1.TypedLocalObjectReference{
				Name:     snapshot.Name,
				Kind:     "VolumeSnapshot",
				APIGroup: &csisnapshotv1.SchemeGroupVersion.Group,
			},
			VolumeMode:       dataVolume.Spec.PVC.VolumeMode,
			AccessModes:      dataVolume.Spec.PVC.AccessModes,
			StorageClassName: dataVolume.Spec.PVC.StorageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: dataVolume.Spec.PVC.Resources.Requests,
			},
		},
	}
}
