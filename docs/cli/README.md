CLI Documentation
=================

Kolide Fleet provides a server which allows you to manage and orchestrate an osquery deployment across of a set of workstations and servers. For certain use-cases, it makes sense to maintain the configuration and data of an osquery deployment in source-controlled files. It is also desirable to be able to manage these files with a familiar command-line tool. To facilitate this, Kolide Fleet includes a `fleetctl` CLI for managing osquery fleets in this way.

For more information, see:

- [Documentation on the file format](./file-format.md)
- [The setup guide for new CLI users](./setup-guide.md)

## Inspiration

Inspiration for the `fleetctl` command-line experience as well as the file format has been principally derived from the [Kubernetes](https://kubernetes.io/) container orchestration tool. This is for a few reasons:

- Format Familiarity: At Kolide, we love Kubernetes and we think it is the future of production infrastructure management. We believe that many of the people that use this interface to manage Fleet will also be Kubernetes operators. By using a familiar command-line interface and file format, the cognitive overhead can be reduced since the operator is already familiar with how these tools work and behave.
- Established Best Practices: Kubernetes deployments can easily become very complex. Because of this, Kubernetes operators have an established set of best practices that they often follow when writing and maintaining config files. Some of these best practices and tips are documented on the [official Kubernetes website](https://kubernetes.io/docs/concepts/configuration/overview/#general-config-tips) and some are documented by [the community](https://www.mirantis.com/blog/introduction-to-yaml-creating-a-kubernetes-deployment/). Since the file format and workflow is so similar, we can re-use these best practices when managing Fleet configurations.

### `fleetctl` - The CLI

The `fleetctl` tool is heavily inspired by the [`kubectl`](https://kubernetes.io/docs/user-guide/kubectl-overview/) tool. If you are familiar with `kubectl`, this will all feel very familiar to you. If not, some further explanation would likely be helpful.

Fleet exposes the aspects of an osquery deployment as a set of "objects". Objects may be a query, a pack, a set of configuration options, etc. The documentation for [Declarative Management of Kubernetes Objects Using Configuration Files](https://kubernetes.io/docs/tutorials/object-management-kubectl/declarative-object-management-configuration/) says the following about the object lifecycle:

> Objects can be created, updated, and deleted by storing multiple object configuration files in a directory and using `kubectl apply` to recursively create and update those objects as needed.

Similarly, Fleet objects can be created, updated, and deleted by storing multiple object configuration files in a directory and using `fleetctl apply` to recursively create and update those objects as needed.

### Using goquery with `fleetctl`

Fleet and `fleetctl` have built in support for [goquery](https://github.com/AbGuthrie/goquery).

Use `fleetctl goquery` to open up the goquery console. When used with Fleet, goquery can connect using either a hostname or UUID.

```
$ ./build/fleetctl get hosts
+--------------------------------------+--------------+----------+---------+
|                 UUID                 |   HOSTNAME   | PLATFORM | STATUS  |
+--------------------------------------+--------------+----------+---------+
| 192343D5-0000-0000-B85B-58F656BED4C7 | 6523f89187f8 | centos   | online  |
+--------------------------------------+--------------+----------+---------+
$ ./build/fleetctl goquery
goquery> .connect 6523f89187f8
Verified Host(6523f89187f8) Exists.
.
goquery | 6523f89187f8:> .query select unix_time from time
...
------------------------------
| host_hostname | unix_time  |
------------------------------
| 6523f89187f8  | 1579842569 |
------------------------------
goquery | 6523f89187f8:>
```
