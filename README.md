# fallernetes

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/fallernetes)](https://artifacthub.io/packages/search?repo=fallernetes)

The fallernetes operator was inspired by @UnfamousThomas's [thesis](https://github.com/UnfamousThomas/thesis-initial). It is a simple operator for managing the
shutdown cycle of game servers in Kubernetes.

It borrows some structure and architecture from Agones, but is less focused on server allocation, and more
on how to block deletion.

## Description
As said in the previous paragraph, it is a Kubernetes operator. To get started, visit the [documentation](https://mirrorstudios.github.io/fallernetes-documentation/).


## Contributing

### Developing

To get started with developing the software, you need:
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Structure
This project has a few different modules:
- Operator: Handles the actual operator logic in Kubernetes.
- Sidecar: Is injected alongside the game servers to pods to store deletion states.
- Service: A small utility for managing gameservers through a REST API more easily.
