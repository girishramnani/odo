package component

import (
	"fmt"

	"github.com/openshift/odo/pkg/component"
	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/pkg/errors"

	appCmd "github.com/openshift/odo/pkg/odo/cli/application"
	"github.com/openshift/odo/pkg/odo/util/completion"

	"github.com/openshift/odo/pkg/odo/util"
	ktemplates "k8s.io/kubernetes/pkg/kubectl/util/templates"

	"github.com/spf13/cobra"
)

// LinkRecommendedCommandName is the recommended link command name
const LinkRecommendedCommandName = "link"

var (
	linkExample = ktemplates.Examples(`# Link the current component to the 'my-postgresql' service
%[1]s my-postgresql

# Link component 'nodejs' to the 'my-postgresql' service
%[1]s my-postgresql --component nodejs

# Link current component to the 'backend' component (backend must have a single exposed port)
%[1]s backend

# Link component 'nodejs' to the 'backend' component
%[1]s backend --component nodejs

# Link current component to port 8080 of the 'backend' component (backend must have port 8080 exposed) 
%[1]s backend --port 8080`)

	linkLongDesc = `Link component to a service or component

If the source component is not provided, the current active component is assumed.
In both use cases, link adds the appropriate secret to the environment of the source component. 
The source component can then consume the entries of the secret as environment variables.

For example:

We have created a frontend application called 'frontend' using:
odo create nodejs frontend

We've also created a backend application called 'backend' with port 8080 exposed:
odo create nodejs backend --port 8080

We can now link the two applications:
odo link backend --component frontend

Now the frontend has 2 ENV variables it can use:
COMPONENT_BACKEND_HOST=backend-app
COMPONENT_BACKEND_PORT=8080

If you wish to use a database, we can use the Service Catalog and link it to our backend:
odo service create dh-postgresql-apb --plan dev -p postgresql_user=luke -p postgresql_password=secret
odo link dh-postgresql-apb

Now backend has 2 ENV variables it can use:
DB_USER=luke
DB_PASSWORD=secret`
)

// LinkOptions encapsulates the options for the odo link command
type LinkOptions struct {
	componentContext string
	*commonLinkOptions
}

// NewLinkOptions creates a new LinkOptions instance
func NewLinkOptions() *LinkOptions {
	options := LinkOptions{}
	options.commonLinkOptions = newCommonLinkOptions()
	return &options
}

// Complete completes LinkOptions after they've been created
func (o *LinkOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	return o.complete(name, cmd, args)

}

// Validate validates the LinkOptions based on completed values
func (o *LinkOptions) Validate() (err error) {

	alreadyLinkedSecretNames, err := component.GetComponentLinkedSecretNames(o.Client, o.Component(), o.Application)
	if err != nil {
		return err
	}
	for _, alreadyLinkedSecretName := range alreadyLinkedSecretNames {
		if alreadyLinkedSecretName == o.secretName {
			targetType := "component"
			if o.isTargetAService {
				targetType = "service"
			}
			return fmt.Errorf("Component %s has previously been linked to %s %s", o.Component(), targetType, o.suppliedName)
		}
	}
	return
}

// Run contains the logic for the odo link command
func (o *LinkOptions) Run() (err error) {
	if err := o.LocalConfigInfo.AddLink(o.suppliedName, o.Application, o.linkPort, o.isTargetAService); err != nil {
		return errors.Wrap(err, "error while adding link to the local config")
	}

	log.Successf("Added link for %v in the config", o.suppliedName)
	log.Italic("\nPlease use `odo push` command to create the link")
	return
}

// NewCmdLink implements the link odo command
func NewCmdLink(name, fullName string) *cobra.Command {
	o := NewLinkOptions()

	linkCmd := &cobra.Command{
		Use:         fmt.Sprintf("%s <service>  OR %s <component>", name, name),
		Short:       "Link component to a service or component",
		Long:        linkLongDesc,
		Example:     fmt.Sprintf(linkExample, fullName),
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{"command": "component"},
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	linkCmd.PersistentFlags().StringVar(&o.portStr, "port", "", "Port of the backend to which to link")

	linkCmd.SetUsageTemplate(util.CmdUsageTemplate)
	//Adding `--application` flag
	appCmd.AddApplicationFlag(linkCmd)
	//Adding context flag
	genericclioptions.AddContextFlag(linkCmd, &o.componentContext)

	completion.RegisterCommandHandler(linkCmd, completion.LinkCompletionHandler)

	return linkCmd
}
