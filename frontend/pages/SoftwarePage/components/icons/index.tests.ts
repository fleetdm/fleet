import AcrobatReader from "./AcrobatReader";
import AdobePlugin from "./AdobePlugin";
import Extension from "./Extension";
import GoBinary from "./GoBinary";
import Package from "./Package";
import AdobeCreativeCloud from "./png/AdobeCreativeCloud.png";
import CodexApp from "./png/CodexApp.png";

import { getMatchedSoftwareIcon } from "./index";

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

  describe("Go binaries", () => {
    it("uses the Go icon for a binary whose name matches an application", () => {
      expect(
        getMatchedSoftwareIcon({ name: "zoom", source: "go_binaries" })
      ).toBe(GoBinary);
    });

    it("uses the Go icon for a binary whose name matches nothing", () => {
      expect(
        getMatchedSoftwareIcon({ name: "gopls", source: "go_binaries" })
      ).toBe(GoBinary);
    });
  });

  describe("MCP servers and AI skills", () => {
    it.each([
      { name: "slack", source: "mcp_servers" },
      { name: "zoom", source: "mcp_servers" },
      { name: "git", source: "ai_skills" },
    ])(
      "uses the package icon for $source named after an application ($name)",
      ({ name, source }) => {
        expect(getMatchedSoftwareIcon({ name, source })).toBe(Package);
      }
    );
  });

  describe("other sources keep matching on name first", () => {
    it("matches an AI CLI tool to its vendor's application", () => {
      expect(getMatchedSoftwareIcon({ name: "codex", source: "ai_clis" })).toBe(
        CodexApp
      );
    });

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
