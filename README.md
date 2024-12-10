# Dataplane Node Exporter

This is a prometheus exporter for dataplane (compute, network) nodes running
with OpenvSwitch. It supports the default linux kernel and userspace DPDK data
paths.

The Dataplane Node Exporter is distributed under the [Apache 2.0][license]
license.

[license]: https://spdx.org/licenses/Apache-2.0.html

## Build

A go 1.21 installation (or more recent) is required. For convenience, `make`
can be used to avoid typing too much.

```bash
make
```

## Configuration

By default, the configuration will be loaded from a YAML file located at
`/etc/dataplane-node-exporter.yaml`. If the file does not exist, the default
configuration will be used.

The location of the configuration file can be changed via the
`DATAPLANE_NODE_EXPORTER_YAML` environment variable.

The default configuration file can be found in the git repository:
[`dataplane-node-exporter.yaml`][conf].

[conf]: https://github.com/openstack-k8s-operators/dataplane-node-exporter/blob/main/etc/dataplane-node-exporter.yaml

## Running

The exporter will need read and write access to the `ovsdb-server` socket that
is used with `ovs-vswitchd`. Its default path is `/run/openvswitch/db.sock`.

Some collectors will need access to the `ovs-vswitchd` unixctl socket. This
socket path is resolved using the PID file of `ovs-vswitchd` at
`/run/openvswitch/ovs-vswitchd.pid` =>
`/run/openvswitch/ovs-vswitchd.$PID.ctl`.

The collector for OVN will need access to the `ovn-controller` unixctl socket. This
socket path is resolved using the PID file of `ovn-controller` at
`/run/ovn/ovn-controller.pid` => `/run/ovn/ovn-controller.$PID.ctl`.

The bridge collector will need access to each bridge OpenFlow management socket
located at `/run/openvswitch/$BRIDGE_NAME.mgmt`.

```console
$ ./dataplane-node-exporter
NOTICE  14:49:18 main.go:86: listening on http://:1981/metrics
```

## Metrics

The complete list of supported metrics can be displayed using the `-l` flag:

```console
$ ./dataplane-node-exporter -l
ovs_bridge_port_count collector=bridge set=base type=gauge labels=bridge,datapath_type help="The number of ports in a bridge."
ovs_bridge_flow_count collector=bridge set=base type=gauge labels=bridge,datapath_type help="The number of openflow rules configured on a bridge."
...
ovs_pmd_rxq_usage collector=pmd-rxq set=perf type=gauge labels=numa,cpu,interface,rxq help="Percentage of CPU cycles used to process packets from one Rxq."
ovs_build_info collector=vswitch set=base type=gauge labels=ovs_version,dpdk_version,db_version help="Version and library from which OVS binaries were built."
ovs_dpdk_initialized collector=vswitch set=base type=gauge labels= help="Has the DPDK subsystem been initialized."
```

## Contributing

[Fork the project][fork] if you haven't already done so. Configure your clone
to point at your fork and keep a reference on the upstream repository. You can
also take the opportunity to configure git to use SSH for pushing and https://
for pulling.

[fork]: https://github.com/openstack-k8s-operators/dataplane-node-exporter/fork

```console
$ git remote remove origin
$ git remote add upstream https://github.com/openstack-k8s-operators/dataplane-node-exporter
$ git remote add origin https://github.com/rjarry/dataplane-node-exporter
$ git fetch --all
Fetching origin
From https://github.com/rjarry/dataplane-node-exporter
 * [new branch]                main       -> origin/main
Fetching upstream
From https://github.com/openstack-k8s-operators/dataplane-node-exporter
 * [new branch]                main       -> upstream/main
$ git config url.git@github.com:.pushinsteadof https://github.com/
```

Create a local branch named after the topic of your future commits:

```bash
git checkout -b irq-counters
```

Patch the code. Ensure that your code is properly formatted with `make format`.
Ensure that everything builds and works as expected. Ensure that you did not
break anything.

- Do not forget to update the configuration files, if applicable.
- Run the linters using `make lint`.

Once you are happy with your work, you can create a commit (or several
commits). Follow these general rules:

- Limit the first line (title) of the commit message to 60 characters.
- Use a short prefix for the commit title for readability with `git log
  --oneline`. Do not use the `fix:` nor `feature:` prefixes. See recent commits
  for inspiration.
- Only use lower case letters for the commit title except when quoting symbols
  or known acronyms.
- Use the body of the commit message to actually explain what your patch does
  and why it is useful. Even if your patch is a one line fix, the description
  is not limited in length and may span over multiple paragraphs. Use proper
  English syntax, grammar and punctuation.
- Address only one issue/topic per commit.
- Describe your changes in imperative mood, e.g. *"make xyzzy do frotz"*
  instead of *"[This patch] makes xyzzy do frotz"* or *"[I] changed xyzzy to do
  frotz"*, as if you are giving orders to the codebase to change its behaviour.
- If you are fixing an issue, add an appropriate `Closes: <ISSUE_URL>` trailer.
- If you are fixing a regression introduced by another commit, add a `Fixes:
  <SHORT_ID_12_LONG> "<COMMIT_TITLE>"` trailer.
- When in doubt, follow the format and layout of the recent existing commits.
- The following trailers are accepted in commits. If you are using multiple
  trailers in a commit, it's preferred to also order them according to this
  list.

  * `Closes: <URL>` closes the referenced issue.
  * `Fixes: <sha> ("<title>")` reference the commit that introduced a regression.
  * `Link:`
  * `Cc:`
  * `Suggested-by:`
  * `Requested-by:`
  * `Reported-by:`
  * `Co-authored-by:`
  * `Signed-off-by:` compulsory!
  * `Tested-by:`
  * `Reviewed-by:`
  * `Acked-by:`

There is a great reference for commit messages in the [Linux kernel
documentation][linux-commits].

[linux-commits]: https://www.kernel.org/doc/html/latest/process/submitting-patches.html#describe-your-changes

IMPORTANT: you must sign-off your work using `git commit --signoff`. Follow the
[Linux kernel developer's certificate of origin][signoff] for more details. All
contributions are made under the [Apache 2.0][license] license. Please use your
real name and not a pseudonym. Here is an example:

    Signed-off-by: Robin Jarry <rjarry@redhat.com>

[signoff]: https://www.kernel.org/doc/html/latest/process/submitting-patches.html#sign-your-work-the-developer-s-certificate-of-origin

Once you are happy with your commits, you can verify that they are correct with
the following command:

```console
$ make check-commits
ok    [1/1] 'collectors: add irq-counters metrics'
2/2 valid commits
```

You can then push your topic branch on your fork:

```console
$ git push origin irq-counters
Enumerating objects: 11, done.
Counting objects: 100% (11/11), done.
Delta compression using up to 8 threads
Compressing objects: 100% (6/6), done.
Writing objects: 100% (6/6), 771 bytes | 771.00 KiB/s, done.
Total 6 (delta 5), reused 0 (delta 0), pack-reused 0 (from 0)
remote: Resolving deltas: 100% (5/5), completed with 5 local objects.
remote:
remote: Create a pull request for 'irq-counters' on GitHub by visiting:
remote:      https://github.com/rjarry/dataplane-node-exporter/pull/new/irq-counters
remote:
To github.com:rjarry/dataplane-node-exporter
 * [new branch]                irq-counters -> irq-counters
```

Before your pull request can be applied, it needs to be reviewed and approved
by project members.
