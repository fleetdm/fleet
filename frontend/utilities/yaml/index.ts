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

  // add a comment besides the "command_line_flags" if it is empty
  let addFlagsComment = false;
  if (
    !agentOpts.command_line_flags ||
    Object.keys(agentOpts.command_line_flags).length === 0
  ) {
    // delete it so it does not render, and will add it explicitly after (along with the comment)
    delete agentOpts.command_line_flags;
    addFlagsComment = true;
  }

  let yamlString = yaml.dump(agentOpts);
  if (addFlagsComment) {
    yamlString +=
      "# Requires Fleet's osquery installer\n# command_line_flags: {}\n";
  }

  return yamlString;
};

export default constructErrorString;
