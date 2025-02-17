# dataplane-node-exporter-test



## Introduction

Functional testcases for [dataplane-node-exporter](https://github.com/openstack-k8s-operators/dataplane-node-exporter.git)

It will create a ovs-dpdk bridge together to dataplane-node-exporter. Traffic will be injected and statistics generated
by dataplane-node-exporter will be checked 

## How to use it

Steps:
1. Configure the environment. Run the script [config_test_environment.sh](https://github.com/openstack-k8s-operators/dataplane-node-exporter-test/blob/main/test/config_test_environment.sh)

   ```
   ./test/config_test_environment.sh 
   ```

   It will configure an ovs bridge and 2 namespaces connected to the ovs bridge

2. Run testcases. Run the script [run_tests.sh](https://github.com/openstack-k8s-operators/dataplane-node-exporter-test/blob/main/test/run_tests.sh)

   ```
   ./test/run_tests.sh
   ```

## Configuration files

Two configuration files are used:
1. [stats.csv](https://github.com/openstack-k8s-operators/dataplane-node-exporter-test/blob/main/test/stats.csv)
   Generated with the following command:

   ```
   ./dataplane-node-exporter --l csv > test/stats.csv
   ```
2. [stats_conf.csv](https://github.com/openstack-k8s-operators/dataplane-node-exporter-test/blob/main/test/stats_conf.csv)
   Contains a list of statistics to skip (due to some issues with those stats) or a new threshold to compare values for
   those statistics if there is a huge variability in their values


