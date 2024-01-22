import React, { useContext, useState } from "react";

import { COLORS } from "styles/var/colors";
import configAPI from "services/entities/config";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink/CustomLink";
import Button from "components/buttons/Button/Button";
import SectionHeader from "components/SectionHeader";
import validateUrl from "components/forms/validators/valid_url";
import ReactTooltip from "react-tooltip";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

const baseClass = "idp-section";

type IIdpFormData = {
  idpName: string;
  entityId: string;
  metadataUrl: string;
};

type FormNames = keyof IIdpFormData;

const validateMetadataUrl = (val: string) => {
  return validateUrl({ url: val });
};

const IdpSection = () => {
  const { config } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [formData, setFormData] = useState<IIdpFormData>({
    idpName: config?.mdm.end_user_authentication?.idp_name || "",
    entityId: config?.mdm.end_user_authentication?.entity_id || "",
    metadataUrl: config?.mdm.end_user_authentication?.metadata_url || "",
  });

  // we only validate this one input so just going to use simple boolean to
  // track validation. If we need to validate more inputs in the future we can
  // use a formErrors object.
  const [isValidMetadataUrl, setIsValidMetadataUrl] = useState(true);

  const completedForm = Object.values(formData).every((value) => value !== "");

  const onInputChange = (newVal: { name: FormNames; value: string }) => {
    const { name, value } = newVal;
    const newFormData: IIdpFormData = { ...formData, [name]: value };
    setFormData(newFormData);
  };

  const onSubmit = async (e: React.FormEvent<SubmitEvent>) => {
    e.preventDefault();
    if (!validateMetadataUrl(formData.metadataUrl)) {
      setIsValidMetadataUrl(false);
      return;
    }
    setIsValidMetadataUrl(true);

    try {
      await configAPI.update({
        mdm: {
          end_user_authentication: {
            idp_name: formData.idpName,
            entity_id: formData.entityId,
            metadata_url: formData.metadataUrl,
          },
        },
      });
      renderFlash("success", "Successfully updated end user authentication!");
    } catch (err) {
      renderFlash("error", "Could not update. Please try again.");
    }
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="End user authentication" />
      <form>
        <p>
          Connect Fleet to your identity provider to require end users to
          authenticate when they first setup their new macOS hosts.{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience##end-user-authentication-and-eula"
            text="Learn more"
            newTab
          />
        </p>
        <InputField
          label="Identity provider name"
          onChange={onInputChange}
          name="idpName"
          value={formData.idpName}
          parseTarget
          tooltip="A required human friendly name for the identity provider that will provide single sign-on authentication."
        />
        <InputField
          label="Entity ID"
          onChange={onInputChange}
          name="entityId"
          value={formData.entityId}
          parseTarget
          tooltip="The required entity ID is a URI that you use to identify Fleet when configuring the identity provider."
        />
        <InputField
          label="Metadata URL"
          onChange={onInputChange}
          name="metadataUrl"
          value={formData.metadataUrl}
          parseTarget
          error={!isValidMetadataUrl && "Must be a valid URL."}
          tooltip="The metadata URL supplied by the identity provider."
        />
        <Button
          disabled={!completedForm}
          onClick={onSubmit}
          className="button-wrap"
        >
          <span data-tip data-for="save-button">
            Save
          </span>
        </Button>
        <ReactTooltip
          id="save-button"
          place="top"
          effect="solid"
          type="dark"
          backgroundColor={COLORS["tooltip-bg"]}
        >
          Complete all fields to save end user authentication.
        </ReactTooltip>
      </form>
    </div>
  );
};

export default IdpSection;
