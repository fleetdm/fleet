import {
  deriveVersionOptions,
  getPreselectedVersionValue,
  LATEST_VERSION_VALUE,
} from "./helpers";

const v = (id: number, version: string) => ({
  id,
  version,
  filename: `installer-${version}.pkg`,
  uploaded_at: "2026-01-01T00:00:00Z",
});

describe("VersionsModal helpers", () => {
  describe("deriveVersionOptions", () => {
    it("always leads with the 'Latest' option", () => {
      const opts = deriveVersionOptions([]);
      expect(opts).toEqual([
        {
          value: LATEST_VERSION_VALUE,
          label: "Automatically update to latest",
        },
      ]);
    });

    it("orders Latest, then exact pins (newest first), then the major option", () => {
      // Mirrors the Figma example: two cached versions in different majors.
      const opts = deriveVersionOptions([
        v(1, "148.0.7778.179"),
        v(2, "149.0.7827.54"),
      ]);
      expect(opts).toEqual([
        {
          value: LATEST_VERSION_VALUE,
          label: "Automatically update to latest",
        },
        { value: "149.0.7827.54", label: "Pin to 149.0.7827.54" },
        { value: "148.0.7778.179", label: "Pin to 148.0.7778.179" },
        { value: "^149", label: "Pin to major version (149)" },
      ]);
    });

    it("offers a single major option tracking the latest version's major", () => {
      const opts = deriveVersionOptions([
        v(1, "149.0.7827.54"),
        v(2, "149.0.7700.0"),
        v(3, "148.0.1"),
      ]);
      const majorOpts = opts.filter((o) => o.value.startsWith("^"));
      expect(majorOpts).toEqual([
        { value: "^149", label: "Pin to major version (149)" },
      ]);
      // Major option is last, after the exact pins.
      expect(opts[opts.length - 1].value).toBe("^149");
    });
  });

  describe("getPreselectedVersionValue", () => {
    it("maps null/undefined/empty to Latest", () => {
      expect(getPreselectedVersionValue(null)).toBe(LATEST_VERSION_VALUE);
      expect(getPreselectedVersionValue(undefined)).toBe(LATEST_VERSION_VALUE);
      expect(getPreselectedVersionValue("")).toBe(LATEST_VERSION_VALUE);
    });

    it("passes through an exact-version pin", () => {
      expect(getPreselectedVersionValue("149.0.7827.54")).toBe("149.0.7827.54");
    });

    it("passes through a major-version pin", () => {
      expect(getPreselectedVersionValue("^149")).toBe("^149");
    });
  });
});
