# Rules

`zda` uses rules to determine which capabilities to attach to a Zigbee node that is being Enumerated. This allows
flexibility to configure capabilities for devices that do not follow the Zigbee (dotdot) Cluster Library specification.

## Zigbee "Device" Primer

Common parlance might have a Zigbee light bulb described as a device, however a device has a very specific meaning
within Zigbee. This section hopes to help explain the concepts in situ.

* `node` - a piece of hardware that joins a Zigbee network, such as a light bulb
* `endpoint` - a logical grouping of functionality on a `node`, such as 'OnOff', 'Level' and 'Color', all together.
* `device` - a grouping of `endpoints` that are exposed to the user as a single piece of equipment that can be controlled.

The majority of equipment will tend to only have one of each of these, however some equipment will have multiple. For
example, for a period of time GLEDOPTO sold LED strip controllers that exposed RGB and CCT as separate devices.

`device`'s are identified by their `DeviceID` on the endpoints, it is up to implementing software to aggregate `endpoints` 
by their `DeviceID` and present these to the user as cohesive devices.

## Filter Execution

`zda` rules are executed against each `endpoint` individually, after this process they are then aggregated
into `device`'s
that are presented to the user. Because `devices` can be split across multiple `endpoints` when a filter executes it
can need access to all `endpoints` data.

### Filter Language

`zda` uses the [antonmedv/expr](https://github.com/expr-lang/expr) library to provide filter functionality,
its [language reference](https://github.com/expr-lang/expr/blob/master/docs/Language-Definition.md)
is fully supported. `zda` expects the output of the executed filter to be a boolean value, rule execution will fail if a
rule does not.

### Filter Input Object

The below example shows a zigbee `node` with multiple endpoints, note that the product information is only available on
`endpoint` 0, however the functionality is available on `endpoint 1`.

```json
{
  "Node": {
    "ManufacturerCode": 1000,
    "Type": "router" // Either "router" or "enddevice"
  },
  "Self": 1,
  "Product": {
    0: {
      "Name": "NEXUS-7",
      "Manufacturer": "Tyrell Corporation",
      "Version": "1.0.0",
      "Serial": "N7FAA52318"
    }
  },
  "Endpoint": {
    0: {
      "ID": 0,
      "ProfileID": 260,
      "DeviceID": 0,
      "InClusters": [
        0
      ],
      "OutClusters": []
    },
    1: {
      "ID": 1,
      "ProfileID": 260,
      "DeviceID": 0,
      "InClusters": [
        6,
        64512
      ],
      "OutClusters": []
    }
  }
}
```

Filters are executed for each `endpoint`, the input object passed in is nearly identical for each `endpoint` apart from
the `Self` value changing for each `endpoint` being evaluated.

This allows enumeration of `endpoints` to depend on data from other endpoints. For example:

```
any(values(Product), {.Manufacturer == "Tyrell Corporation"}) and 64512 in Endpoints[Self].InClusters
```

This essentially says if any product information has the "Tyrell Corporation" as manufacturer, and 64512 is in the
input cluster list of the `endpoint` currently being evaluated, then match.