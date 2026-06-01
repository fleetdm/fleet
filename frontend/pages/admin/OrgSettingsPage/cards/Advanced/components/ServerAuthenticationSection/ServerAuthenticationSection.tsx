import React from "react";
import SettingsSection from "pages/admin/components/SettingsSection";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";

import { IAdvancedSectionProps } from "../../Advanced";

const ServerAuthenticationSection = ({
  formData,
  onInputChange,
  formErrors = {},
  onInputBlur,
  appConfig,
}: IAdvancedSectionProps) => {
  const {
    ssoUserURL,
    mdmAppleServerURL,
    domain,
    verifySSLCerts,
    enableStartTLS,
  } = formData;
  return (
    <SettingsSection title="Server & authentication">
      <GitOpsModeTooltipWrapper
        position="left"
        isInputField
        renderChildren={(disableChildren) => (
          <InputField
            disabled={disableChildren}
            label="SSO user URL"
            onChange={onInputChange}
            onBlur={onInputBlur}
            name="ssoUserURL"
            value={ssoUserURL}
            parseTarget
            error={formErrors.ssoUserURL}
            tooltip={
              !disableChildren &&
              "Update this URL if you want your Fleet users (admins, maintainers, observers) to login via SSO using a URL that's different than the base URL of your Fleet instance. If not configured, login via SSO will use the base URL of the Fleet instance."
            }
          />
        )}
      />
      {appConfig?.mdm.enabled_and_configured && (
        <GitOpsModeTooltipWrapper
          position="left"
          isInputField
          renderChildren={(disableChildren) => (
            <InputField
              disabled={disableChildren}
              label="Apple MDM server URL"
              onChange={onInputChange}
              onBlur={onInputBlur}
              name="mdmAppleServerURL"
              value={mdmAppleServerURL}
              parseTarget
              error={formErrors.mdmAppleServerURL}
              tooltip={
                !disableChildren &&
                "Update this URL if you're self-hosting Fleet and you want your hosts to talk to this URL for MDM features. If not configured, hosts will use the base URL of the Fleet instance."
              }
              helpText="If this URL changes and hosts already have MDM turned on, the end users will have to turn MDM off and back on to use MDM features."
            />
          )}
        />
      )}
      <InputField
        label="Domain"
        onChange={onInputChange}
        onBlur={onInputBlur}
        name="domain"
        value={domain}
        parseTarget
        error={formErrors.domain}
        tooltip={
          <>
            If you need to specify a HELO domain, <br />
            you can do it here{" "}
            <em>
              (Default: <strong>Blank</strong>)
            </em>
          </>
        }
      />
      <Checkbox
        onChange={onInputChange}
        name="verifySSLCerts"
        value={verifySSLCerts}
        parseTarget
        labelTooltipContent={
          <>
            Turn this off (not recommended) <br />
            if you use a self-signed certificate{" "}
            <em>
              <br />
              (Default: <strong>On</strong>)
            </em>
          </>
        }
      >
        Verify SSL certs
      </Checkbox>
      <Checkbox
        onChange={onInputChange}
        name="enableStartTLS"
        value={enableStartTLS}
        parseTarget
        labelTooltipContent={
          <>
            Detects if STARTTLS is enabled <br />
            in your SMTP server and starts <br />
            to use it.{" "}
            <em>
              (Default: <strong>On</strong>)
            </em>
          </>
        }
      >
        Enable STARTTLS
      </Checkbox>
    </SettingsSection>
  );
};

export default ServerAuthenticationSection;
