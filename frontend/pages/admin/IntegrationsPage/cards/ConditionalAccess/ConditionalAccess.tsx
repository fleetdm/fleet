import React, { useContext, useEffect, useState } from "react";

import { size } from "lodash";

import paths from "router/paths";

import { NotificationContext } from "context/notification";

import conditionalAccessAPI, {
  ConfirmMSConditionalAccessResponse,
} from "services/entities/conditional_access";
import configAPI from "services/entities/config";

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
import DataError from "components/DataError";
import Modal from "components/Modal";
import { IConfig } from "interfaces/config";

const baseClass = "conditional-access";

const MSETID = "microsoft_entra_tenant_id";

interface IDeleteConditionalAccessModal {
  toggleDeleteConditionalAccessModal: () => void;
  onDelete: () => void;
  isUpdating: boolean;
}

const DeleteConditionalAccessModal = ({
  toggleDeleteConditionalAccessModal,
  onDelete,
  isUpdating,
}: IDeleteConditionalAccessModal) => {
  return (
    <Modal
      title="Delete"
      onExit={toggleDeleteConditionalAccessModal}
      onEnter={onDelete}
    >
      <>
        <p>
          Fleet will be disconnected from Microsoft Entra and will stop blocking
          end users from logging in with single sign-on.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onDelete}
            isLoading={isUpdating}
          >
            Delete
          </Button>
          <Button
            onClick={toggleDeleteConditionalAccessModal}
            variant="inverse-alert"
          >
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

// conditions –> UI phases:
// 	- no config.tenant id –> "form"
//  - config.tenant id:
//    - and config.confirmed –> "configured"
//    - not config.confirmed –> "confirming-configured", hit confirmation endpoint
//      - confirmation endpoint returns false –> "form", prefilled with current tid
//      - confirmation endpoint returns true –> "configured"
//      - conf ep returns error –> DataError, under header
// 	- form submitted –> "form-submitted", new tab to MS stuff
//

interface IFormData {
  [MSETID]: string;
}

interface IFormErrors {
  [MSETID]?: string | null;
}

enum Phase {
  Form = "form",
  FormSubmitted = "form-submitted",
  ConfirmingConfigured = "confirming-configured",
  ConfirmationError = "confirmation-error",
  Configured = "configured",
}

const validate = (formData: IFormData) => {
  const errs: IFormErrors = {};
  if (!formData[MSETID]) {
    errs[MSETID] = "Tenant ID must be present";
  }
  return errs;
};

const ConditionalAccess = () => {
  // HOOKS
  const { renderFlash } = useContext(NotificationContext);

  const { isPremiumTier, setConfig, config: contextConfig } = useContext(
    AppContext
  );

  const [phase, setPhase] = useState<Phase>(Phase.Form);
  const [isUpdating, setIsUpdating] = useState(false);

  // this page is unique in that it triggers a server process that will result in an update to
  // config, but via an endpoint (conditional access) other than the usual PATCH config, so we want
  // to both reference config context AND conditionally (when `isUpdating` from the Configured
  // phase) access `refetchConfig` and associated useQuery capability

  // see frontend/docs/patterns.md > ### Reading and updating configs for why this is atypical

  const { refetch: refetchConfig } = useQuery<IConfig, Error, IConfig>(
    ["config"],
    () => configAPI.loadAll(),
    {
      select: (data: IConfig) => data,
      enabled: isUpdating && phase === Phase.Configured,
      onSuccess: (_config) => {
        if (
          !_config?.conditional_access?.microsoft_entra_connection_configured
        ) {
          setPhase(Phase.Form);
        }
        setConfig(_config);
        setIsUpdating(false);
      },
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const [formData, setFormData] = useState<IFormData>({
    [MSETID]:
      contextConfig?.conditional_access?.microsoft_entra_tenant_id || "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  const [
    showDeleteConditionalAccessModal,
    setShowDeleteConditionalAccessModal,
  ] = useState(false);

  // "loading" state here is encompassed by phase === Phase.ConfirmingConfigured state, don't need
  // to use useQuery's
  // "error" state handled by onError callback
  // success state handled by onSuccess callback
  useQuery<
    ConfirmMSConditionalAccessResponse,
    Error,
    ConfirmMSConditionalAccessResponse
  >(["confirmAccess"], conditionalAccessAPI.confirmMicrosoftConditionalAccess, {
    ...DEFAULT_USE_QUERY_OPTIONS,
    // only make this call at the appropriate UI phase
    enabled: phase === Phase.ConfirmingConfigured && isPremiumTier,
    onSuccess: ({ configuration_completed }) => {
      if (configuration_completed) {
        setPhase(Phase.Configured);
        renderFlash(
          "success",
          "Successfully verified conditional access integration"
        );
      } else {
        setPhase(Phase.Form);
        renderFlash(
          "error",
          "Could not verify conditional access integration. Please try connecting again."
        );
      }
    },
    onError: () => {
      // distinct from successful confirmation response of `false`, this handles an API error
      setPhase(Phase.ConfirmationError);
    },
  });

  const {
    microsoft_entra_tenant_id: contextConfigMsetId,
    microsoft_entra_connection_configured: contextConfigMseConfigured,
  } = contextConfig?.conditional_access || {};

  // only checks if tenant id already present in config, not if user added it to the form
  useEffect(() => {
    if (contextConfigMsetId) {
      if (!contextConfigMseConfigured) {
        setPhase(Phase.ConfirmingConfigured);
      } else {
        // tenant id is present and connection is configured
        setPhase(Phase.Configured);
      }
    }
  }, [contextConfigMsetId, contextConfigMseConfigured]);

  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
  }

  // HANDLERS

  const toggleDeleteConditionalAccessModal = () => {
    setShowDeleteConditionalAccessModal(!showDeleteConditionalAccessModal);
  };

  const onSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
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
        formData[MSETID]
      );
      setIsUpdating(false);
      setPhase(Phase.FormSubmitted);
      window.open(msAuthURL);
    } catch (e) {
      renderFlash(
        "error",
        "Could not update conditional access integration settings."
      );
      setIsUpdating(false);
    }
  };

  const onDeleteConditionalAccess = async () => {
    setIsUpdating(true);
    try {
      await conditionalAccessAPI.deleteMicrosoftConditionalAccess();
      renderFlash("success", "Successfully disconnected from Microsoft Entra.");
      toggleDeleteConditionalAccessModal();
      refetchConfig();
    } catch {
      renderFlash(
        "error",
        "Could not disconnect from Microsoft Entra, please try again."
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

  const renderContent = () => {
    switch (phase) {
      case Phase.Form:
        return (
          <form onSubmit={onSubmit} autoComplete="off">
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
              name={MSETID}
              value={formData[MSETID]}
              parseTarget
              onBlur={onInputBlur}
              error={formErrors[MSETID]}
            />
            <Button
              type="submit"
              disabled={!!size(formErrors)}
              className="button-wrap"
              isLoading={isUpdating}
            >
              Save
            </Button>
          </form>
        );
      case Phase.FormSubmitted:
        return (
          <InfoBanner>
            To complete your integration, follow the instructions in the other
            tab, then refresh this page to verify.
          </InfoBanner>
        );
      case Phase.ConfirmingConfigured:
        // checking integration
        return <Spinner />;
      case Phase.ConfirmationError:
        return <DataError />;
      case Phase.Configured:
        return (
          <InfoBanner color="grey" className={`${baseClass}__success`}>
            <div className="tenant-id">
              <Icon name="success" />
              <b>Microsoft Entra tenant ID:</b>{" "}
              <TooltipTruncatedText value={formData[MSETID]} />
            </div>
            <Button
              variant="text-icon"
              onClick={toggleDeleteConditionalAccessModal}
            >
              Delete
              <Icon name="trash" />
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
        Block hosts failing any policies from logging in with single sign-on.
        Enable or disable on the{" "}
        <CustomLink url={paths.MANAGE_POLICIES} text="Policies" /> page.
      </p>
      {renderContent()}
      {showDeleteConditionalAccessModal && (
        <DeleteConditionalAccessModal
          onDelete={onDeleteConditionalAccess}
          toggleDeleteConditionalAccessModal={
            toggleDeleteConditionalAccessModal
          }
          isUpdating={isUpdating}
        />
      )}
    </div>
  );
};

export default ConditionalAccess;
