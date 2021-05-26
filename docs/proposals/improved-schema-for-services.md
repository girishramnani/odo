
# Improved schema for operator backed services

## Abstract
This proposal is about improving the experience for a user of the operator backed services. Currently a user doesn't know exaustively which fields are available in a CR as we depend on optional metadata present in CRDs, to improve this we are considering using the metadata available from the cluster about the CRDs.
Getting that metadata from the cluster is challenging because a normal cluster user ( plain vanilla ) doesn't have access to that metadata so we are gonna follow a 3 tiered approach

- check if the user has access to the CRD, if so then use that.
- get the `swagger.json` from the openapi endpoint provided by kubernetes. a sample can be found here https://gist.github.com/girishramnani/f29949d5cb8c6547102776437e05ac19
- finally if we cannot get the openapi data as well then use the `ClusterServiceVersion` to get the parameters.

This change would affect multiple service commands and the changes are described briefly below -
- `odo catalog describe service` should include the metadata fields with description so the users can provide then when doing `odo service create`
- `odo catalog list service` shouldn't change much other then listing CRDs per operator
- `odo service create` should take flags and dynamically fill the CRD structs - metadata would be used for validation
- `odo service create --from-file` would be used by adapter team or whoever wants to provide the whole CRD themselves as a file. this feature is already present and working.

## Implementation plan

Note - Below I would mention `cluster` manytimes and that means both openshift and kuberenetes. If there is something specific to one or the other then I would mention it.

Below is the step wise plan for implementing this feature - 

### Getting the metadata

#### Admin Access/Access to CRD API path
First we would consider the scenerio where the user has admin access or privileges to access the CRD API to the cluster.
Cluster provides an api to get metadata for any CRD needed. URL - `<cluster-url>/api/kubernetes/apis/apiextensions.k8s.io/v1/customresourcedefinitions`. 

Sample output - 
https://gist.github.com/girishramnani/cbb4400e463efe89c13f1386e0788793

We care about the `openAPIV3Schema` as that would be used to build the CRD struct. 

#### Kuberenetes cluster Swagger has the schema

If the User doesn't have CRD access then we fetch the cluster's `swagger.json` from the endpoint `<cluster-url>/api/kubernetes/openapi/v2`. This is a very large document as it holds all the definitions present on the cluster.

So this needs to be cached and refreshed whenever a new operator is installed.


#### Fetch ClusterServiceVersion to generate the information

If none of the above approaches work, we finally fallback to getting the information from `ClusterServiceVersion`

We generate the description in this approach based on `spec.customresourcedefinitions` present in the ClusterServiceVersion.
This CRD def is different then whats provided by the `CustomResourceDefinition` as it has less information.

This is how one of the `customresourcedefinition` looks like ( Kafka from strimzi )
```
{
  "parameters": [
    {
      "description": "Kafka version",
      "displayName": "Version",
      "path": "kafka.version",
      "x-descriptors": [
        "urn:alm:descriptor:com.tectonic.ui:text"
      ]
    },
    {
      "description": "The desired number of Kafka brokers.",
      "displayName": "Kafka Brokers",
      "path": "kafka.replicas",
      "x-descriptors": [
        "urn:alm:descriptor:com.tectonic.ui:podCount"
      ]
    },
    {
      "description": "The type of storage used by Kafka brokers",
      "displayName": "Kafka storage",
      "path": "kafka.storage.type",
      "x-descriptors": [
        "urn:alm:descriptor:com.tectonic.ui:select:ephemeral",
        "urn:alm:descriptor:com.tectonic.ui:select:persistent-claim",
        "urn:alm:descriptor:com.tectonic.ui:select:jbod"
      ]
    },
    {
      "description": "Limits describes the minimum/maximum amount of compute resources required/allowed",
      "displayName": "Kafka Resource Requirements",
      "path": "kafka.resources",
      "x-descriptors": [
        "urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
      ]
    },
    {
      "description": "The desired number of Zookeeper nodes.",
      "displayName": "Zookeeper Nodes",
      "path": "zookeeper.replicas",
      "x-descriptors": [
        "urn:alm:descriptor:com.tectonic.ui:podCount"
      ]
    },
    {
      "description": "The type of storage used by Zookeeper nodes",
      "displayName": "Zookeeper storage",
      "path": "zookeeper.storage.type",
      "x-descriptors": [
        "urn:alm:descriptor:com.tectonic.ui:select:ephemeral",
        "urn:alm:descriptor:com.tectonic.ui:select:persistent-claim",
        "urn:alm:descriptor:com.tectonic.ui:select:jbod"
      ]
    },
    {
      "description": "Limits describes the minimum/maximum amount of compute resources required/allowed",
      "displayName": "Zookeeper Resource Requirements",
      "path": "zookeeper.resources",
      "x-descriptors": [
        "urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
      ]
    }
  ]
}
```

### Allowing user to set the parameters 

The current approach of sending map values via cli are as follows - 

- we would add a `-p` cobra param as a list
- each of the parameter would represent the key in a map and value in a map `e.g. -p "key"="value"`
- we would allow json path in the key for the user to specific any field in the map that they want to set e.g. `services[0].namespace`.
- we will pass all values as string. 

Sample - `odo service create servicebinding.coreos.io/Servicebinding/<version> -p  "services[0].envVarPrefix"="SVC" -p "services[0].namespace"="openshift"`

We would using https://github.com/tidwall/sjson to map the keys of the json.

this would yield into a map that looks like this

```

{
  "services":[
    {
      "envVarPrefix": "SVC",
      "namespace": "openshift"
    }
  ]
}

```

- odo also needs to have smart auto-complete which auto selects the version if the CR only has one version.


### Using the metadata to validate the user input

At this stage the user either has access to the `openAPIV3Schema` or `ClusterServiceVersion` and also the user has provided the service parameters they want to set as well.

#### We have access to openAPIV3Schema

<details open>
<summary> Below is an extract of an example `openAPIV3Schema` which would be used for explaination </summary>

```

{
  "application": {
    "type": "object",
    "required": [
      "group",
      "resource",
      "version"
    ],
    "properties": {
      "bindingPath": {
        "type": "object",
        "properties": {
          "containersPath": {
            "type": "string"
          },
          "secretPath": {
            "type": "string"
          }
        }
      },
      "group": {
        "type": "string"
      },
      "labelSelector": {
        "type": "object",
        "properties": {
          "matchExpressions": {
            "type": "array",
            "items": {
              "type": "object",
              "required": [
                "key",
                "operator"
              ],
              "properties": {
                "key": {
                  "type": "string"
                },
                "operator": {
                  "type": "string"
                },
                "values": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                }
              }
            }
          },
          "matchLabels": {
            "type": "object",
            "additionalProperties": {
              "type": "string"
            }
          }
        }
      },
      "name": {
        "type": "string"
      },
      "resource": {
        "type": "string"
      },
      "version": {
        "type": "string"
      }
    }
  },
  "customEnvVar": {
    "type": "array",
    "items": {
      "type": "object",
      "required": [
        "name"
      ],
      "properties": {
        "name": {
          "type": "string"
        },
        "value": {
          "type": "string"
        },
        "valueFrom": {
          "type": "object",
          "properties": {
            "configMapKeyRef": {
              "type": "object",
              "required": [
                "key"
              ],
              "properties": {
                "key": {
                  "type": "string"
                },
                "name": {
                  "type": "string"
                },
                "optional": {
                  "type": "boolean"
                }
              }
            },
            "fieldRef": {
              "type": "object",
              "required": [
                "fieldPath"
              ],
              "properties": {
                "apiVersion": {
                  "type": "string"
                },
                "fieldPath": {
                  "type": "string"
                }
              }
            },
            "resourceFieldRef": {
              "type": "object",
              "required": [
                "resource"
              ],
              "properties": {
                "containerName": {
                  "type": "string"
                },
                "divisor": {
                  "type": "string"
                },
                "resource": {
                  "type": "string"
                }
              }
            },
            "secretKeyRef": {
              "type": "object",
              "required": [
                "key"
              ],
              "properties": {
                "key": {
                  "type": "string"
                },
                "name": {
                  "type": "string"
                },
                "optional": {
                  "type": "boolean"
                }
              }
            }
          }
        }
      }
    }
  },
  "detectBindingResources": {
    "type": "boolean"
  },
  "envVarPrefix": {
    "type": "string"
  },
  "mountPathPrefix": {
    "type": "string"
  },
  "services": {
    "type": "array",
    "items": {
      "type": "object",
      "required": [
        "group",
        "kind",
        "version"
      ],
      "properties": {
        "envVarPrefix": {
          "type": "string"
        },
        "group": {
          "type": "string"
        },
        "id": {
          "type": "string"
        },
        "kind": {
          "type": "string"
        },
        "name": {
          "type": "string"
        },
        "namespace": {
          "type": "string"
        },
        "version": {
          "type": "string"
        }
      }
    }
  }
}

```

Note - removed `description` fields to make the sample concise. 
</details>



#### We have access to ClusterServiceVersion

The approach is to just go through the keys provided by the user against the ones present in the ClusterServiceVersion's CRDDescription. if the user has provided parameters which aren't present in the Description ( SpecDescriptors ) then we show an error with all the parameters that are incorrectly provided.

### "odo catalog list services"

- We need to show the different versions for the service when you execute `odo catalog list services`. it already shows the `Operators` and the respective CRs they provide for the user to `describe` on.

### "odo catalog describe service"

#### Approach where user has access to the CustomResourceDefinition
TODO

#### Approach where user only has access to ClusterServiceVersion 

`odo catalog service describe strimzi-cluster-operator.v0.21.1/Kafka -o json` would show the `ClusterServiceVersion`'s `CRDDescription` shown in the `Fetch ClusterServiceVersion to generate the information` section

For human readable output a non tablular approach would be used.

```
- FieldPath: zookeeper.resources
   DisplayName: Zookeeper Resource Requirements
   Description: Limits describes the minimum/maximum amount of compute resources required/allowed
- FieldPath: zookeeper.storage.type
   DisplayName: ....
   ...

```
 

### "odo service create"


#### JSON output


#### --input-file flag



