import React from "react";

import { getPathWithQueryParams } from "utilities/url";
import CustomLink from "components/CustomLink";
import paths from "router/paths";

interface IRunScriptHelpTextProps {
  className?: string;
  isTechnician: boolean;
  canRunScripts: boolean;
  teamId?: number;
}

const RunScriptHelpText = ({
  className,
  isTechnician,
  canRunScripts,
  teamId,
}: IRunScriptHelpTextProps) => {
  const hostsUrl = getPathWithQueryParams(paths.MANAGE_HOSTS, {
    team_id: teamId,
  });

  if (isTechnician) {
    return (
      <div className={className}>
        To run this script on a host, go to the{" "}
        <CustomLink text="Hosts" url={hostsUrl} /> page and select a host. Then,
        click <b>Actions &gt; Run script</b>.
      </div>
    );
  }

  return (
    <div className={className}>
      To run this script on a host, go to the{" "}
      <CustomLink text="Hosts" url={hostsUrl} /> page and select a host.
      {canRunScripts && (
        <>
          <br />
          To run the script across multiple hosts, add a policy automation on
          the{" "}
          <CustomLink
            text="Policies"
            url={getPathWithQueryParams(paths.MANAGE_POLICIES, {
              team_id: teamId,
            })}
          />{" "}
          page.
        </>
      )}
    </div>
  );
};

export default RunScriptHelpText;
