import React from "react";

import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import SettingsSection from "pages/admin/components/SettingsSection";
import paths from "router/paths";

const LocalAccountSection = () => {
  return (
    <SettingsSection title="Local account">
      <TooltipWrapper
        tipContent={
          !isMacMdmEnabledAndConfigured ? (
            <span>
              To enable, first turn on{" "}
              <CustomLink
                url={paths.ADMIN_INTEGRATIONS_MDM_APPLE}
                text="Apple MDM"
                variant="tooltip-link"
              />
              .
            </span>
          ) : undefined
        }
        disableTooltip={!!isMacMdmEnabledAndConfigured}
        underline={false}
        position="left"
        showArrow
      >
        <Checkbox
          disabled={gitOpsModeEnabled || !isMacMdmEnabledAndConfigured}
          value={enableManagedLocalAccount}
          onChange={onToggleManagedLocalAccount}
          helpText={
            <span>
              Fleet generates a user (_fleetadmin) and unique password for each
              host, accessible in <b>Host details</b> &gt;{" "}
              <b>Show managed account</b>.
            </span>
          }
        >
          <TooltipWrapper
            tipContent={
              <>
                Creates a hidden managed local admin account for
                <br />
                remote troubleshooting on macOS hosts.
              </>
            }
          >
            Managed local account
          </TooltipWrapper>
        </Checkbox>
      </TooltipWrapper>
    </SettingsSection>
  );
};

export default LocalAccountSection;
