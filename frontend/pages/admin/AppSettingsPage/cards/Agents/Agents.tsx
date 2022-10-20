import React, { useState, useEffect } from "react";
import yaml from "js-yaml";
import paths from "router/paths";
import { constructErrorString, agentOptionsToYaml } from "utilities/yaml";
import { EMPTY_AGENT_OPTIONS } from "utilities/constants";

import Button from "components/buttons/Button";
// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import YamlAce from "components/YamlAce";
import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";
import { IAppConfigFormProps, IAppConfigFormErrors } from "../constants";

const baseClass = "app-config-form";

const Agents = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const { ADMIN_TEAMS } = paths;

  const [formData, setFormData] = useState<any>({
    agentOptions: agentOptionsToYaml(appConfig.agent_options),
  });
  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const { agentOptions } = formData;

  const handleAgentOptionsChange = (value: string) => {
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

  // onChange basic yaml validation only
  useEffect(() => {
    validateForm();
  }, [agentOptions]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI and allows empty agent options
    const formDataToSubmit = agentOptions
      ? {
          agent_options: yaml.load(agentOptions),
        }
      : { agent_options: EMPTY_AGENT_OPTIONS };

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
              target="_blank"
              rel="noopener noreferrer"
            >
              Learn more about agent options
              <img src={ExternalLinkIcon} alt="Open external link" />
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
                target="_blank"
                rel="noopener noreferrer"
              >
                Learn more about{" "}
                <span className="no-wrap">
                  teams
                  <img alt="Open external link" src={ExternalLinkIcon} />
                </span>
              </a>
            </InfoBanner>
          )}
          <p className={`${baseClass}__component-label`}>
            <b>YAML</b>
          </p>
          <YamlAce
            wrapperClassName={`${baseClass}__text-editor-wrapper`}
            onChange={handleAgentOptionsChange}
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
        className="save-loading"
        isLoading={isUpdatingSettings}
      >
        Save
      </Button>
    </form>
  );
};

export default Agents;
