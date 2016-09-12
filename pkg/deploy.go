package pkg

import (
	"fmt"
	"time"

	supergiant "github.com/supergiant/supergiant/pkg/client"
	"github.com/supergiant/supergiant/pkg/model"
)

// Deploy will either setup a new Component, or apply changes to an existing.
func Deploy(sg *supergiant.Client, componentID *int64) error {
	includes := []string{
		"App.Kube.CloudAccount",
		"PrivateImageKeys",
		"CurrentRelease",
		"TargetRelease",
		"Instances.Volumes",
	}
	component := new(model.Component)
	if err := sg.Components.GetWithIncludes(componentID, component, includes); err != nil {
		return err
	}

	// If first deploy, concurrently start Instances and return
	if component.CurrentRelease == nil {
		for _, instance := range component.Instances {
			if instance.Started {
				continue
			}
			if err := sg.Instances.Start(instance); err != nil {
				return err
			}
		}
		for _, instance := range component.Instances {
			if err := sg.Instances.WaitForStarted(instance); err != nil {
				return err
			}
		}
		return nil
	}

	elasticsearch := newEsClient(component.Addresses.External[0].Address)

	// Wait for initial shard recovery before doing any restarts
	if err := elasticsearch.waitForShardRecovery(); err != nil {
		return err
	}
	// Update minimum master nodes in case out of sync
	if err := elasticsearch.setMinMasterNodes(component.TargetRelease.InstanceCount/2 + 1); err != nil {
		return err
	}

	// If we're removing instances
	if component.CurrentRelease.InstanceCount > component.TargetRelease.InstanceCount {
		instancesRemoving := component.CurrentRelease.InstanceCount - component.TargetRelease.InstanceCount

		// Set awareness attributes to move shards off of Instances that will be removed
		var awarenessAttrs []string
		for i := instancesRemoving; i > 0; i-- {
			ordinal := component.Instances[len(component.Instances)-i].Num
			attr := fmt.Sprintf("n%d", ordinal)
			awarenessAttrs = append(awarenessAttrs, attr)
		}
		if err := elasticsearch.setAwarenessAttrs(awarenessAttrs); err != nil {
			return err
		}
		// Be sure to clear the awareness attributes
		defer elasticsearch.clearAwarenessAttrs()

		// Wait for shards to move off of the marked Instances
		if err := elasticsearch.waitForShardRecovery(); err != nil {
			return err
		}
	}

	// Prevent shards from moving around during restarts
	if err := elasticsearch.disableShardRebalancing(); err != nil {
		return err
	}
	defer elasticsearch.enableShardRebalancing()

	for _, instance := range component.Instances {
		// Remove Instances
		if (instance.Num + 1) > component.TargetRelease.InstanceCount {
			if err := sg.Instances.Delete(instance.ID, instance); err != nil {
				return err
			}
			if err := sg.Instances.WaitForDeleted(instance); err != nil {
				return err
			}

			// Wait for shards to recover after removing Instances
			if err := elasticsearch.waitForShardRecovery(); err != nil {
				return err
			}
			continue
		}

		// Stop Instance if not using new Release
		if *instance.ReleaseID != *component.TargetReleaseID {

			// Prevent shard moving and flush to disk before restart
			if err := elasticsearch.disableShardAllocation(); err != nil {
				return err
			}
			if err := elasticsearch.flushTranslog(); err != nil {
				return err
			}

			if err := sg.Instances.Stop(instance); err != nil {
				return err
			}
			if err := sg.Instances.WaitForStopped(instance); err != nil {
				return err
			}
		}

		// Start Instance
		if instance.Started {
			continue
		}
		if err := sg.Instances.Start(instance); err != nil {
			return err
		}
		if err := sg.Instances.WaitForStarted(instance); err != nil {
			return err
		}

		time.Sleep(30 * time.Second)

		// After each restart, re-enable shard moving and wait for it to settle
		if err := elasticsearch.enableShardAllocation(); err != nil {
			return err
		}
		if err := elasticsearch.waitForShardRecovery(); err != nil {
			return err
		}
	}

	return nil
}
