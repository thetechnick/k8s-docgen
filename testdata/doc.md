## testdata



* [TestObject](#testobject)


### TestObject




**Example**

```yaml
apiVersion: testdata
kind: TestObject
metadata:
  name: example
  namespace: default
spec:
  empty: {}
  example: Test 123
  exampleObj:
    field1: apps/v1
    field2: false
  field1: amet
  field2: true
  field3:
  - consetetur
  field4: 42
  field5: 42
  field6: 42
  object:
    empty: {}
    example: Test 123
    field1: lorem
    field2: true
    field3:
    - ipsum
    field4: 42
    field5: 42
    field6: 42
  objects:
  - empty: {}
    example: Test 123
    field1: dolor
    field2: true
    field3:
    - sit
    field4: 42
    field5: 42
    field6: 42
  uid: 3490a790-05f8-4bd7-8333-1001c49fccd2

```


| Field | Description |
| ----- | ----------- |
| `metadata` <br>metav1.ObjectMeta |  |
| `spec` <b>required</b><br><a href="#testobjectspec">TestObjectSpec</a> |  |




---

### Object



| Field | Description |
| ----- | ----------- |
| `field1` <b>required</b><br>string |  |
| `field2` <br>bool |  |
| `field3` <b>required</b><br>[]string |  |
| `field4` <b>required</b><br>int |  |
| `field5` <b>required</b><br>int32 |  |
| `field6` <b>required</b><br>int64 |  |
| `empty` <b>required</b><br><a href="#emptyobject">EmptyObject</a> |  |
| `example` <b>required</b><br>string |  |


Used in:
* [TestObjectSpec](#testobjectspec)
* [TestObjectSpec](#testobjectspec)
* [TestObjectSpec](#testobjectspec)


### TestObjectSpec



| Field | Description |
| ----- | ----------- |
| `object` <b>required</b><br><a href="#object">Object</a> |  |
| `objects` <b>required</b><br><a href="#object">[]Object</a> |  |
| `exampleObj` <b>required</b><br><a href="#object">Object</a> |  |
| `uid` <b>required</b><br>types.UID |  |
| `field1` <b>required</b><br>string |  |
| `field2` <br>bool |  |
| `field3` <b>required</b><br>[]string |  |
| `field4` <b>required</b><br>int |  |
| `field5` <b>required</b><br>int32 |  |
| `field6` <b>required</b><br>int64 |  |
| `empty` <b>required</b><br><a href="#emptyobject">EmptyObject</a> |  |
| `example` <b>required</b><br>string |  |


Used in:
* [TestObject](#testobject)