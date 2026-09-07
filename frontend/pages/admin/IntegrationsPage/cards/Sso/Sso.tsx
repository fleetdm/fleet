import React, { useCallback, useRef, useState } from "react";
import { isEqual } from "lodash";

import useFormValidation, { trimFormData } from "hooks/useFormValidation";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TabText from "components/TabText";
import TabNav from "components/TabNav";
import PATHS from "router/paths";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { IAppConfigFormProps } from "../../../OrgSettingsPage/cards/constants";
import EndUserAuthSection from "../IdentityProviders/components/EndUserAuthSection";
import { ISsoFormData, newSsoFormData, validateSsoForm } from "./helpers";

export const AUTH_TARGETS_BY_INDEX = ["fleet-users", "end-users"];

const Sso = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
  router,
  subsection,
}: IAppConfigFormProps): JSX.Element => {
  const gitOpsModeEnabled = appConfig.gitops.gitops_mode_enabled;
  const selectedAuthTarget = subsection as string;

  const {
    formData,
    setField,
    commitFields,
    reset,
    getError,
    clearFieldError,
    validateField,
    handleSubmit: onFormSubmit,
    isSubmitting,
  } = useFormValidation<ISsoFormData>({
    initialFormData: newSsoFormData(appConfig),
    validate: validateSsoForm,
    isSubmitting: isUpdatingSettings,
  });

  const {
    enableSso,
    idpName,
    entityId,
    idpImageUrl,
    metadata,
    metadataUrl,
    enableSsoIdpLogin,
    enableJitProvisioning,
  } = formData;

  const originalFormData = useRef(formData);

  const onValidSubmit = async (submitData: ISsoFormData) => {
    // Formatting of API not UI
    const formDataToSubmit = {
      sso_settings: {
        entity_id: submitData.entityId,
        idp_image_url: submitData.idpImageUrl,
        metadata: submitData.metadata,
        metadata_url: submitData.metadataUrl,
        idp_name: submitData.idpName,
        enable_sso: submitData.enableSso,
        enable_sso_idp_login: submitData.enableSsoIdpLogin,
        enable_jit_provisioning: submitData.enableJitProvisioning,
        issuer_uri: appConfig.sso_settings?.issuer_uri ?? "",
        enable_jit_role_sync:
          appConfig.sso_settings?.enable_jit_role_sync ?? false,
      },
    };

    if (await handleSubmit(formDataToSubmit)) {
      originalFormData.current = submitData;
      reset(submitData);
    }
  };

  const [endUserHasUnsavedChanges, setEndUserHasUnsavedChanges] = useState(
    false
  );

  const hasUnsavedChanges =
    !isEqual(trimFormData(formData), originalFormData.current) ||
    endUserHasUnsavedChanges;

  const handleTabChange = useCallback(
    (index: number) => {
      if (
        hasUnsavedChanges &&
        // eslint-disable-next-line no-alert
        !confirm("Switch tabs?\n\nChanges you made will not be saved.")
      ) {
        return;
      }

      reset(originalFormData.current);
      // The End users panel unmounts on switch, discarding its own edits, but
      // the flag it last reported stays behind.
      setEndUserHasUnsavedChanges(false);
      const newSubsection = AUTH_TARGETS_BY_INDEX[index];
      router.push(
        newSubsection === "end-users"
          ? PATHS.ADMIN_INTEGRATIONS_SSO_END_USERS
          : PATHS.ADMIN_INTEGRATIONS_SSO_FLEET_USERS
      );
    },
    [hasUnsavedChanges, reset, router]
  );

  const renderFleetSsoTab = () => {
    return (
      <form onSubmit={onFormSubmit(onValidSubmit)} autoComplete="off">
        {/* "form" class applies global form styling to fields for free */}
        <div
          className={`form ${
            gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""
          }`}
        >
          <Checkbox
            onChange={(value: boolean) => commitFields({ enableSso: value })}
            name="enableSso"
            value={enableSso}
            disabled={isSubmitting}
          >
            Enable single sign-on
          </Checkbox>
          <InputField
            label="Identity provider name"
            name="idpName"
            value={idpName}
            error={getError("idpName")}
            onChange={(value: string) => setField("idpName", value)}
            onFocus={() => clearFieldError("idpName")}
            onBlur={() => validateField("idpName")}
            disabled={isSubmitting}
            tooltip="A required human friendly name for the identity provider that will provide single sign-on authentication."
          />
          <InputField
            label="Entity ID"
            helpText="The URI you provide here must exactly match the Entity ID field used in the identity provider configuration."
            name="entityId"
            value={entityId}
            error={getError("entityId")}
            onChange={(value: string) => setField("entityId", value)}
            onFocus={() => clearFieldError("entityId")}
            onBlur={() => validateField("entityId")}
            disabled={isSubmitting}
            tooltip="The Entity ID is a required URI that you use to identify Fleet when configuring the identity provider. Okta calls this Audience Restriction."
          />
          <InputField
            label="IdP image URL"
            name="idpImageUrl"
            value={idpImageUrl}
            error={getError("idpImageUrl")}
            onChange={(value: string) => setField("idpImageUrl", value)}
            onFocus={() => clearFieldError("idpImageUrl")}
            onBlur={() => validateField("idpImageUrl")}
            disabled={isSubmitting}
            tooltip={`An optional link to an image such
            as a logo for the identity provider.`}
          />
          <InputField
            label="Metadata"
            type="textarea"
            name="metadata"
            value={metadata}
            error={getError("metadata")}
            onChange={(value: string) => setField("metadata", value)}
            onFocus={() => clearFieldError("metadata")}
            onBlur={() => validateField("metadata")}
            disabled={isSubmitting}
            tooltip="Metadata XML provided by the identity provider."
          />
          <InputField
            label="Metadata URL"
            helpText={
              <>
                If both <b>Metadata URL</b> and <b>Metadata</b> are specified,{" "}
                <b>Metadata URL</b> will be used.
              </>
            }
            name="metadataUrl"
            value={metadataUrl}
            error={getError("metadataUrl")}
            onChange={(value: string) => setField("metadataUrl", value)}
            onFocus={() => clearFieldError("metadataUrl")}
            onBlur={() => validateField("metadataUrl")}
            disabled={isSubmitting}
            tooltip="Metadata URL provided by the identity provider."
          />
          <Checkbox
            onChange={(value: boolean) =>
              commitFields({ enableSsoIdpLogin: value })
            }
            name="enableSsoIdpLogin"
            value={enableSsoIdpLogin}
            disabled={isSubmitting}
          >
            Allow SSO login initiated by identity provider
          </Checkbox>
          {isPremiumTier && (
            <Checkbox
              onChange={(value: boolean) =>
                commitFields({ enableJitProvisioning: value })
              }
              name="enableJitProvisioning"
              value={enableJitProvisioning}
              disabled={isSubmitting}
              helpText={
                <>
                  <CustomLink
                    url={`${LEARN_MORE_ABOUT_BASE_LINK}/just-in-time-provisioning`}
                    text="Learn more"
                    newTab
                  />{" "}
                  about just-in-time (JIT) user provisioning.
                </>
              }
            >
              Create user and sync permissions on login
            </Checkbox>
          )}
        </div>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={isSubmitting || disableChildren}
              className="button-wrap"
              isLoading={isSubmitting}
            >
              Save
            </Button>
          )}
        />
      </form>
    );
  };

  const onSubmitEndUserSso = async () => {
    // Notify parent component that it needs to re-fetch app config.
    // No formUpdates needed because changes are made inside the card.
    await handleSubmit({});
  };

  const renderEndUserSsoTab = () => (
    <EndUserAuthSection
      endUserAuth={appConfig.mdm?.end_user_authentication}
      onDirtyChange={setEndUserHasUnsavedChanges}
      onSubmit={onSubmitEndUserSso}
    />
  );

  return (
    <SettingsSection title="Authentication (SSO)">
      <PageDescription
        content={
          <>
            Configure authentication for Fleet users logging into Fleet or end
            users enrolling their hosts. To populate identity provider (IdP)
            host vitals and automatically delete Fleet users, head to{" "}
            <CustomLink
              text="Identity provider (IdP)"
              url={PATHS.ADMIN_INTEGRATIONS_IDENTITY_PROVIDER}
            />
            .
          </>
        }
        variant="right-panel"
      />
      <TabNav secondary>
        <Tabs
          selectedIndex={AUTH_TARGETS_BY_INDEX.indexOf(selectedAuthTarget)}
          onSelect={handleTabChange}
        >
          <TabList>
            <Tab>
              <TabText>Fleet users</TabText>
            </Tab>
            <Tab>
              <TabText>End users</TabText>
            </Tab>
          </TabList>
          <TabPanel key="fleet-users">{renderFleetSsoTab()}</TabPanel>
          <TabPanel key="end-users">{renderEndUserSsoTab()}</TabPanel>
        </Tabs>
      </TabNav>
    </SettingsSection>
  );
};

export default Sso;
