import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import yaml from "js-yaml";

import { NotificationContext } from "context/notification";
import { ITeam } from "interfaces/team";
import endpoints from "utilities/endpoints";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import osqueryOptionsAPI from "services/entities/osquery_options";

// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";
// @ts-ignore
import OsqueryOptionsForm from "components/forms/admin/OsqueryOptionsForm";
import InfoBanner from "components/InfoBanner/InfoBanner";
import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

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

  const [formData, setFormData] = useState<{ osquery_options?: string }>({});
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
            osquery_options: yaml.dump(selected.agent_options),
          });
        } else {
          handlePageError({ status: 404 });
        }
      },
      onError: (error) => handlePageError(error),
    }
  );

  const onSaveOsqueryOptionsFormSubmit = async (updatedForm: {
    osquery_options: string;
  }) => {
    const { TEAMS_AGENT_OPTIONS } = endpoints;
    const { error } = validateYaml(updatedForm.osquery_options);
    if (error) {
      return renderFlash("error", error.reason);
    }

    try {
      await osqueryOptionsAPI.update(
        updatedForm,
        TEAMS_AGENT_OPTIONS(teamIdFromURL)
      );
      return renderFlash("success", "Successfully saved agent options");
    } catch (response) {
      console.error(response);
      return renderFlash("error", "Could not save agent options");
    }
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
          <span style={{ whiteSpace: "nowrap" }}>
            options
            <img alt="Open external link" src={ExternalLinkIcon} />
          </span>
        </a>
      </p>
      <InfoBanner className={`${baseClass}__config-docs`}>
        See Fleet documentation for an example file that includes the overrides
        option.{" "}
        <a
          href="https://fleetdm.com/docs/using-fleet/configuration-files#overrides-option"
          target="_blank"
          rel="noopener noreferrer"
        >
          Go to Fleet docs
          <img alt="Open external link" src={ExternalLinkIcon} />
        </a>
      </InfoBanner>
      <div className={`${baseClass}__form-wrapper`}>
        <OsqueryOptionsForm
          formData={formData}
          handleSubmit={onSaveOsqueryOptionsFormSubmit}
        />
      </div>
    </div>
  );
};

export default AgentOptionsPage;
