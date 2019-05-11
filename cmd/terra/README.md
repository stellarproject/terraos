# Terra

![terra](terra.png)

Terra handles node management for the Stellar Project.  Terra provides node level cluster management
to enable building and maintaining cluster nodes.  One or more nodes form a Terra cluster.  A manifest
list is applied to one of the nodes and the config is replicated among the cluster.  Each node
then applies the manifest list.

# Getting Started
To build, run `make`.  This will create binaries in `bin/`.

# Example
The following example will show basic Terra usage.  First we will launch the Terra agent:

```
$> terra
```

View the current cluster nodes:

```
$> tctl cluster nodes
ID                  ADDRESS             LABELS
dev                 127.0.0.1:9005
```

Create a simple manifest list as `simple.json`:

```
{
  "manifests": [
    {
      "node_id": "",
      "labels": {},
      "assemblies": [
        {
          "image": "docker.io/ehazlett/terra-simple:latest"
        }
      ]
    }
  ]
}
```

Apply the manifest list to the cluster:

```
$> tctl manifest apply simple.json
```

View the current cluster manifest list:

```
$> tctl manifest list
- NodeID:
  Assemblies:
    - Image: docker.io/ehazlett/terra-simple:latest

```

Watch the manifest apply:

```
INFO[0000] starting terra agent on 127.0.0.1:9005
INFO[0001] updated manifest list                         updated="2018-12-01 00:54:39.869278759 -0500 EST m=+1.899460685"
INFO[0001] applying assembly                             image="docker.io/ehazlett/terra-simple:latest"
INFO[0001] assembly applied successfully                 assembly="docker.io/ehazlett/terra-simple:latest"
INFO[0001] apply complete
```

You can now add more nodes and they will automatically receive the replicated manifest list and apply the manifest list.


[Photo](https://www.pexels.com/photo/astronomy-atmosphere-earth-exploration-220201/)
