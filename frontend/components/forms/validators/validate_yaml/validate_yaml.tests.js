import validateYaml from "./index";

// Valid indentations take up two spaces
const malformedYaml = [
  "spec:\nconfig:\n    options:\n      logger_plugin: tls\n      pack_delimiter: /\n      logger_tls_period: 10\n      distributed_plugin: tls\n      disable_distributed: false\n      logger_tls_endpoint: /api/v1/osquery/log\n      distributed_interval: 8\n      distributed_tls_max_attempts: 5\n    decorators:\n      load:\n        - SELECT uuid AS host_uuid FROM system_info;\n        - SELECT hostname FROM system_info;\n  overrides: {}\n",

  "spec:\nconfig:\n    options:\n      logger_plugin: tls\n      pack_delimiter /\n      logger_tls_period: 10\n      distributed_plugin: tls\n      disable_distributed: false\n      logger_tls_endpoint: /api/v1/osquery/log\n      distributed_interval: 8\n      distributed_tls_max_attempts: 5\n    decorators:\n      load:\n        - SELECT uuid AS host_uuid FROM system_info;\n        - SELECT hostname FROM system_info;\n  overrides: {}\n",
];

const validYaml = [
  "spec:\n  config:\n    options:\n      logger_plugin: tls\n      pack_delimiter: /\n      logger_tls_period: 10\n      distributed_plugin: tls\n      disable_distributed: false\n      logger_tls_endpoint: /api/v1/osquery/log\n      distributed_interval: 8\n      distributed_tls_max_attempts: 5\n    decorators:\n      load:\n        - SELECT uuid AS host_uuid FROM system_info;\n        - SELECT hostname FROM system_info;\n  overrides: {}\n",
];

describe("validateYaml", () => {
  it("rejects malformed yaml", () => {
    malformedYaml.forEach((yaml) => {
      const { error, valid } = validateYaml(yaml);

      expect(valid).toEqual(false);
      expect(error.name).toEqual("Syntax Error");
      expect(error.reason).toBeTruthy();
      expect(error.line).toBeGreaterThan(0);
    });
  });

  it("rejects blank entries", () => {
    const { error, valid } = validateYaml();

    expect(valid).toEqual(false);
    expect(error).toEqual("YAML text must be present");
  });

  it("accepts valid yaml", () => {
    validYaml.forEach((yaml) => {
      const { error, valid } = validateYaml(yaml);
      expect(valid).toEqual(true);
      expect(error).toBeFalsy();
    });
  });
});
