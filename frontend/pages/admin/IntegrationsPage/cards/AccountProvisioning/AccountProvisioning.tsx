import React, { useState } from "react";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import { IInputFieldParseTarget } from "interfaces/form_field";
import Button from "components/buttons/Button";

const baseClass = "account-provisioning";

const AccountProvisioning = () => {
  const [tokenUrl, setTokenUrl] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    if (name === "tokenUrl") setTokenUrl(value as string);
    if (name === "clientId") setClientId(value as string);
    if (name === "clientSecret") setClientSecret(value as string);
  };

  return (
    <SettingsSection title="Account provisioning" className={baseClass}>
      <PageDescription
        variant="right-panel"
        content={
          <>
            Create and sync macOS accounts using IdP credentials with any IdP
            that supports OAuth ROPG{" "}
            <span style={{ whiteSpace: "nowrap" }}>
              (Okta){" "}
              <CustomLink
                newTab
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/idp-account-sync`}
                text="Learn more"
              />
            </span>
          </>
        }
      />
      <form onSubmit={() => {}}>
        <InputField
          label="Token URL"
          name="tokenUrl"
          value={tokenUrl}
          onChange={onInputChange}
          parseTarget
          helpText="Your IdP URL for verifying login credentials. For Okta, this is typically https://yourdomain.okta.com/oauth2/v1/token."
        />
        <InputField
          label="Client ID"
          name="clientId"
          value={clientId}
          onChange={onInputChange}
          parseTarget
          helpText="In Okta, this will be in the Client Credentials section."
        />
        <InputField
          label="Client secret"
          name="clientSecret"
          value={clientSecret}
          onChange={onInputChange}
          parseTarget
          helpText="In Okta, this will be in the Client Credentials section."
        />
        <div className="button-wrap">
          <Button type="submit">Save</Button>
        </div>
      </form>
    </SettingsSection>
  );
};

export default AccountProvisioning;
