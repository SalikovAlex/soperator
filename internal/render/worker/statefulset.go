package worker

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	slurmv1 "nebius.ai/slurm-operator/api/v1"
	"nebius.ai/slurm-operator/internal/consts"
	"nebius.ai/slurm-operator/internal/render/common"
	"nebius.ai/slurm-operator/internal/utils"
	"nebius.ai/slurm-operator/internal/values"
)

// RenderStatefulSet renders new [appsv1.StatefulSet] containing Slurm worker pods
func RenderStatefulSet(
	namespace,
	clusterName string,
	nodeFilters []slurmv1.K8sNodeFilter,
	secrets *slurmv1.Secrets,
	volumeSources []slurmv1.VolumeSource,
	worker *values.SlurmWorker,
) (appsv1.StatefulSet, error) {
	labels := common.RenderLabels(consts.ComponentTypeWorker, clusterName)
	matchLabels := common.RenderMatchLabels(consts.ComponentTypeWorker, clusterName)

	stsVersion, podVersion, err := common.GenerateVersionsAnnotationPlaceholders()
	if err != nil {
		return appsv1.StatefulSet{}, fmt.Errorf("generating versions annotation placeholders: %w", err)
	}

	nodeFilter := utils.MustGetBy(
		nodeFilters,
		worker.K8sNodeFilterName,
		func(f slurmv1.K8sNodeFilter) string { return f.Name },
	)

	volumes := []corev1.Volume{
		common.RenderVolumeSlurmConfigs(clusterName),
		common.RenderVolumeMungeKey(secrets.MungeKey.Name, secrets.MungeKey.Key),
		common.RenderVolumeMungeSocket(),
		renderVolumeNvidia(),
		renderVolumeBoot(),
	}
	var pvcTemplateSpecs []values.PVCTemplateSpec

	{
		if worker.VolumeSpool.VolumeSourceName != nil {
			volumes = append(
				volumes,
				common.RenderVolumeSpoolFromSource(
					consts.ComponentTypeWorker,
					volumeSources,
					*worker.VolumeSpool.VolumeSourceName,
				),
			)
		} else {
			pvcTemplateSpecs = append(
				pvcTemplateSpecs,
				values.PVCTemplateSpec{
					Name: common.RenderVolumeNameSpool(consts.ComponentTypeWorker),
					Spec: worker.VolumeSpool.VolumeClaimTemplateSpec,
				},
			)
		}
	}
	{
		if worker.VolumeJail.VolumeSourceName != nil {
			volumes = append(
				volumes,
				common.RenderVolumeJailFromSource(
					volumeSources,
					*worker.VolumeJail.VolumeSourceName,
				),
			)
		} else {
			pvcTemplateSpecs = append(
				pvcTemplateSpecs,
				values.PVCTemplateSpec{
					Name: consts.VolumeNameJail,
					Spec: worker.VolumeJail.VolumeClaimTemplateSpec,
				},
			)
		}
	}

	return appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      worker.StatefulSet.Name,
			Namespace: namespace,
			Labels:    labels,
			Annotations: map[string]string{
				consts.AnnotationVersions: string(stsVersion),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: worker.Service.Name,
			Replicas:    &worker.StatefulSet.Replicas,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					MaxUnavailable: &worker.StatefulSet.MaxUnavailable,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						consts.AnnotationVersions: string(podVersion),
						fmt.Sprintf(
							"%s/%s", consts.AnnotationApparmorKey, consts.ContainerNameSlurmd,
						): consts.AnnotationApparmorValueUnconfined,
						fmt.Sprintf(
							"%s/%s", consts.AnnotationApparmorKey, consts.ContainerNameMunge,
						): consts.AnnotationApparmorValueUnconfined,
					},
				},
				Spec: corev1.PodSpec{
					Affinity:     nodeFilter.Affinity,
					NodeSelector: nodeFilter.NodeSelector,
					Tolerations:  nodeFilter.Tolerations,
					InitContainers: []corev1.Container{
						renderContainerToolkitValidation(&worker.ContainerToolkitValidation),
					},
					Containers: []corev1.Container{
						renderContainerSlurmd(&worker.ContainerSlurmd, worker.MaxGPU),
						common.RenderContainerMunge(&worker.ContainerMunge),
					},
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: common.RenderVolumeClaimTemplates(
				consts.ComponentTypeWorker,
				namespace,
				clusterName,
				pvcTemplateSpecs,
			),
		},
	}, nil
}
