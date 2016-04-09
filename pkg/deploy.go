package pkg

import (
	"fmt"
	"time"

	supergiant "github.com/supergiant/supergiant/client"
)

func Deploy(appName *string, componentName *string) error {

	fmt.Println("Starting Supergiant Client on http://api-public.supergiant.svc.cluster.local:8080/v0")

	sg := supergiant.New("http://api-public.supergiant.svc.cluster.local:8080/v0", "", "", true)

	fmt.Println("Loading app")

	app, err := sg.Apps().Get(appName)
	if err != nil {
		return err
	}

	fmt.Println("Loading component")

	component, err := app.Components().Get(componentName)
	if err != nil {
		return err
	}

	var currentRelease *supergiant.ReleaseResource
	if component.CurrentReleaseTimestamp != nil {
		currentRelease, err = component.CurrentRelease()
		if err != nil {
			return err
		}
	}

	targetRelease, err := component.TargetRelease()
	if err != nil {
		return err
	}

	targetList, err := targetRelease.Instances().List()
	if err != nil {
		return err
	}
	targetInstances := targetList.Items

	if currentRelease == nil { // first release
		for _, instance := range targetInstances {
			if err = instance.Start(); err != nil {
				return err
			}
		}
		for _, instance := range targetInstances {
			if err = instance.WaitForStarted(); err != nil {
				return err
			}
		}
		return nil
	}

	currentList, err := currentRelease.Instances().List()
	if err != nil {
		return err
	}
	currentInstances := currentList.Items

	es := newEsClient(component.Addresses.External[0].Address)

	if err := es.waitForShardRecovery(); err != nil {
		return err
	}

	// remove instances
	if currentRelease.InstanceCount > targetRelease.InstanceCount {

		// reduceReplicas()

		if err := es.setMinMasterNodes(targetRelease.InstanceCount/2 + 1); err != nil {
			return err
		}

		instancesRemoving := currentRelease.InstanceCount - targetRelease.InstanceCount

		var awarenessAttrs []string
		for i := instancesRemoving; i > 0; i-- {
			id := *currentInstances[len(currentInstances)-i].ID
			attr := "n" + id
			awarenessAttrs = append(awarenessAttrs, attr)
		}
		if err := es.setAwarenessAttrs(awarenessAttrs); err != nil {
			return err
		}
		if err := es.waitForShardRecovery(); err != nil {
			return err
		}

		for _, instance := range currentInstances[len(currentInstances)-instancesRemoving:] {

			if err := instance.Stop(); err != nil {
				return err
			}

			if err := es.waitForShardRecovery(); err != nil {
				return err
			}

		}

		if err := es.clearAwarenessAttrs(); err != nil {
			return err
		}

	}

	if err := es.disableShardRebalancing(); err != nil {
		return err
	}

	// update instances
	// NOTE instance count should be the same at this point between releases
	for i := 0; i < currentRelease.InstanceCount; i++ {
		currentInstance := currentInstances[i]
		targetInstance := targetInstances[i]

		if err := es.disableShardAllocation(); err != nil {
			return err
		}

		if err := es.flushTranslog(); err != nil {
			return err
		}

		currentInstance.Stop()
		currentInstance.WaitForStopped()

		targetInstance.Start()
		targetInstance.WaitForStarted()

		// assertNodeConnected()
		time.Sleep(30 * time.Second)

		if err := es.enableShardAllocation(); err != nil {
			return err
		}

		if err := es.waitForShardRecovery(); err != nil {
			return err
		}
	}

	if err := es.enableShardRebalancing(); err != nil {
		return err
	}

	// add new instances
	if currentRelease.InstanceCount < targetRelease.InstanceCount {
		instancesAdding := targetRelease.InstanceCount - currentRelease.InstanceCount
		newInstances := targetInstances[len(targetInstances)-instancesAdding:]
		for _, instance := range newInstances {
			if err := instance.Start(); err != nil {
				return err
			}
		}
		for _, instance := range newInstances {
			if err := instance.WaitForStarted(); err != nil {
				return err
			}
		}
	}

	return nil
}
