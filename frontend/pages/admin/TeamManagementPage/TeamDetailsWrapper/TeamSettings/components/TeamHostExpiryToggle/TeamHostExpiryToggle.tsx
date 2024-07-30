import Checkbox from "components/forms/fields/Checkbox";
import Icon from "components/Icon";
import React from "react";
import { Link } from "react-router";

const baseClass = "team-host-expiry-toggle";

interface ITeamHostExpiryToggle {
  globalHostExpiryEnabled: boolean;
  globalHostExpiryWindow?: number;
  teamExpiryEnabled: boolean;
  setTeamExpiryEnabled: (value: boolean) => void;
}

const TeamHostExpiryToggle = ({
  globalHostExpiryEnabled,
  globalHostExpiryWindow,
  teamExpiryEnabled,
  setTeamExpiryEnabled,
}: ITeamHostExpiryToggle) => {
  const renderHelpText = () =>
    // this will never be rendered while globalHostExpiryWindow is undefined
    globalHostExpiryEnabled ? (
      <div className="help-text">
        Host expiry is globally enabled in organization settings. By default,
        hosts expire after {globalHostExpiryWindow} days.{" "}
        {!teamExpiryEnabled && (
          <Link
            to=""
            onClick={(e: React.MouseEvent) => {
              e.preventDefault();
              setTeamExpiryEnabled(true);
            }}
            className={`${baseClass}__add-custom-window`}
          >
            <>
              Add custom expiry window
              <Icon name="chevron-right" color="core-fleet-blue" size="small" />
            </>
          </Link>
        )}
      </div>
    ) : (
      <></>
    );
  return (
    <div className={`${baseClass}`}>
      <Checkbox
        name="enableHostExpiry"
        onChange={setTeamExpiryEnabled}
        value={teamExpiryEnabled || globalHostExpiryEnabled} // Still shows checkmark if global expiry is enabled though the checkbox will be disabled.
        disabled={globalHostExpiryEnabled}
        helpText={renderHelpText()}
        tooltipContent={
          <>
            When enabled, allows automatic cleanup of
            <br />
            hosts that have not communicated with Fleet in
            <br />
            the number of days specified in the{" "}
            <strong>
              Host expiry
              <br />
              window
            </strong>{" "}
            setting.{" "}
            <em>
              (Default: <strong>Off</strong>)
            </em>
          </>
        }
      >
        Enable host expiry
      </Checkbox>
    </div>
  );
};

export default TeamHostExpiryToggle;
