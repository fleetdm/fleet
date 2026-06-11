import { agentOptionsToYaml } from "utilities/yaml";

const FLAGS_COMMENT = "# Requires fleetd agent\n";

describe("agentOptionsToYaml", () => {
  it("adds a commented-out placeholder when the command_line_flags key is absent", () => {
    expect(agentOptionsToYaml({ config: {} })).toContain(
      `${FLAGS_COMMENT}# command_line_flags: {}\n`
    );
  });

  it("adds a commented-out placeholder when agent options are unset", () => {
    expect(agentOptionsToYaml(null)).toContain(
      `${FLAGS_COMMENT}# command_line_flags: {}\n`
    );
  });

  it("renders command_line_flags set to an empty object as-is, with the comment above it", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: {},
    });
    expect(result).toContain(`${FLAGS_COMMENT}command_line_flags: {}`);
    expect(result).not.toContain("# command_line_flags: {}");
  });

  it("renders command_line_flags set to null as-is, with the comment above it", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: null,
    });
    expect(result).toContain(`${FLAGS_COMMENT}command_line_flags: null`);
    expect(result).not.toContain("# command_line_flags: {}");
  });

  it("renders non-empty command_line_flags with the comment above it", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: { verbose: true },
    });
    expect(result).toContain(
      `${FLAGS_COMMENT}command_line_flags:\n  verbose: true`
    );
    expect(result).not.toContain("# command_line_flags: {}");
  });

  it("omits an empty overrides key", () => {
    expect(agentOptionsToYaml({ config: {}, overrides: {} })).not.toContain(
      "overrides"
    );
  });
});
