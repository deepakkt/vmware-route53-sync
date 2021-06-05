## VMWare Route53 Sync

### Intro

This module runs as a daemon syncs VMWare machines IP's to a Route53 subdomain

![Architecture](https://github.com/deepakkt/vmware-route53-sync/blob/main/images/architecture.png?raw=true)

### Why is this needed

* The idea is to reference a VMWare machine by its DNS name
* VMWare runs its VM's under hosts
* To optimize host usage, VMWare can move the VM's to a different host
* This can often cause the IP's to change, breaking the original DNS entry
* So this daemon monitors the VM's and syncs to R53

### How to use this?

This can be run stand alone, but it was originally designed to run like a Kubernetes controller