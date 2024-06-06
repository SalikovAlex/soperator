package worker

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"nebius.ai/slurm-operator/internal/consts"
	"nebius.ai/slurm-operator/internal/render/common"
	"nebius.ai/slurm-operator/internal/values"
)

// renderContainerToolkitValidation renders init [corev1.Container] for toolkit validation
func renderContainerToolkitValidation(container *values.Container) corev1.Container {
	return corev1.Container{
		Name:            consts.ContainerNameToolkitValidation,
		Image:           container.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"sh",
			"-c",
		},
		Args: []string{
			fmt.Sprintf("until [ -f %s/validations/toolkit-ready ]; do", consts.VolumeMountPathNvidia),
			"echo 'waiting for nvidia container stack to be setup';",
			"sleep 5;",
			"done",
		},
		VolumeMounts: []corev1.VolumeMount{
			renderVolumeMountNvidia(),
		},
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		TerminationMessagePath:   "/dev/termination-log",
	}
}

// renderContainerSlurmd renders [corev1.Container] for slurmd
func renderContainerSlurmd(container *values.Container, maxGPU int32) corev1.Container {
	return corev1.Container{
		Name:            consts.ContainerNameSlurmd,
		Image:           container.Image,
		ImagePullPolicy: corev1.PullAlways, // TODO use digest and set to corev1.PullIfNotPresent
		Env: []corev1.EnvVar{
			{
				Name: "K8S_POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			}, {
				Name: "K8S_POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
		},
		Ports: []corev1.ContainerPort{{
			Name:          consts.ContainerNameSlurmd,
			ContainerPort: container.Port,
			Protocol:      corev1.ProtocolTCP,
		}},
		VolumeMounts: []corev1.VolumeMount{
			common.RenderVolumeMountSlurmConfigs(),
			common.RenderVolumeMountSpool(consts.ComponentTypeWorker, consts.SlurmdName),
			common.RenderVolumeMountJail(),
			common.RenderVolumeMountMungeSocket(),
			renderVolumeMountNvidia(),
			renderVolumeMountBoot(),
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt32(container.Port),
				},
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					consts.ContainerSecurityContextCapabilitySysAdmin,
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:              container.Resources.CPU,
				corev1.ResourceMemory:           container.Resources.Memory,
				corev1.ResourceEphemeralStorage: container.Resources.EphemeralStorage,
				consts.AnnotationMaxGPU:         resource.MustParse(string(maxGPU)),
			},
		},
	}
}
