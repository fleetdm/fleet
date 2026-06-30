import {
  buildTitleWithPackages,
  createMockPackages,
  MAX_PACKAGES_PER_TITLE,
} from "./multi-package-handlers";

describe("multi-package-handlers", () => {
  describe("createMockPackages", () => {
    it("creates the requested number of packages with distinct installer_ids", () => {
      const pkgs = createMockPackages(3);
      expect(pkgs).toHaveLength(3);
      expect(pkgs.map((p) => p.installer_id)).toEqual([1, 2, 3]);
    });

    it("gives each package a unique filename, version, and hash", () => {
      const pkgs = createMockPackages(2);
      expect(pkgs[0].name).not.toEqual(pkgs[1].name);
      expect(pkgs[0].version).not.toEqual(pkgs[1].version);
      expect(pkgs[0].hash_sha256).not.toEqual(pkgs[1].hash_sha256);
    });

    it("respects per-index overrides", () => {
      const pkgs = createMockPackages(2, (i) => ({
        version: `v${i + 10}`,
      }));
      expect(pkgs[0].version).toBe("v10");
      expect(pkgs[1].version).toBe("v11");
    });

    it("returns an empty array when count is 0", () => {
      expect(createMockPackages(0)).toEqual([]);
    });
  });

  describe("buildTitleWithPackages", () => {
    it("derives software_package from packages[0]", () => {
      const pkgs = createMockPackages(2);
      const title = buildTitleWithPackages(pkgs);
      expect(title.packages).toEqual(pkgs);
      expect(title.software_package).toEqual(pkgs[0]);
    });

    it("sets software_package to null when packages is empty", () => {
      const title = buildTitleWithPackages([]);
      expect(title.packages).toEqual([]);
      expect(title.software_package).toBeNull();
    });

    it("merges title-level overrides", () => {
      const pkgs = createMockPackages(1);
      const title = buildTitleWithPackages(pkgs, { id: 99, name: "Custom" });
      expect(title.id).toBe(99);
      expect(title.name).toBe("Custom");
      // Derived field stays in lockstep with the supplied packages list.
      expect(title.software_package).toEqual(pkgs[0]);
    });
  });

  it("exposes the per-title cap so tests don't hard-code 10", () => {
    expect(MAX_PACKAGES_PER_TITLE).toBe(10);
  });
});
