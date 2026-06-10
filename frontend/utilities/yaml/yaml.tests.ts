import { agentOptionsToYaml } from "utilities/yaml";

const FLAGS_COMMENT =
  "# Requires Fleet's osquery installer\n" +
  "# Setting this to null or {} will clear all local osquery flags on hosts\n" +
  "# command_line_flags: {}\n";

describe("agentOptionsToYaml", () => {
  it("adds a command_line_flags comment when the key is absent", () => {
    expect(agentOptionsToYaml({ config: {} })).toContain(FLAGS_COMMENT);
  });

  it("adds a command_line_flags comment when agent options are unset", () => {
    expect(agentOptionsToYaml(null)).toContain(FLAGS_COMMENT);
  });

  it("renders command_line_flags set to an empty object as-is", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: {},
    });
    expect(result).toContain("command_line_flags: {}");
    expect(result).not.toContain(FLAGS_COMMENT);
  });

  it("renders command_line_flags set to null as-is", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: null,
    });
    expect(result).toContain("command_line_flags: null");
    expect(result).not.toContain(FLAGS_COMMENT);
  });

  it("renders non-empty command_line_flags without the comment", () => {
    const result = agentOptionsToYaml({
      config: {},
      command_line_flags: { verbose: true },
    });
    expect(result).toContain("command_line_flags:\n  verbose: true");
    expect(result).not.toContain(FLAGS_COMMENT);
  });

  it("omits an empty overrides key", () => {
    expect(agentOptionsToYaml({ config: {}, overrides: {} })).not.toContain(
      "overrides"
    );
  });
});
