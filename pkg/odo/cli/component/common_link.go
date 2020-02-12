package component

import (
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/openshift/odo/pkg/component"
	componentlabels "github.com/openshift/odo/pkg/component/labels"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	svc "github.com/openshift/odo/pkg/service"
	"github.com/openshift/odo/pkg/util"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

type commonLinkOptions struct {
	wait             bool
	portStr          string
	secretName       string
	isTargetAService bool
	linkPort         int
	operationName    string
	suppliedName     string

	*genericclioptions.Context
}

func newCommonLinkOptions() *commonLinkOptions {
	return &commonLinkOptions{}
}

// Complete completes LinkOptions after they've been created
func (o *commonLinkOptions) complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.operationName = name
	suppliedName := args[0]
	o.suppliedName = suppliedName
	o.Context = genericclioptions.NewContextCreatingAppIfNeeded(cmd)

	port, err := strconv.Atoi(o.portStr)
	if err != nil {
		return err
	}
	o.linkPort = port

	svcExists, err := svc.SvcExists(o.Client, suppliedName, o.Application)
	if err != nil {
		// we consider this error to be non-terminal since it's entirely possible to use odo without the service catalog
		glog.V(4).Infof("Unable to determine if %s is a service. This most likely means the service catalog is not installed. Proceesing to only use components", suppliedName)
		svcExists = false
	}

	cmpExists, err := component.Exists(o.Client, suppliedName, o.Application)
	if err != nil {
		return fmt.Errorf("Unable to determine if component exists:\n%v", err)
	}

	if !cmpExists && !svcExists {
		// as there is a chance that the service or component doesn't exist yet
		log.Warningf("Neither a service nor a component named %s could be located. Links will be updated on `odo push` if the service/component exists.", suppliedName, o.operationName)
		return
	}

	o.isTargetAService = svcExists

	if svcExists {
		if cmpExists {
			glog.V(4).Infof("Both a service and component with name %s - assuming a(n) %s to the service is required", suppliedName, o.operationName)
		}

	}

	return nil
}

func (o *commonLinkOptions) run() (err error) {
	linkType := "Component"
	if o.isTargetAService {
		linkType = "Service"
	}

	if err != nil {
		return err
	}

	switch o.operationName {
	case "link":
		log.Successf("%s %s has been successfully linked to the component %s\n", linkType, o.suppliedName, o.Component())
	case "unlink":
		log.Successf("%s %s has been successfully unlinked from the component %s\n", linkType, o.suppliedName, o.Component())
	default:
		return fmt.Errorf("unknown operation %s", o.operationName)
	}

	secret, err := o.Client.GetSecret(o.secretName, o.Project)
	if err != nil {
		return err
	}

	if len(secret.Data) == 0 {
		log.Infof("There are no secret environment variables to expose within the %s service", o.suppliedName)
	} else {
		if o.operationName == "link" {
			log.Infof("The below secret environment variables were added to the '%s' component:\n", o.Component())
		} else {
			log.Infof("The below secret environment variables were removed from the '%s' component:\n", o.Component())
		}

		// Output the environment variables
		for i := range secret.Data {
			fmt.Printf("· %v\n", i)
		}

		// Retrieve the first variable to use as an example.
		// Have to use a range to access the map
		var exampleEnv string
		for i := range secret.Data {
			exampleEnv = i
			break
		}

		// Output what to do next if first linking...
		if o.operationName == "link" {
			log.Italicf(`
You can now access the environment variables from within the component pod, for example:
$%s is now available as a variable within component %s`, exampleEnv, o.Component())
		}
	}

	if o.wait {
		if err := o.waitForLinkToComplete(); err != nil {
			return err
		}
	}

	return
}

func (o *commonLinkOptions) waitForLinkToComplete() (err error) {
	labels := componentlabels.GetLabels(o.Component(), o.Application, true)
	selectorLabels, err := util.NamespaceOpenShiftObject(labels[componentlabels.ComponentLabel], labels["app"])
	if err != nil {
		return err
	}
	podSelector := fmt.Sprintf("deploymentconfig=%s", selectorLabels)

	// first wait for the pod to be pending (meaning that the deployment is being put into effect)
	// we need this intermediate wait because there is a change that the this point could be reached
	// without Openshift having had the time to launch the new deployment
	_, err = o.Client.WaitAndGetPod(podSelector, corev1.PodPending, "Waiting for component to redeploy")
	if err != nil {
		return err
	}

	// now wait for the pod to be running
	_, err = o.Client.WaitAndGetPod(podSelector, corev1.PodRunning, "Waiting for component to start")
	return err
}
