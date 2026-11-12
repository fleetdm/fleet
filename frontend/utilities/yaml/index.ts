import yaml from "js-yaml";

import { IValidateYamlError } from "components/forms/validators";

export const constructErrorString = (yamlError: IValidateYamlError) => {
  if (yamlError === null) return "";
  if (typeof yamlError === "string") return yamlError;
  return `${yamlError.name}: ${yamlError.reason} at line ${yamlError.line}`;
};

export const agentOptionsToYaml = (agentOpts: any) => {
  agentOpts ||= { config: {} };

  // hide the "overrides" key if it is empty
  if (!agentOpts.overrides || Object.keys(agentOpts.overrides).length === 0) {
    delete agentOpts.overrides;
  }

  return yaml.dump(agentOpts);
};

export default constructErrorString;
