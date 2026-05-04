import yaml from "js-yaml";

interface IYAMLError {
  name: string;
  reason: string;
  line: string;
}

export const constructErrorString = (yamlError: IYAMLError) => {
  return `${yamlError.name}: ${yamlError.reason} at line ${yamlError.line}`;
};

export const agentOptionsToYaml = (agentOpts: any) => {
  agentOpts ||= { config: {} };

  // hide the "overrides" key if it is empty
  if (!agentOpts.overrides || Object.keys(agentOpts.overrides).length === 0) {
    delete agentOpts.overrides;
  }

  // hide the "command_line_flags" key if it is empty
  if (
    !agentOpts.command_line_flags ||
    Object.keys(agentOpts.command_line_flags).length === 0
  ) {
    delete agentOpts.command_line_flags;
  }

  return yaml.dump(agentOpts);
};

export default constructErrorString;
