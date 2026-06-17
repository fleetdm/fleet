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

  const flagsComment = "# Requires fleetd agent\n";

  // always show the informational comment above "command_line_flags". When
  // the key is present — even set to {} or null — render it as-is, since
  // those empty values have special semantics; when absent, suggest it with
  // a commented-out placeholder.
  let yamlString = yaml.dump(agentOpts);
  if ("command_line_flags" in agentOpts) {
    yamlString = yamlString.replace(
      /^command_line_flags:/m,
      `${flagsComment}command_line_flags:`
    );
  } else {
    yamlString += `${flagsComment}# command_line_flags: {}\n`;
  }

  return yamlString;
};

export default constructErrorString;
