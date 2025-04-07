/*
 * NOTE: This is an example of how to define data for your mock responses.
 * Be sure to copy this file into `../mocks` and only edit that copy!
 * Also please check the README for how to use the mock service :)
 */

const HOST_ID = {
  host: {
    created_at: "2021-03-31T00:00:00Z",
    updated_at: "2021-03-31T00:00:00Z",
    software: [],
    id: 1337,
    detail_updated_at: "2021-03-31T00:00:00Z",
    label_updated_at: "2021-03-31T00:00:00Z",
    policy_updated_at: "2021-03-31T00:00:00Z",
    last_enrolled_at: "2021-03-31T00:00:00Z",
    seen_time: "2021-03-31T00:00:00ZZ",
    refetch_requested: false,
    hostname: "myf1337d3v1c3",
    display_name: "myf1337d3v1c3",
    uuid: "13371337-0000-0000-1337-133713371337",
    platform: "rhel",
    osquery_version: "5.1.0",
    os_version: "Ubuntu 20.4.0",
    build: "",
    platform_like: "deb",
    code_name: "",
    uptime: 1337133713371337,
    memory: 143593800000,
    cpu_type: "1337",
    cpu_subtype: "1337",
    cpu_brand: "Intel(R) Core(TM) i3-37k CPU @ 13.37GHz",
    cpu_physical_cores: 8,
    cpu_logical_cores: 8,
    hardware_vendor: "",
    hardware_model: "",
    hardware_version: "",
    hardware_serial: "",
    computer_name: "myf1337d3v1c3",
    primary_ip: "133.7.133.7",
    primary_mac: "13:37:13:37:13:37",
    distributed_interval: 1337,
    config_tls_refresh: 1337,
    logger_tls_period: 1337,
    team_id: null,
    pack_stats: [],
    team_name: null,
    users: [
      {
        uid: 1337,
        username: "root",
        type: "",
        groupname: "root",
        shell: "/bin/bash",
      },
    ],
    gigs_disk_space_available: 13.37,
    percent_disk_space_available: 13.37,
    issues: {
      total_issues_count: 1337,
      critical_issues_count: 1330,
      failing_policies_count: 7,
    },
    labels: [],
    packs: [],
    policies: [],
    status: "online",
    display_text: "myf1337d3v1c3",
  },
};
const HOST_1337 = {
  ...HOST_ID,
  team_id: 1337,
  team_name: "h4x0r",
};

export default {
  ALL_HOSTS: {
    hosts: [HOST_ID.host],
  },
  HOSTS_TEAM_ID: {
    hosts: [{ ...HOST_ID.host, team_id: 2, team_name: "n00bz" }],
  },
  HOSTS_TEAM_1337: {
    hosts: [HOST_1337.host],
  },
  HOST_ID,
  HOST_1337,
  DEVICE_MAPPING: {
    host_id: 1337,
    device_mapping: null,
    foo: "bar",
  },
  MACADMINS: {
    macadmins: null,
    foo: "bar",
  },
};
