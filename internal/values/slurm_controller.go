package values

import (
	slurmv1 "nebius.ai/slurm-operator/api/v1"
	"nebius.ai/slurm-operator/internal/consts"
	"nebius.ai/slurm-operator/internal/naming"
)

// SlurmController contains the data needed to deploy and reconcile the Slurm Controllers
type SlurmController struct {
	slurmv1.SlurmNode

	ContainerSlurmctld Container
	ContainerMunge     Container

	Service     Service
	StatefulSet StatefulSet

	VolumeSpool slurmv1.NodeVolume
	VolumeJail  slurmv1.NodeVolume
}

func buildSlurmControllerFrom(clusterName string, controller *slurmv1.SlurmNodeController) SlurmController {
	return SlurmController{
		SlurmNode: *controller.SlurmNode.DeepCopy(),
		ContainerSlurmctld: buildContainerFrom(
			controller.Slurmctld,
			consts.ContainerNameSlurmctld,
		),
		ContainerMunge: buildContainerFrom(
			controller.Munge,
			consts.ContainerNameMunge,
		),
		Service: buildServiceFrom(naming.BuildServiceName(consts.ComponentTypeController, clusterName)),
		StatefulSet: buildStatefulSetFrom(
			naming.BuildStatefulSetName(consts.ComponentTypeController, clusterName),
			controller.SlurmNode.Size,
		),
		VolumeSpool: *controller.Volumes.Spool.DeepCopy(),
		VolumeJail:  *controller.Volumes.Jail.DeepCopy(),
	}
}
