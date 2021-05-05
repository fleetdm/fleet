import React from "react";

// import OsqueryOptionsForm from "components/forms/admin/OsqueryOptionsForm";
import InfoBanner from "components/InfoBanner/InfoBanner";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "agent-options";

const AgentOptionsPage = (): JSX.Element => {
  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        This file describes options returned to osquery when it checks for
        configuration.
      </p>
      <InfoBanner className={`${baseClass}__config-docs`}>
        See Fleet documentation for an example file that includes the overrides
        option.{" "}
        <a
          href="https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md#osquery-configuration-options"
          target="_blank"
          rel="noreferrer"
        >
          Go to Fleet docs{" "}
          <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
        </a>
      </InfoBanner>
      <div className={`${baseClass}__form-wrapper`}>
        {/* <OsqueryOptionsForm
          formData={formData}
          handleSubmit={onSaveOsqueryOptionsFormSubmit}
        /> */}
      </div>
    </div>
  );
};

export default AgentOptionsPage;
