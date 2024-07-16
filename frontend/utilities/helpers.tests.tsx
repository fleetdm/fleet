import { compareVersions } from "./helpers";

describe("helpers utilities", () => {
  describe("compareVersions function", () => {
    it("properly checks if a version is older than another", () => {
      expect(compareVersions("14.4.1", "14.4.2")).toEqual(-1);
      expect(compareVersions("14.4.1", "14.5")).toEqual(-1);
      expect(compareVersions("14.4.1", "15")).toEqual(-1);

      expect(compareVersions("14.4", "14.4.2")).toEqual(-1);
      expect(compareVersions("14.4", "14.5")).toEqual(-1);
      expect(compareVersions("14.4", "15")).toEqual(-1);

      expect(compareVersions("14", "14.4.2")).toEqual(-1);
      expect(compareVersions("14", "14.0.5")).toEqual(-1);
      expect(compareVersions("14", "15")).toEqual(-1);
    });

    it("properly checks if a version is newer than another", () => {
      expect(compareVersions("14.4.4", "14.4.3")).toEqual(1);
      expect(compareVersions("14.3.4", "14.3")).toEqual(1);
      expect(compareVersions("14.0.4", "14")).toEqual(1);

      expect(compareVersions("14.5", "14.4.3")).toEqual(1);
      expect(compareVersions("14.5", "14.3")).toEqual(1);
      expect(compareVersions("14.5", "14")).toEqual(1);

      expect(compareVersions("14", "13.9.21")).toEqual(1);
      expect(compareVersions("14", "13.9")).toEqual(1);
      expect(compareVersions("14", "13")).toEqual(1);
    });

    it("properly checks if a version is equal to another", () => {
      expect(compareVersions("14.0.4", "14.0.4")).toEqual(0);
      expect(compareVersions("14.3", "14.3")).toEqual(0);
      expect(compareVersions("14", "14")).toEqual(0);
      expect(compareVersions("14.3", "14.3.0")).toEqual(0);
      expect(compareVersions("14", "14.0.0")).toEqual(0);
    });
  });
});
