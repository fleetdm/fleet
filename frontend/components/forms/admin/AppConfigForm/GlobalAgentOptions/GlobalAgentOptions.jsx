import React, { Component } from "react";
import PropTypes from "prop-types";
import { size } from "lodash";

import Button from "components/buttons/Button";
import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import YamlAce from "components/YamlAce";
import validateYaml from "components/forms/validators/validate_yaml";
import constructErrorString from "utilities/yaml";

const baseClass = "osquery-options-form";

const validate = (formData) => {
  const errors = {};
  const { error: yamlError, valid: yamlValid } = validateYaml(
    formData.osquery_options
  );

  if (!yamlValid) {
    errors.osquery_options = constructErrorString(yamlError);
  }

  const valid = !size(errors);
  return { valid, errors };
};

class GlobalAgentOptions extends Component {
  static propTypes = {
    formData: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    fields: PropTypes.shape({
      agent_options: formFieldInterface.isRequired,
    }).isRequired,
  };

  render() {
    const { handleSubmit, fields } = this.props;

    return (
      <YamlAce
        {...fields.agent_options}
        error={fields.agent_options.error}
        wrapperClassName={`${baseClass}__text-editor-wrapper`}
      />
    );
  }
}

export default Form(GlobalAgentOptions, {
  fields: ["agent_options"],
  validate,
});
