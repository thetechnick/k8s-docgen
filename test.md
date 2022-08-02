## `package-operator.run/v1alpha1`

Package v1alpha1 contains API Schema definitions for the dhcp v1alpha1 API group

* [ClusterObjectSet](#clusterobjectset)
* [ObjectSet](#objectset)


### ClusterObjectSet

ClusterObjectSet reconciles a collection of objects across ordered phases and aggregates their status.


**Example**

```yaml
apiVersion: package-operator.run/v1alpha1
kind: ClusterObjectSet
metadata:
  name: example
spec:
  availabilityProbes:
  - probes:
    - condition:
        status: "True"
        type: Available
      fieldsEqual:
        fieldA: .spec.fieldA
        fieldB: .status.fieldB
    selector:
      kind:
        group: apps
        kind: Deployment
  lifecycleState: Active
  pausedFor:
  - group: apps
    kind: Deployment
    name: example-deployment
  phases:
  - name: lorem
    objects:
    - object:
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: example-deployment
status:
  phase: Pending

```


| Field | Description |
| ----- | ----------- |
| `metadata` <br>metav1.ObjectMeta |  |
| `spec` <br><a href="#clusterobjectsetspec">ClusterObjectSetSpec</a> |  |
| `status` <br><a href="#clusterobjectsetstatus">ClusterObjectSetStatus</a> |  |


### ObjectSet

ObjectSet reconciles a collection of objects across ordered phases and aggregates their status.


**Example**

```yaml
apiVersion: package-operator.run/v1alpha1
kind: ObjectSet
metadata:
  name: example
  namespace: default
spec:
  availabilityProbes:
  - probes:
    - condition:
        status: "True"
        type: Available
      fieldsEqual:
        fieldA: .spec.fieldA
        fieldB: .status.fieldB
    selector:
      kind:
        group: apps
        kind: Deployment
  lifecycleState: Active
  pausedFor:
  - group: apps
    kind: Deployment
    name: example-deployment
  phases:
  - name: ipsum
    objects:
    - object:
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: example-deployment
status:
  phase:Pending: null

```


| Field | Description |
| ----- | ----------- |
| `metadata` <br>metav1.ObjectMeta |  |
| `spec` <br><a href="#objectsetspec">ObjectSetSpec</a> |  |
| `status` <br><a href="#objectsetstatus">ObjectSetStatus</a> |  |




### ClusterObjectSetSpec

ClusterObjectSetSpec defines the desired state of a ClusterObjectSet.

| Field | Description |
| ----- | ----------- |
| `lifecycleState` <br><a href="#objectsetlifecyclestate">ObjectSetLifecycleState</a> | Specifies the lifecycle state of the ObjectSet. |
| `pausedFor` <br><a href="#objectsetpausedobject">[]ObjectSetPausedObject</a> | Pause reconciliation of specific objects, while still reporting status. |
| `phases` <b>required</b><br><a href="#objectsettemplatephase">[]ObjectSetTemplatePhase</a> | Reconcile phase configuration for a ObjectSet.<br>Phases will be reconciled in order and the contained objects checked<br>against given probes before continuing with the next phase. |
| `availabilityProbes` <b>required</b><br><a href="#objectsetprobe">[]ObjectSetProbe</a> | Availability Probes check objects that are part of the package.<br>All probes need to succeed for a package to be considered Available.<br>Failing probes will prevent the reconciliation of objects in later phases. |


### ClusterObjectSetStatus

ClusterObjectSetStatus defines the observed state of a ClusterObjectSet.

| Field | Description |
| ----- | ----------- |
| `conditions` <br>[]metav1.Condition | Conditions is a list of status conditions ths object is in. |
| `phase` <br><a href="#objectsetstatusphase">ObjectSetStatusPhase</a> | Deprecated: This field is not part of any API contract<br>it will go away as soon as kubectl can print conditions!<br>Human readable status - please use .Conditions from code |
| `pausedFor` <br><a href="#objectsetpausedobject">[]ObjectSetPausedObject</a> | List of objects, the controller has paused reconciliation on. |


### ObjectSetObject

An object that is part of the phase of an ObjectSet.

| Field | Description |
| ----- | ----------- |
| `object` <b>required</b><br>runtime.RawExtension |  |


### ObjectSetPausedObject

Specifies that the reconciliation of a specific object should be paused.

| Field | Description |
| ----- | ----------- |
| `kind` <b>required</b><br><a href="#string">string</a> | Object Kind. |
| `group` <b>required</b><br><a href="#string">string</a> | Object Group. |
| `name` <b>required</b><br><a href="#string">string</a> | Object Name. |


### ObjectSetProbe

ObjectSetProbe define how ObjectSets check their children for their status.

| Field | Description |
| ----- | ----------- |
| `probes` <b>required</b><br><a href="#probe">[]Probe</a> | Probe configuration parameters. |
| `selector` <b>required</b><br><a href="#probeselector">ProbeSelector</a> | Selector specifies which objects this probe should target. |


### ObjectSetSpec

ObjectSetSpec defines the desired state of a ObjectSet.

| Field | Description |
| ----- | ----------- |
| `lifecycleState` <br><a href="#objectsetlifecyclestate">ObjectSetLifecycleState</a> | Specifies the lifecycle state of the ObjectSet. |
| `pausedFor` <br><a href="#objectsetpausedobject">[]ObjectSetPausedObject</a> | Pause reconciliation of specific objects, while still reporting status. |
| `phases` <b>required</b><br><a href="#objectsettemplatephase">[]ObjectSetTemplatePhase</a> | Reconcile phase configuration for a ObjectSet.<br>Phases will be reconciled in order and the contained objects checked<br>against given probes before continuing with the next phase. |
| `availabilityProbes` <b>required</b><br><a href="#objectsetprobe">[]ObjectSetProbe</a> | Availability Probes check objects that are part of the package.<br>All probes need to succeed for a package to be considered Available.<br>Failing probes will prevent the reconciliation of objects in later phases. |


### ObjectSetStatus

ObjectSetStatus defines the observed state of a ObjectSet.

| Field | Description |
| ----- | ----------- |
| `conditions` <br>[]metav1.Condition | Conditions is a list of status conditions ths object is in. |
| `phase` <br><a href="#objectsetstatusphase">ObjectSetStatusPhase</a> | Deprecated: This field is not part of any API contract<br>it will go away as soon as kubectl can print conditions!<br>Human readable status - please use .Conditions from code |
| `pausedFor` <br><a href="#objectsetpausedobject">[]ObjectSetPausedObject</a> | List of objects, the controller has paused reconciliation on. |


### ObjectSetTemplatePhase

ObjectSet reconcile phase.

| Field | Description |
| ----- | ----------- |
| `name` <b>required</b><br><a href="#string">string</a> | Name of the reconcile phase, must be unique within a ObjectSet. |
| `objects` <b>required</b><br><a href="#objectsetobject">[]ObjectSetObject</a> | Objects belonging to this phase. |


### ObjectSetTemplateSpec

ObjectSet specification.

| Field | Description |
| ----- | ----------- |
| `phases` <b>required</b><br><a href="#objectsettemplatephase">[]ObjectSetTemplatePhase</a> | Reconcile phase configuration for a ObjectSet.<br>Phases will be reconciled in order and the contained objects checked<br>against given probes before continuing with the next phase. |
| `availabilityProbes` <b>required</b><br><a href="#objectsetprobe">[]ObjectSetProbe</a> | Availability Probes check objects that are part of the package.<br>All probes need to succeed for a package to be considered Available.<br>Failing probes will prevent the reconciliation of objects in later phases. |


### PackageProbeKindSpec

Kind package probe parameters.

| Field | Description |
| ----- | ----------- |
| `group` <b>required</b><br><a href="#string">string</a> | Object Group to apply a probe to. |
| `kind` <b>required</b><br><a href="#string">string</a> | Object Kind to apply a probe to. |


### Probe

Defines probe parameters to check parts of a package.

| Field | Description |
| ----- | ----------- |
| `condition` <br><a href="#probeconditionspec">ProbeConditionSpec</a> |  |
| `fieldsEqual` <br><a href="#probefieldsequalspec">ProbeFieldsEqualSpec</a> |  |


### ProbeConditionSpec

Condition Probe parameters.

| Field | Description |
| ----- | ----------- |
| `type` <b>required</b><br><a href="#string">string</a> | Condition type to probe for. |
| `status` <b>required</b><br><a href="#string">string</a> | Condition status to probe for. |


### ProbeFieldsEqualSpec

Compares two fields specified by JSON Paths.

| Field | Description |
| ----- | ----------- |
| `fieldA` <b>required</b><br><a href="#string">string</a> |  |
| `fieldB` <b>required</b><br><a href="#string">string</a> |  |


### ProbeSelector



| Field | Description |
| ----- | ----------- |
| `kind` <br><a href="#packageprobekindspec">PackageProbeKindSpec</a> | Kind specific configuration parameters. Only present if Type = Kind. |