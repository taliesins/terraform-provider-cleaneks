Clean EKS Terraform Provider
=========================

#This is beta code. Working on adding acceptance tests so that I can mark it as release quality.

- [Website](https://github.com/taliesins/terraform-provider-hyperv)
- [Releases](https://github.com/taliesins/terraform-provider-hyperv/releases)
- [Documentation](https://registry.terraform.io/providers/taliesins/hyperv/latest/docs)
- [Issues](https://github.com/taliesins/terraform-provider-hyperv/issues)

![Hashi Logo](https://cdn.rawgit.com/taliesins/terraform-provider-hyperv/master/website/logo-hashicorp.svg "Hashi Logo")
![EKS Logo](https://cdn.rawgit.com/taliesins/terraform-provider-hyperv/master/website/windows-server-2016-logo.svg "EKS Logo")

Introduction
------------
When an EKS cluster is created by default AWS CNI, Kube Proxy and CoreDNS are configured for the cluster. This is
done to make the process of using EKS simplier. When using IaC tools this causes problems, as our desired state is
to remove created resources. This provider aims to solve that problem.

In more advanced scenario's you might want to replace AWS CNI that is used with something like Cilium CNI. To get 
the full power of Cilium we would want to replace instead of chain Cilium on top of AWS CNI and using Kube Proxy.

The other scenario where this provider is useful is if you want to manage these components yourself with something
like Helm. A pattern that is often followed is called App of Apps, where there is an umbrella Helm chart that calls
all the other helm charts required. Mixing this with a Kubernetes native deployment system like ArgoCD allows a
very easy way to administer a Kubernetes cluster using GitOps. 

For example using ArgoCD application CRD to define the core App of Apps for a cluster. So to update CoreDNS you 
would update the Helm Chart version in Git and ArgoCD would detect the change and roll out the new version of 
CoreDNS. It could also be used to override Helm Chart values for example to configure automatic scaling of CoreDNS 
which is **not enabled** by default by AWS.

Features
------------
- Remove AWS CNI
- Remove Kube Proxy
- Import CoreDNS deployment into Helm and remove AWS component label 
- Import CoreDNS service into Helm and remove AWS component label

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 1.55.x
-	[Go](https://golang.org/doc/install) 1.22 (to build the provider plugin)
-   EKS Cluster
-   Kubernetes token that allows management of core services

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/taliesins/terraform-provider-cleaneks`

```sh
$ mkdir -p $GOPATH/src/github.com/taliesins; cd $GOPATH/src/github.com/taliesins
$ git clone https://github.com/taliesins/terraform-provider-cleaneks.git
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/taliesins/terraform-provider-cleaneks
$ make build
```

Using the provider
----------------------
## Fill in for each provider

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.22+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

You should also use the terraform [documentation](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider#prepare-terraform-for-local-provider-install) to setup the terraform environment correctly so that you can use your locally compiled version.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-cleaneks
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```

Debugging the Provider
----------------------

To set Terraform log level:
```
set TF_LOG=TRACE
```
