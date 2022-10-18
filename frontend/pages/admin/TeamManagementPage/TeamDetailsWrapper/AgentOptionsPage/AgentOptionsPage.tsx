import React, { useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { constructErrorString, agentOptionsToYaml } from "utilities/yaml";

import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import { ITeam } from "interfaces/team";
import endpoints from "utilities/endpoints";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import osqueryOptionsAPI from "services/entities/osquery_options";

// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";
import Button from "components/buttons/Button";
// @ts-ignore
import YamlAce from "components/YamlAce";
import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";
// import format_api_errors from "utilities/format_api_errors";
// import osquery_options from "services/entities/osquery_options";

const baseClass = "agent-options";

interface IAgentOptionsPageProps {
  params: {
    team_id: string;
  };
}

const AgentOptionsPage = ({
  params: { team_id },
}: IAgentOptionsPageProps): JSX.Element => {
  const teamIdFromURL = parseInt(team_id, 10);
  const { renderFlash } = useContext(NotificationContext);

  const [teamName, setTeamName] = useState("");
  const [formData, setFormData] = useState<{ osquery_options?: string }>({});
  const [formErrors, setFormErrors] = useState<any>({});

  const handlePageError = useErrorHandler();

  useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ILoadTeamsResponse) => data.teams,
      onSuccess: (data) => {
        const selected = data.find((team) => team.id === teamIdFromURL);

        if (selected) {
          setFormData({
            osquery_options: agentOptionsToYaml(selected.agent_options),
          });
          setTeamName(selected.name);
        } else {
          handlePageError({ status: 404 });
        }
      },
      onError: (error) => handlePageError(error),
    }
  );

  const validateForm = () => {
    // Basic yaml validation only, not agent options validation
    const errors: any = {};

    if (formData.osquery_options) {
      const { error: yamlError, valid: yamlValid } = validateYaml(
        formData.osquery_options
      );
      if (!yamlValid) {
        errors.agent_options = constructErrorString(yamlError);
      }
    }

    setFormErrors(errors);
  };

  // Validates forms when certain information is changed
  useEffect(() => {
    validateForm();
  }, [formData]);

  const onSaveOsqueryOptionsFormSubmit = async (updatedForm: {
    osquery_options: string;
  }) => {
    const { TEAMS_AGENT_OPTIONS } = endpoints;

    osqueryOptionsAPI
      .update(updatedForm, TEAMS_AGENT_OPTIONS(teamIdFromURL))
      .then(() => {
        renderFlash("success", "Successfully saved agent options");
      })
      .catch((response: { data: IApiError }) => {
        console.error(response);
        return renderFlash(
          "error",
          `Could not update ${teamName} team agent options. ${response.data.errors[0].reason}`
        );
      });
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();
    const emptyForm = { osquery_options: "" };

    const formDataToSubmit = emptyForm;

    onSaveOsqueryOptionsFormSubmit(formDataToSubmit);
  };

  const handleAgentOptionsChange = (value: string) => {
    setFormData({ ...formData, osquery_options: value });
  };

  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Agent options configure the osquery agent. When you update agent
        options, they will be applied the next time a host checks in to Fleet.
        <br />
        <a
          href="https://fleetdm.com/docs/using-fleet/fleet-ui#configuring-agent-options"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn more about agent{" "}
          <span className="no-wrap">
            options
            <img alt="Open external link" src={ExternalLinkIcon} />
          </span>
        </a>
      </p>
      <div className={`${baseClass}__form-wrapper`}>
        <form
          className={`${baseClass}__form`}
          onSubmit={onFormSubmit}
          autoComplete="off"
        >
          <div className={`${baseClass}__btn-wrap`}>
            <p>YAML</p>
            <Button type="submit" variant="brand">
              Save options
            </Button>
          </div>
          <YamlAce
            wrapperClassName={`${baseClass}__text-editor-wrapper`}
            onChange={handleAgentOptionsChange}
            name="osqueryOptions"
            value={formData.osquery_options}
            parseTarget
            error={formErrors.osquery_options}
          />
        </form>
      </div>
    </div>
  );
};

export default AgentOptionsPage;
