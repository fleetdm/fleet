import helpers from "./helpers";

const targets = {
  hosts: [
    {
      id: 3,
      cpu_brand: "Intel(R) Core(TM) i5-6267U CPU @ 2.90GHz",
      cpu_physical_cores: 2,
      label: "OS X El Capitan 10.11",
      name: "osx-10.11",
      platform: "darwin",
    },
    {
      id: 4,
      cpu_brand: "Intel(R) Core(TM) i5-6267U CPU @ 2.90GHz",
      cpu_physical_cores: 2,
      label: "Jason Meller's Macbook Pro",
      name: "jmeller.local",
      platform: "darwin",
    },
  ],
  labels: [
    {
      id: 4,
      label: "All Macs",
      name: "macs",
      count: 1234,
    },
  ],
};

describe("targets - helpers", () => {
  describe("#appendTargetTypeToTargets", () => {
    const { appendTargetTypeToTargets } = helpers;

    it("adds the target_type attribute", () => {
      expect(appendTargetTypeToTargets(targets.hosts, "hosts")).toEqual([
        {
          id: 3,
          cpu_brand: "Intel(R) Core(TM) i5-6267U CPU @ 2.90GHz",
          cpu_physical_cores: 2,
          cpu_type: "2 x 2.9 GHz",
          label: "OS X El Capitan 10.11",
          name: "osx-10.11",
          platform: "darwin",
          target_type: "hosts",
        },
        {
          id: 4,
          cpu_brand: "Intel(R) Core(TM) i5-6267U CPU @ 2.90GHz",
          cpu_physical_cores: 2,
          cpu_type: "2 x 2.9 GHz",
          label: "Jason Meller's Macbook Pro",
          name: "jmeller.local",
          platform: "darwin",
          target_type: "hosts",
        },
      ]);

      expect(appendTargetTypeToTargets(targets.labels, "labels")).toEqual([
        {
          id: 4,
          label: "All Macs",
          name: "macs",
          count: 1234,
          target_type: "labels",
        },
      ]);
    });
  });
});
