import React, { useState, useEffect } from "react";

// @ts-ignore
import constructErrorString from "utilities/yaml";
import yaml from "js-yaml";
import paths from "router/paths";

import Button from "components/buttons/Button";
// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";

import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import YamlAce from "components/YamlAce";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";
import { IAppConfigFormProps, IAppConfigFormErrors } from "../constants";

const baseClass = "app-config-form";

const Agents = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
}: IAppConfigFormProps): JSX.Element => {
  const { ADMIN_TEAMS } = paths;

  const [formData, setFormData] = useState<any>({
    agentOptions: yaml.dump(appConfig.agent_options) || {},
  });

  const { agentOptions } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleAceInputChange = (value: string) => {
    setFormData({ ...formData, agentOptions: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (agentOptions) {
      const { error: yamlError, valid: yamlValid } = validateYaml(agentOptions);
      if (!yamlValid) {
        errors.agent_options = constructErrorString(yamlError);
      }
    }

    setFormErrors(errors);
  };

  // Validates forms when certain information is changed
  useEffect(() => {
    validateForm();
  }, [agentOptions]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      agent_options: yaml.load(agentOptions),
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <div className={`${baseClass}__yaml`}>
          <h2>Agent options</h2>
          <p className={`${baseClass}__section-description`}>
            Agent options configure the osquery agent. When you update agent
            options, they will be applied the next time a host checks in to
            Fleet.
            <br />
            <a
              href="https://fleetdm.com/docs/using-fleet/fleet-ui#configuring-agent-options"
              className={`${baseClass}__learn-more`}
              target="_blank"
              rel="noopener noreferrer"
            >
              Learn more about agent options&nbsp;
              <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
            </a>
          </p>
          {isPremiumTier ? (
            <InfoBanner>
              These options are not applied to hosts on a team. To update agent
              options for hosts on a team, head to the&nbsp;
              <a href={ADMIN_TEAMS}>Teams page</a>&nbsp;and select a team.
            </InfoBanner>
          ) : (
            <InfoBanner>
              Want some hosts to have different options?&nbsp;
              <a
                href="https://fleetdm.com/docs/using-fleet/teams"
                className={`${baseClass}__learn-more ${baseClass}__learn-more--inline`}
                target="_blank"
                rel="noopener noreferrer"
              >
                Learn more about teams&nbsp;
                <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
              </a>
            </InfoBanner>
          )}
          <p className={`${baseClass}__component-label`}>
            <b>YAML</b>
          </p>
          <YamlAce
            wrapperClassName={`${baseClass}__text-editor-wrapper`}
            onChange={handleAceInputChange}
            name="agentOptions"
            value={agentOptions}
            parseTarget
            error={formErrors.agent_options}
          />
        </div>
      </div>
      <Button
        type="submit"
        variant="brand"
        disabled={Object.keys(formErrors).length > 0}
      >
        Save
      </Button>
    </form>
  );
};

export default Agents;
