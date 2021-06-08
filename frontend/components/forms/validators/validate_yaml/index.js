import yaml from "js-yaml";

const invalidYamlResponse = (message) => {
  return { valid: false, error: message };
};

const validYamlResponse = { valid: true, error: null };

export const validateYaml = (yamlText) => {
  if (!yamlText) {
    return invalidYamlResponse("YAML text must be present");
  }

  try {
    yaml.safeLoad(yamlText);

    return validYamlResponse;
  } catch (error) {
    if (error instanceof yaml.YAMLException) {
      return invalidYamlResponse({
        name: "Syntax Error",
        reason: error.reason,
        line: error.mark.line,
      });
    }

    return invalidYamlResponse(error.message);
  }
};

export default validateYaml;
