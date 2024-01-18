import Button from "components/buttons/Button";
import React from "react";
import { handleInputChange } from "react-select-5/dist/declarations/src/utils";

const baseClass = "team-settings";

interface ITeamSettings {}

const TeamSettings = ({}: ITeamSettings) => {
  return (
    <section className={`${baseClass}`}>
      <div className="section-header">Settings</div>
      <form>
        <TeamHostExpiryOption
          globalExpiry={globalExpiry}
          teamExpiryEnabled={teamExpiryEnabled}
          setTeamExpiryEnabled={setTeamExpiryEnabled}
        />
        <
        
        {
        // encompasses both when global setting is not enabled and
        // when it is and the user has opted to override it with a local setting
        teamExpiryEnabled && 
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
