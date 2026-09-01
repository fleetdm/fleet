import { getMatchedSoftwareIcon } from "./index";

import AcrobatReader from "./AcrobatReader";
import AdobeCreativeCloud from "./png/AdobeCreativeCloud.png";
import AdobePlugin from "./AdobePlugin";
import Extension from "./Extension";

describe("getMatchedSoftwareIcon", () => {
  describe("Adobe plugins", () => {
    it("uses the Adobe plugin icon for a plugin named after its host application", () => {
      expect(
        getMatchedSoftwareIcon({
          name: "Adobe Creative Cloud Libraries",
          source: "adobe_plugins",
        })
      ).toBe(AdobePlugin);
    });

    it("uses the Adobe plugin icon for a third-party plugin", () => {
      expect(
        getMatchedSoftwareIcon({
          name: "Artisan Pro X",
          source: "adobe_plugins",
        })
      ).toBe(AdobePlugin);
    });

    it("uses the Adobe plugin icon for a plugin whose name matches a strict rule", () => {
      expect(
        getMatchedSoftwareIcon({ name: "zoom", source: "adobe_plugins" })
      ).toBe(AdobePlugin);
    });
  });

  describe("other sources keep matching on name first", () => {
    it("matches an Adobe application by exact name", () => {
      expect(
        getMatchedSoftwareIcon({
          name: "Adobe Creative Cloud",
          source: "apps",
        })
      ).toBe(AdobeCreativeCloud);
    });

    it("matches an Adobe application by name prefix", () => {
      expect(
        getMatchedSoftwareIcon({
          name: "Adobe Acrobat Reader DC",
          source: "apps",
        })
      ).toBe(AcrobatReader);
    });

    it("falls back to the source icon when the name matches nothing", () => {
      expect(
        getMatchedSoftwareIcon({
          name: "Some Unmatched Extension",
          source: "vscode_extensions",
        })
      ).toBe(Extension);
    });
  });
});
