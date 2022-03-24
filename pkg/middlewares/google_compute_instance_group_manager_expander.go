package middlewares

import (
	"github.com/snyk/driftctl/pkg/resource"
	"github.com/snyk/driftctl/pkg/resource/google"
)

type GoogleComputeInstanceGroupManagerInstances struct{}

// NewGoogleComputeInstanceGroupManagerInstances imports remote instance groups when they're managed by a managed instance group manager.
// Creating a "google_compute_instance_group_manager" resource via Terraform leads to having several unmanaged instance groups.
// This middleware adds remote instance groups to the state by matching them with managed instance group managers.
func NewGoogleComputeInstanceGroupManagerInstances() *GoogleComputeInstanceGroupManagerInstances {
	return &GoogleComputeInstanceGroupManagerInstances{}
}

func (a GoogleComputeInstanceGroupManagerInstances) Execute(remoteResources, resourcesFromState *[]*resource.Resource) error {
	var newStateResources []*resource.Resource

	instanceGroups := make([]*resource.Resource, 0)
	for _, remoteResource := range *remoteResources {
		// Ignore all resources other than google_compute_instance_group
		if remoteResource.ResourceType() != google.GoogleComputeInstanceGroupResourceType {
			continue
		}
		instanceGroups = append(instanceGroups, remoteResource)
	}

	for _, stateResource := range *resourcesFromState {
		newStateResources = append(newStateResources, stateResource)

		// Ignore all resources other than google_compute_instance_group_manager
		if stateResource.ResourceType() != google.GoogleComputeInstanceGroupManagerResourceType {
			continue
		}

		name := stateResource.Attributes().GetString("name")

		for _, group := range instanceGroups {
			// Import instance group in the state
			if n := group.Attributes().GetString("name"); n != nil && *n == *name {
				newStateResources = append(newStateResources, group)
			}
		}
	}

	*resourcesFromState = newStateResources

	return nil
}
