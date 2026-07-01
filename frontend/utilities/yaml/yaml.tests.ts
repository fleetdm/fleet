import { agentOptionsToYaml } from "utilities/yaml";

describe("agentOptionsToYaml", () => {
  it("omits command_line_flags when absent", () => {
    expect(agentOptionsToYaml({ config: {} })).not.toContain(
      "command_line_flags"
    );
  });

  it("omits command_line_flags when agent options are unset", () => {
    expect(agentOptionsToYaml(null)).not.toContain("command_line_flags");
  });

  it("renders command_line_flags when set to an empty object", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: {},
    });
    expect(result).toContain("command_line_flags: {}");
  });

  it("renders command_line_flags when set to null", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: null,
    });
    expect(result).toContain("command_line_flags: null");
  });

  it("renders non-empty command_line_flags", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: { verbose: true },
    });
    expect(result).toContain("command_line_flags:\n  verbose: true");
  });

  it("omits an empty overrides key", () => {
    expect(agentOptionsToYaml({ config: {}, overrides: {} })).not.toContain(
      "overrides"
    );
  });
});
