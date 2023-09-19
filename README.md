# image-metadata-collector

# Contributing
We are looking forward to contributions. Take a look at our [Contribution Guidelines](CONTRIBUTING.md) before submitting Pull Requests.

# Responsible Disclosure and Security
The [SECURITY.md](SECURITY.md) includes information on responsible disclosure and security related topics like security patches.

# Development
## Image Collector
To perform integration tests for the image collector, you need a kind cluster:
```bash
cd test_actions/imagecollector
kind delete cluster; kind create cluster && ./setup.bash
```

# Legal Notice
The purpose of the ClusterImageScanner is not to replace the penetration testers or make them obsolete. We strongly recommend running extensive tests by experienced penetration testers on all your applications.
The ClusterImageScanner is to be used only for testing purpose of your running applications/containers. You need a written agreement of the organization of the _environment under scan_ to scan components with the ClusterScanner.

# Author Information
This project is developed by [Signal Iduna](https://www.signal-iduna.de) and [SDA SE](https://sda.se/). 
