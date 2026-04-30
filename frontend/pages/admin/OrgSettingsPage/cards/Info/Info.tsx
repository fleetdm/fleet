import React, { useContext, useEffect, useRef, useState } from "react";

import { IInputFieldParseTarget } from "interfaces/form_field";
import { IOrgLogoStorableMode } from "interfaces/org_logo";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import validUrl from "components/forms/validators/valid_url";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";

import logoAPI from "services/entities/logo";
import { NotificationContext } from "context/notification";
import {
  ORG_LOGO_ACCEPT,
  validateOrgLogoFile,
} from "utilities/file/orgLogoFile";

import { IAppConfigFormProps } from "../constants";

interface IOrgInfoFormData {
  orgName: string;
  orgSupportURL: string;
}

interface IOrgInfoFormErrors {
  org_name?: string | null;
  org_support_url?: string | null;
}

interface ILogoModeState {
  pendingUpload: { file: File; url: string } | null;
  pendingDelete: boolean;
}

const baseClass = "app-config-form";
const cardClass = "org-info";

interface ILogoCardProps {
  label: string;
  mode: IOrgLogoStorableMode;
  state: ILogoModeState;
  originalUrl: string;
  inputRef: React.MutableRefObject<HTMLInputElement | null>;
  onEdit: () => void;
  onDelete: () => void;
  onFileChange: (
    e: React.ChangeEvent<HTMLInputElement>,
    mode: IOrgLogoStorableMode
  ) => void;
}

const LogoCard = ({
  label,
  mode,
  state,
  originalUrl,
  inputRef,
  onEdit,
  onDelete,
  onFileChange,
}: ILogoCardProps): JSX.Element => {
  let previewSrc = originalUrl;
  if (state.pendingUpload) {
    previewSrc = state.pendingUpload.url;
  } else if (state.pendingDelete) {
    previewSrc = "";
  }

  return (
    <div className={`${cardClass}__logo-card`}>
      <div className={`${cardClass}__logo-card-header`}>
        <span className="form-field__label">{label}</span>
        <div className={`${cardClass}__logo-card-actions`}>
          <GitOpsModeTooltipWrapper
            position="top"
            tipOffset={4}
            renderChildren={(disableChildren) => (
              <Button
                variant="icon"
                onClick={onEdit}
                disabled={disableChildren}
                title="Replace logo"
              >
                <Icon name="pencil" color="core-fleet-green" />
              </Button>
            )}
          />
          <GitOpsModeTooltipWrapper
            position="top"
            tipOffset={4}
            renderChildren={(disableChildren) => (
              <Button
                variant="icon"
                onClick={onDelete}
                disabled={disableChildren}
                title="Remove logo"
              >
                <Icon name="trash" color="core-fleet-green" />
              </Button>
            )}
          />
        </div>
      </div>
      <div
        className={`${cardClass}__icon-preview ${cardClass}__${mode}-background`}
      >
        <OrgLogoIcon className={`${cardClass}__icon-img`} src={previewSrc} />
      </div>
      <input
        ref={inputRef}
        type="file"
        accept={ORG_LOGO_ACCEPT}
        onChange={(e) => onFileChange(e, mode)}
        className={`${cardClass}__hidden-file-input`}
      />
    </div>
  );
};

const Info = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = appConfig.gitops.gitops_mode_enabled;

  const [formData, setFormData] = useState<IOrgInfoFormData>({
    orgName: appConfig.org_info.org_name || "",
    orgSupportURL:
      appConfig.org_info.contact_url || "https://fleetdm.com/company/contact",
  });

  const { orgName, orgSupportURL } = formData;
  const [formErrors, setFormErrors] = useState<IOrgInfoFormErrors>({});

  const lightOriginalUrl =
    appConfig.org_info.org_logo_url_light_mode ||
    appConfig.org_info.org_logo_url_light_background ||
    "";
  const darkOriginalUrl =
    appConfig.org_info.org_logo_url_dark_mode ||
    appConfig.org_info.org_logo_url ||
    "";

  const [lightLogo, setLightLogo] = useState<ILogoModeState>({
    pendingUpload: null,
    pendingDelete: false,
  });
  const [darkLogo, setDarkLogo] = useState<ILogoModeState>({
    pendingUpload: null,
    pendingDelete: false,
  });
  const [isSaving, setIsSaving] = useState(false);

  const lightInputRef = useRef<HTMLInputElement | null>(null);
  const darkInputRef = useRef<HTMLInputElement | null>(null);

  // Revoke every live blob URL on unmount
  const pendingUrlsRef = useRef<string[]>([]);
  useEffect(() => {
    pendingUrlsRef.current = [
      lightLogo.pendingUpload?.url,
      darkLogo.pendingUpload?.url,
    ].filter((u): u is string => !!u);
  }, [lightLogo.pendingUpload, darkLogo.pendingUpload]);
  useEffect(
    () => () => {
      pendingUrlsRef.current.forEach((u) => URL.revokeObjectURL(u));
    },
    []
  );

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const computeFormErrors = (): IOrgInfoFormErrors => {
    const errors: IOrgInfoFormErrors = {};

    if (!orgName) {
      errors.org_name = "Organization name must be present";
    }

    if (!orgSupportURL) {
      errors.org_support_url = `Organization support URL must be present`;
    } else if (
      !validUrl({ url: orgSupportURL, protocols: ["http", "https", "file"] })
    ) {
      errors.org_support_url = "Organization support URL is not a valid URL";
    }

    return errors;
  };

  const validateForm = () => {
    setFormErrors(computeFormErrors());
  };

  const setLogoFile = (
    setter: React.Dispatch<React.SetStateAction<ILogoModeState>>,
    file: File
  ) => {
    const url = URL.createObjectURL(file);
    setter((prev) => {
      if (prev.pendingUpload) URL.revokeObjectURL(prev.pendingUpload.url);
      return {
        ...prev,
        pendingUpload: { file, url },
        pendingDelete: false,
      };
    });
  };

  const onLogoFileChange = async (
    e: React.ChangeEvent<HTMLInputElement>,
    mode: IOrgLogoStorableMode
  ) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    const result = await validateOrgLogoFile(file);
    if (!result.valid) {
      renderFlash("error", result.error || "Invalid logo file.");
      return;
    }
    setLogoFile(mode === "light" ? setLightLogo : setDarkLogo, file);
  };

  const onLightDelete = () =>
    setLightLogo((prev) => {
      if (prev.pendingUpload) URL.revokeObjectURL(prev.pendingUpload.url);
      return { ...prev, pendingUpload: null, pendingDelete: true };
    });
  const onDarkDelete = () =>
    setDarkLogo((prev) => {
      if (prev.pendingUpload) URL.revokeObjectURL(prev.pendingUpload.url);
      return { ...prev, pendingUpload: null, pendingDelete: true };
    });

  const onFormSubmit = async (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Validate synchronously before any API call so an invalid org name or
    // support URL aborts the submit before logo ops run.
    const errors = computeFormErrors();
    setFormErrors(errors);
    if (Object.keys(errors).length > 0) return;

    setIsSaving(true);

    try {
      // Step 1: org_info PATCH. Errors here surface as an org-info
      // failure so the user knows which half of the form failed; logo
      // ops never fire if this fails.
      let orgInfoOk = false;
      try {
        const formDataToSubmit = {
          org_info: {
            org_name: orgName,
            contact_url: orgSupportURL,
          },
        };
        orgInfoOk = await handleSubmit(formDataToSubmit);
      } catch (e) {
        renderFlash(
          "error",
          "Couldn't save organization info. Please try again."
        );
        return;
      }
      if (!orgInfoOk) return;

      // Step 2: logo ops. Their own try so a logo failure shows a
      // logo-specific message and doesn't hide a successful org-info
      // save behind a misleading flash.
      const lightDeleted = lightLogo.pendingDelete && !!lightOriginalUrl;
      const darkDeleted = darkLogo.pendingDelete && !!darkOriginalUrl;
      const logoOps: (() => Promise<unknown>)[] = [];

      if (
        lightDeleted &&
        darkDeleted &&
        !lightLogo.pendingUpload &&
        !darkLogo.pendingUpload
      ) {
        logoOps.push(() => logoAPI.delete("all"));
      } else {
        if (lightLogo.pendingUpload) {
          const f = lightLogo.pendingUpload.file;
          logoOps.push(() => logoAPI.upload(f, "light"));
        } else if (lightDeleted) {
          logoOps.push(() => logoAPI.delete("light"));
        }
        if (darkLogo.pendingUpload) {
          const f = darkLogo.pendingUpload.file;
          logoOps.push(() => logoAPI.upload(f, "dark"));
        } else if (darkDeleted) {
          logoOps.push(() => logoAPI.delete("dark"));
        }
      }

      try {
        // eslint-disable-next-line no-restricted-syntax, no-await-in-loop
        for (const op of logoOps) {
          // eslint-disable-next-line no-await-in-loop
          await op();
        }

        const reset = (prev: ILogoModeState) => {
          if (prev.pendingUpload) URL.revokeObjectURL(prev.pendingUpload.url);
          return { ...prev, pendingUpload: null, pendingDelete: false };
        };
        setLightLogo(reset);
        setDarkLogo(reset);
      } catch (e) {
        renderFlash("error", "Couldn't update logo. Please try again.");
      }
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <SettingsSection className={cardClass} title="Organization info">
      <PageDescription
        variant="right-panel"
        content={
          <p className={`${baseClass}__section-description`}>
            This logo is displayed in the top navigation, setup experience
            window, and MDM migration dialog. Please use{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/organization-logo-size"
              text="recommended sizes"
              newTab
            />
          </p>
        }
      />
      <form onSubmit={onFormSubmit} autoComplete="off">
        <div className={`${cardClass}__logo-grid`}>
          <LogoCard
            label="Organization logo (light mode)"
            mode="light"
            state={lightLogo}
            originalUrl={lightOriginalUrl}
            inputRef={lightInputRef}
            onEdit={() => lightInputRef.current?.click()}
            onDelete={onLightDelete}
            onFileChange={onLogoFileChange}
          />
          <LogoCard
            label="Organization logo (dark mode)"
            mode="dark"
            state={darkLogo}
            originalUrl={darkOriginalUrl}
            inputRef={darkInputRef}
            onEdit={() => darkInputRef.current?.click()}
            onDelete={onDarkDelete}
            onFileChange={onLogoFileChange}
          />
        </div>
        <InputField
          label="Organization name"
          onChange={onInputChange}
          name="orgName"
          value={orgName}
          parseTarget
          onBlur={validateForm}
          error={formErrors.org_name}
          disabled={gitOpsModeEnabled}
        />
        <InputField
          label={
            <TooltipWrapper
              tipContent={
                <>
                  URL is used in &quot;Reach out to IT&quot; links shown to the
                  end
                  <br />
                  user (e.g. self-service and during MDM migration).
                </>
              }
            >
              Organization support URL
            </TooltipWrapper>
          }
          onChange={onInputChange}
          name="orgSupportURL"
          value={orgSupportURL}
          parseTarget
          onBlur={validateForm}
          error={formErrors.org_support_url}
          disabled={gitOpsModeEnabled}
        />
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={Object.keys(formErrors).length > 0 || disableChildren}
              className="button-wrap"
              isLoading={isUpdatingSettings || isSaving}
            >
              Save
            </Button>
          )}
        />
      </form>
    </SettingsSection>
  );
};

export default Info;
