import Button from "components/buttons/Button";
import useTeamIdParam from "hooks/useTeamIdParam";
import { ITeamConfig } from "interfaces/team";
import { ITeamSubnavProps } from "interfaces/team_subnav";
import React, { useState } from "react";
import { useQuery } from "react-query";
import { handleInputChange } from "react-select-5/dist/declarations/src/utils";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

const baseClass = "team-settings";

const TeamSettings = ({
  location, router
}: ITeamSubnavProps) => {

  const [formData, setFormData] = useState({}); // TODO type

  // encompasses both when global setting is not enabled and
  // when it is and the user has opted to override it with a local setting
  const [UITeamExpiryEnabled, setUITeamExpiryEnabled] = useState(false);

// TODO? - render flash on successful/failed update

  const { isRouteOk, teamIdForApi } = useTeamIdParam(
    {
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: false,
      observer: false,
      observer_plus: false,
    },
  });

  // TODO
  // const onSubmit = 

  // TODO get globalExpiry from config

  const {
    data: teamData, isLoading, error
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team_details", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled: isRouteOk && !!teamIdForApi,
      select: (data) => data.team,
      onSuccess: (teamData) => {
        setUITeamExpiryEnabled(teamData.host_expiry_settings.host_expiry_enabled);
      }
    }
  );

    <section className={`${baseClass}`}>
      <div className="section-header">Settings</div>
      <form>
        <TeamHostExpiryOption
          globalExpiry={globalExpiry}
          teamExpiryEnabled={UITeamExpiryEnabled}
          setTeamExpiryEnabled={setUITeamExpiryEnabled}
        />
        <
        
        {
        UITeamExpiryEnabled && 
          <InputField
          label="Host expiry window"
          onChange={handleInputChange}
          name="host-expiry-window"
          value={teamHostExpiryWindow}
          />
        }
        <Button
          type="submit"
          variant="brand"
          className="button-wrap"
          isLoading={isUpdatingTeamSettings}
        >
          Save
        </Button>
      </form>
    </section>
  );
};

export default TeamSettings;
