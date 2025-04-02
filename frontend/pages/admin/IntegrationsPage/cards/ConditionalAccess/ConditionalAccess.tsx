import React, { useContext, useEffect, useState } from "react";

import { size } from "lodash";

import paths from "router/paths";

import { NotificationContext } from "context/notification";

import conditionalAccessAPI, {
  ConfirmMSConditionalAccessResponse,
} from "services/entities/conditional_access";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import Button from "components/buttons/Button";
import { IFormField } from "interfaces/form_field";
import { AppContext } from "context/app";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import InfoBanner from "components/InfoBanner";
import Icon from "components/Icon";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { useQuery } from "react-query";
// import { getErrorReason } from "interfaces/errors";

const baseClass = "conditional-access";

const msetid = "microsoft_entra_tenant_id";

// conditionas –> UI phases:
// 	- not premium –> -1
// 	- no tenant id –> "form"
// 	- tenant id & form submitted –> "form-submitted"
// 	- tenant id & no consent –> "confirming-consent"
//   - tenant id & consent –> "confirmed"

interface IFormData {
  [msetid]: string;
}

interface IFormErrors {
  [msetid]?: string | null;
}

enum Phase {
  Form = "form",
  FormSubmitted = "form-submitted",
  ConfirmingConfigured = "confirming-configured",
  Configured = "configured",
}

const validate = (formData: IFormData) => {
  const errs: IFormErrors = {};
  if (!formData[msetid]) {
    errs[msetid] = "Tenant ID must be present";
  }
  return errs;
};

const ConditionalAccess = () => {
  const { renderFlash } = useContext(NotificationContext);

  const [phase, setPhase] = useState<Phase>(Phase.Form);

  const [formData, setFormData] = useState<IFormData>({
    // [msetid]: "12345",
    [msetid]: "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  const [isUpdating, setIsUpdating] = useState(false);

  const { isPremiumTier, config } = useContext(AppContext);

  // watch curMsetid coming from `config`, populate initial form state once present
  const {
    microsoft_entra_tenant_id: configMsetId,
    microsoft_entra_connection_configured: configMseConfigured,
  } = config?.conditional_access || {};
  useEffect(() => {
    setFormData({
      [msetid]: configMsetId || "aaaaaaaaaaaaaaaa",
    });
  }, [configMsetId]);

  const {
    isLoading: isConfirmingConfigCompleted,
    error: confirmConfigCompletedError,
    refetch: reConfirmConfigCompleted,
  } = useQuery<
    ConfirmMSConditionalAccessResponse,
    Error,
    ConfirmMSConditionalAccessResponse
  >(["confirmAccess"], conditionalAccessAPI.confirmMicrosoftConditionalAccess, {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: phase === Phase.ConfirmingConfigured,
  });

  // only confirm if id was already present in config, not if use addded to formdata
  if (configMsetId) {
    if (!configMseConfigured) {
      // no admin consent
      setPhase(Phase.ConfirmingConfigured);
      // TODO: call verification endpoint
      // if verified, setPhase(3) and set verification to local config
      // or, setting config with confirmed field should cause page re-render with both fields true,
      // leading to next code block and setting to phase 3
      // PATCH server to set config there as well?
    } else {
      // both configMseId and configMseConfigured are true
      setPhase(Phase.Configured);
    }
  }

  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
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
      const {
        microsoft_authentication_url: msAuthURL,
      } = await conditionalAccessAPI.triggerMicrosoftConditionalAccess(
        formData[msetid]
      );
      setIsUpdating(false);
      setPhase(Phase.FormSubmitted);
      window.open(msAuthURL);
    } catch (e) {
      // const message = getErrorReason(e);
      // renderFlash("error", message || "Failed to update settings");
      // TODO - coordinate with Lucas re what this error will contain
      renderFlash(
        "error",
        "Could not update conditional access integration settings."
      );
      setIsUpdating(false);
    }
  };

  const onInputChange = ({ name, value }: IFormField<string>) => {
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

  const renderForm = () => (
    <form onSubmit={handleSubmit} autoComplete="off">
      <InputField
        label="Microsoft Entra tenant ID"
        helpText={
          <>
            You can find this in your Microsoft Entra admin center.{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/microsoft-entra-setup`}
              text="Learn more"
              newTab
            />
          </>
        }
        onChange={onInputChange}
        name={msetid}
        value={formData[msetid]}
        parseTarget
        onBlur={onInputBlur}
        error={formErrors[msetid]}
      />
      <Button
        type="submit"
        variant="brand"
        disabled={!!size(formErrors)}
        className="button-wrap"
        isLoading={isUpdating}
      >
        Save
      </Button>
    </form>
  );

  const onDelete = () => {
    // TODO
  };

  const renderContent = () => {
    switch (phase) {
      case Phase.Form:
        return renderForm();
      case Phase.FormSubmitted:
        return (
          // TODO - confirm border color
          <InfoBanner>
            To complete your integration, follow the instructions in the other
            tab, then refresh this page to verify.
          </InfoBanner>
        );
      case Phase.ConfirmingConfigured:
        // checking integration
        return <Spinner />;
      case Phase.Configured:
        return (
          // TODO - confirm border color
          <InfoBanner color="grey" className={`${baseClass}__success`}>
            <div className="tenant-id">
              <Icon name="success" />
              <b>Microsoft Entra tenant ID:</b>{" "}
              {/* TODO - address buginess with truncation –> tooltip enabling */}
              <TooltipTruncatedText value={formData[msetid]} />
            </div>
            {/* TODO - ensure delete button doesn't get pushed out of banner */}
            <Button
              // className={`${baseClass}__delete-mse-integration`}
              variant="text-icon"
              onClick={onDelete}
            >
              Delete
              <Icon name="trash" color="ui-fleet-black-75" />
            </Button>
          </InfoBanner>
        );
      default:
        return <Spinner />;
    }
  };
  return (
    <div className={baseClass}>
      <SectionHeader title="Conditional access" />
      <p className={`${baseClass}__page-description`}>
        Block hosts failing any policies from logging into third party apps.
        Enable or disable on the{" "}
        <CustomLink url={paths.MANAGE_POLICIES} text="Policies" /> page.
      </p>
      {renderContent()}
    </div>
  );
};

export default ConditionalAccess;
