import React, { useContext, useState } from "react";

import { useQuery } from "react-query";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import configAPI from "services/entities/config";

import { IConfig } from "interfaces/config";
import { IFormField } from "interfaces/form_field";
import { getErrorReason } from "interfaces/errors";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";
import validUrl from "components/forms/validators/valid_url";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

const baseClass = "change-management";

interface IChangeManagementFormData {
  gitOpsModeEnabled: boolean;
  repoURL: string;
}

interface IChangeManagementFormErrors {
  repository_url?: string | null;
}

const validate = (formData: IChangeManagementFormData) => {
  const errs: IChangeManagementFormErrors = {};
  const { gitOpsModeEnabled, repoURL } = formData;
  if (gitOpsModeEnabled) {
    if (!repoURL) {
      errs.repository_url =
        "Git repository URL is required when GitOps mode is enabled";
    } else if (!validUrl({ url: repoURL, protocols: ["http", "https"] })) {
      errs.repository_url = "Git repository URL must be a valid URL";
    }
  }
  return errs;
};

const ChangeManagement = () => {
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [formData, setFormData] = useState<IChangeManagementFormData>({
    // dummy 0 values, will be populated with fresh config API response
    gitOpsModeEnabled: false,
    repoURL: "",
  });
  const [formErrors, setFormErrors] = useState<IChangeManagementFormErrors>({});
  const [isUpdating, setIsUpdating] = useState(false);

  const {
    isLoading: isLoadingConfig,
    error: isLoadingConfigError,
    refetch: refetchConfig,
  } = useQuery<IConfig, Error, IConfig>(
    ["integrations"],
    () => configAPI.loadAll(),
    {
      onSuccess: (data) => {
        const {
          gitops: {
            gitops_mode_enabled: gitOpsModeEnabled,
            repository_url: repoURL,
          },
        } = data;
        setFormData({ gitOpsModeEnabled, repoURL });
        setConfig(data);
      },
    }
  );

  const { isPremiumTier } = useContext(AppContext);

  if (!isPremiumTier) return <PremiumFeatureMessage />;

  const { gitOpsModeEnabled, repoURL } = formData;

  if (isLoadingConfig) {
    return <Spinner />;
  }
  if (isLoadingConfigError) {
    return <DataError />;
  }

  const handleSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    setIsUpdating(true);
    try {
      await configAPI.update({
        gitops: {
          gitops_mode_enabled: formData.gitOpsModeEnabled,
          repository_url: formData.repoURL,
        },
      });
      renderFlash("success", "Successfully updated settings");
      setIsUpdating(false);
      refetchConfig();
    } catch (e) {
      const message = getErrorReason(e);
      renderFlash("error", message || "Failed to update settings");
      setIsUpdating(false);
    }
  };

  const onInputChange = ({ name, value }: IFormField) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    const newErrs = validate(newFormData);
    // only set errors that are updates of existing errors
    // new errors are only set onBlur or submit
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      // @ts-ignore
      if (newErrs[k]) {
        // @ts-ignore
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onInputBlur = () => {
    setFormErrors(validate(formData));
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Change management" />
      <p className={`${baseClass}__page-description`}>
        When using a git repository to manage Fleet, you can optionally put the
        UI in GitOps mode. This prevents you from making changes in the UI that
        would be overridden by GitOps workflows.
      </p>
      <CustomLink
        newTab
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/gitops`}
        text="Learn more about GitOps"
      />
      <form onSubmit={handleSubmit}>
        <Checkbox
          onChange={onInputChange}
          name="gitOpsModeEnabled"
          value={gitOpsModeEnabled}
          parseTarget
        >
          <TooltipWrapper tipContent="GitOps mode is a UI-only setting. API permissions are restricted based on user role.">
            Enable GitOps mode
          </TooltipWrapper>
        </Checkbox>
        {/* Git repository URL */}
        <InputField
          label="Git repository URL"
          onChange={onInputChange}
          name="repoURL"
          value={repoURL}
          parseTarget
          onBlur={onInputBlur}
          error={formErrors.repository_url}
          helpText="When GitOps mode is enabled, you will be directed here to make changes."
          disabled={!gitOpsModeEnabled}
        />
        <Button
          type="submit"
          disabled={!!Object.keys(formErrors).length}
          isLoading={isUpdating}
        >
          Save
        </Button>
      </form>
    </div>
  );
};

export default ChangeManagement;
