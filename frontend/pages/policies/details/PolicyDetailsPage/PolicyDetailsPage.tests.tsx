import { IPolicy } from "interfaces/policy";
import { ILabelPolicy } from "interfaces/label";

import { getLabelModalData } from "./PolicyDetailsPage";

// Stub SoftwareIcon to avoid asset resolution when importing the page module.
jest.mock("pages/SoftwarePage/components/icons/SoftwareIcon", () => {
  return () => null;
});

const labels = (...names: string[]): ILabelPolicy[] =>
  names.map((name, i) => ({ id: i + 1, name }));

const createMockPolicy = (overrides?: Partial<IPolicy>): IPolicy => ({
  id: 1,
  name: "Test policy",
  query: "SELECT 1;",
  description: "",
  author_id: 1,
  author_name: "Admin",
  author_email: "admin@example.com",
  resolution: "",
  platform: "darwin",
  team_id: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "dynamic",
  ...overrides,
});

describe("getLabelModalData", () => {
  it("returns no label data when the policy has no labels", () => {
    expect(getLabelModalData(createMockPolicy())).toEqual({
      includeLabels: undefined,
      includeScopeLabel: undefined,
      excludeLabels: undefined,
      excludeScopeLabel: undefined,
    });
  });

  it("treats empty label arrays as no labels", () => {
    const result = getLabelModalData(
      createMockPolicy({
        labels_include_any: [],
        labels_exclude_all: [],
      })
    );

    expect(result.includeLabels).toBeUndefined();
    expect(result.excludeLabels).toBeUndefined();
  });

  describe("include labels", () => {
    it("resolves labels_include_any with the 'have any' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_include_any: labels("A") })
      );

      expect(result.includeLabels).toEqual(labels("A"));
      expect(result.includeScopeLabel).toBe("have any");
      expect(result.excludeLabels).toBeUndefined();
    });

    it("resolves labels_include_all with the 'have all' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_include_all: labels("A") })
      );

      expect(result.includeLabels).toEqual(labels("A"));
      expect(result.includeScopeLabel).toBe("have all");
    });

    it("prefers labels_include_any over labels_include_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_any: labels("Any"),
          labels_include_all: labels("All"),
        })
      );

      expect(result.includeLabels).toEqual(labels("Any"));
      expect(result.includeScopeLabel).toBe("have any");
    });
  });

  describe("exclude labels", () => {
    it("resolves labels_exclude_any with the 'exclude any' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_exclude_any: labels("A") })
      );

      expect(result.excludeLabels).toEqual(labels("A"));
      expect(result.excludeScopeLabel).toBe("exclude any");
      expect(result.includeLabels).toBeUndefined();
    });

    it("resolves labels_exclude_all with the 'exclude all' scope", () => {
      const result = getLabelModalData(
        createMockPolicy({ labels_exclude_all: labels("A") })
      );

      expect(result.excludeLabels).toEqual(labels("A"));
      expect(result.excludeScopeLabel).toBe("exclude all");
    });

    it("prefers labels_exclude_any over labels_exclude_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_exclude_any: labels("Any"),
          labels_exclude_all: labels("All"),
        })
      );

      expect(result.excludeLabels).toEqual(labels("Any"));
      expect(result.excludeScopeLabel).toBe("exclude any");
    });
  });

  describe("include + exclude combinations", () => {
    it("resolves include_any + exclude_any", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_any: labels("Inc"),
          labels_exclude_any: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have any");
      expect(result.excludeScopeLabel).toBe("exclude any");
    });

    it("resolves include_any + exclude_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_any: labels("Inc"),
          labels_exclude_all: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have any");
      expect(result.excludeScopeLabel).toBe("exclude all");
    });

    it("resolves include_all + exclude_any", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_all: labels("Inc"),
          labels_exclude_any: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have all");
      expect(result.excludeScopeLabel).toBe("exclude any");
    });

    it("resolves include_all + exclude_all", () => {
      const result = getLabelModalData(
        createMockPolicy({
          labels_include_all: labels("Inc"),
          labels_exclude_all: labels("Exc"),
        })
      );

      expect(result.includeScopeLabel).toBe("have all");
      expect(result.excludeScopeLabel).toBe("exclude all");
    });
  });
});
