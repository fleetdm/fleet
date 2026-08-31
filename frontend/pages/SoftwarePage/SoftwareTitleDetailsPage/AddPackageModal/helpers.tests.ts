import { getFileTypeRestriction } from "./helpers";

describe("AddPackageModal helpers — getFileTypeRestriction", () => {
  it("returns a macOS .pkg restriction for a .pkg filename", () => {
    expect(getFileTypeRestriction("GlobalProtect-v6.3.2.pkg")).toEqual({
      accept: ".pkg",
      label: "macOS (.pkg)",
    });
  });

  it("returns a Linux .deb restriction for a .deb filename", () => {
    expect(getFileTypeRestriction("cinc_18.2.11-1_amd64.deb")).toEqual({
      accept: ".deb",
      label: "Linux (.deb)",
    });
  });

  it("returns a Windows .msi restriction for a .msi filename", () => {
    expect(getFileTypeRestriction("ZoomInstaller.msi")).toEqual({
      accept: ".msi",
      label: "Windows (.msi)",
    });
  });

  // .tar.gz needs the dual MIME/extension workaround because browsers can't
  // match the compound extension via `accept` alone.
  it("uses the gzip MIME workaround for .tar.gz", () => {
    expect(getFileTypeRestriction("bundle-1.0.0.tar.gz")).toEqual({
      accept: "application/gzip,.tgz",
      label: "Linux (.tar.gz)",
    });
  });

  it("normalizes .tgz aliases through to .tar.gz", () => {
    // `getExtensionFromFileName` rewrites .tgz → .tar.gz; the restriction
    // should match.
    expect(getFileTypeRestriction("bundle.tgz")).toEqual({
      accept: "application/gzip,.tgz",
      label: "Linux (.tar.gz)",
    });
  });

  it("returns null for an unrecognized extension", () => {
    expect(getFileTypeRestriction("README.txt")).toBeNull();
  });

  it("returns null for a filename without an extension", () => {
    expect(getFileTypeRestriction("installer")).toBeNull();
  });

  it("returns null for an empty string", () => {
    expect(getFileTypeRestriction("")).toBeNull();
  });

  it("returns a macOS & Linux restriction for a .sh script package", () => {
    expect(getFileTypeRestriction("setup.sh")).toEqual({
      accept: ".sh",
      label: "macOS & Linux (.sh)",
    });
  });

  it("returns a Windows .ps1 restriction", () => {
    expect(getFileTypeRestriction("setup.ps1")).toEqual({
      accept: ".ps1",
      label: "Windows (.ps1)",
    });
  });
});
