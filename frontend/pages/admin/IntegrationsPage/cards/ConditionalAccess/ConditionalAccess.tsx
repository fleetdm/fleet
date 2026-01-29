import React, { useContext, useEffect, useState } from "react";

import paths from "router/paths";

import { NotificationContext } from "context/notification";

import conditionalAccessAPI, {
  ConfirmMSConditionalAccessResponse,
} from "services/entities/conditional_access";
import configAPI from "services/entities/config";

import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import Button from "components/buttons/Button";
import { AppContext } from "context/app";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import { useQuery } from "react-query";
import DataError from "components/DataError";
import Modal from "components/Modal";
import { IConfig, isOktaConditionalAccessConfigured } from "interfaces/config";

import SectionCard from "../MdmSettings/components/SectionCard";
import EntraConditionalAccessModal from "./components/EntraConditionalAccessModal";
import OktaConditionalAccessModal from "./components/OktaConditionalAccessModal";

const baseClass = "conditional-access";

interface IDeleteConditionalAccessModal {
  toggleDeleteConditionalAccessModal: () => void;
  onDelete: (config: IConfig) => void;
  provider: "microsoft-entra" | "okta";
  config: IConfig | null;
}

const DeleteConditionalAccessModal = ({
  toggleDeleteConditionalAccessModal,
  onDelete,
  provider,
  config,
}: IDeleteConditionalAccessModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isDeleting, setIsDeleting] = useState(false);

  const providerName =
    provider === "microsoft-entra" ? "Microsoft Entra" : "Okta";

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      let updatedConfig;
      if (provider === "microsoft-entra") {
        await conditionalAccessAPI.deleteMicrosoftConditionalAccess();
        updatedConfig = await configAPI.loadAll();
      } else {
        // For Okta, clear all fields via config API
        updatedConfig = await configAPI.update({
          conditional_access: {
            okta_idp_id: "",
            okta_assertion_consumer_service_url: "",
            okta_audience_uri: "",
            okta_certificate: "",
            // Preserve existing Microsoft Entra settings
            microsoft_entra_tenant_id:
              config?.conditional_access?.microsoft_entra_tenant_id || "",
            microsoft_entra_connection_configured:
              config?.conditional_access
                ?.microsoft_entra_connection_configured || false,
          },
        });
      }
      renderFlash("success", `Successfully disconnected from ${providerName}.`);
      toggleDeleteConditionalAccessModal();
      onDelete(updatedConfig);
    } catch {
      renderFlash(
        "error",
        `Could not disconnect from ${providerName}, please try again.`
      );
    }
    setIsDeleting(false);
  };

  const copy =
    provider === "microsoft-entra" ? (
      <>
        <p>
          Before you delete, first unblock all end users.{" "}
          <CustomLink
            text="Learn how"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/disable-entra-conditional-access`}
            newTab
          />
        </p>
        <p>
          If you don&apos;t, end users will stay blocked even after deleting
          Entra.
        </p>
      </>
    ) : (
      <>
        <p>
          Before you delete, first unblock all end users.{" "}
          <CustomLink
            text="Learn how"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/disable-okta-conditional-access`}
            newTab
          />
        </p>
        <p>
          If you don&apos;t, end users will stay blocked even after deleting
          Okta.
        </p>
      </>
    );

  return (
    <Modal
      title="Delete"
      onExit={toggleDeleteConditionalAccessModal}
      onEnter={handleDelete}
    >
      <>
        {copy}
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={handleDelete}
            isLoading={isDeleting}
            disabled={isDeleting}
          >
            Delete
          </Button>
          <Button
            onClick={toggleDeleteConditionalAccessModal}
            variant="inverse-alert"
            disabled={isDeleting}
          >
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

enum EntraPhase {
  NotConfigured = "not-configured",
  ConfirmingConfigured = "confirming-configured",
  ConfirmationError = "confirmation-error",
  AwaitingOAuth = "awaiting-oauth",
  Configured = "configured",
  ConsentFailed = "consent-failed",
}

const ConditionalAccess = () => {
  // HOOKS
  const { renderFlash } = useContext(NotificationContext);

  const { isPremiumTier, setConfig, config } = useContext(AppContext);

  const [entraPhase, setEntraPhase] = useState<EntraPhase>(
    EntraPhase.NotConfigured
  );

  // Modal states
  const [showEntraModal, setShowEntraModal] = useState(false);
  const [showOktaModal, setShowOktaModal] = useState(false);
  const [providerToDelete, setProviderToDelete] = useState<
    "microsoft-entra" | "okta" | null
  >(null);

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
    enabled: entraPhase === EntraPhase.ConfirmingConfigured && isPremiumTier,
    onSuccess: ({ configuration_completed, setup_error }) => {
      if (configuration_completed) {
        setEntraPhase(EntraPhase.Configured);
        renderFlash(
          "success",
          "Successfully verified Microsoft Entra conditional access integration"
        );
      } else {
        setEntraPhase(EntraPhase.ConsentFailed);
        if (
          // IT admin did not complete the consent.
          !setup_error ||
          // IT admin clicked "Cancel" in the consent dialog.
          setup_error.includes(
            "A Microsoft Entra admin did not consent to the permissions requested by the conditional access integration"
          )
        ) {
          renderFlash(
            "error",
            "Couldn't update. Fleet didn't get permissions for Entra. Please try again and accept the permissions."
          );
        } else if (
          setup_error.includes(
            'No "Fleet conditional access" Entra ID group was found'
          )
        ) {
          renderFlash(
            "error",
            `Couldn't connect. The "Fleet conditional access" group doesn't exist in Entra. Please create the group and try again.`
          );
        } else {
          // For other kind of errors we just show a generic error.
          // We won't render the error as is because the error comes from the MS proxy and they may be too big or unformatted
          // to display in the banner.
          //
          // For troubleshooting:
          //  - The API response contains the setup_error.
          //  - The Fleet server logs the error.
          //  - The MS proxy stores the error in its database.
          renderFlash(
            "error",
            "Couldn't connect. Please contact your Fleet administrator."
          );
        }
      }
    },
    onError: () => {
      // distinct from successful confirmation response of `false`, this handles an API error
      setEntraPhase(EntraPhase.ConfirmationError);
    },
  });

  const {
    microsoft_entra_tenant_id: entraTenantId,
    microsoft_entra_connection_configured: entraConfigured,
  } = config?.conditional_access || {};

  const oktaConfigured = isOktaConditionalAccessConfigured(config);

  // Check if this is a managed cloud deployment (Microsoft Entra requires proxy infrastructure)
  const isManagedCloud = config?.license?.managed_cloud || false;

  // Check Entra configuration state
  // Note: entraPhase is intentionally included in the dependency array to allow
  // manual phase overrides (e.g., AwaitingOAuth) to persist until config changes
  useEffect(() => {
    const finalStates = [
      EntraPhase.AwaitingOAuth, // Don't check config if we're in AwaitingOAuth phase
      EntraPhase.ConfirmationError, // Don't do confirm call if we are in a final error state
      EntraPhase.ConsentFailed, // Don't do confirm call if after tenant ID provided, something went wrong
    ];

    if (finalStates.includes(entraPhase)) {
      return;
    }

    // Don't override if we just successfully confirmed (phase is Configured but config not yet updated)
    // However, if the tenant ID is removed (deleted), we should still update to NotConfigured
    if (
      entraPhase === EntraPhase.Configured &&
      !entraConfigured &&
      entraTenantId
    ) {
      return;
    }

    if (entraTenantId) {
      if (!entraConfigured) {
        setEntraPhase(EntraPhase.ConfirmingConfigured);
      } else {
        // tenant id is present and connection is configured
        setEntraPhase(EntraPhase.Configured);
      }
    } else {
      setEntraPhase(EntraPhase.NotConfigured);
    }
  }, [entraTenantId, entraConfigured, entraPhase]);

  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
  }

  // HANDLERS

  const toggleDeleteModal = () => {
    setProviderToDelete(null);
  };

  const toggleEntraModal = () => {
    setShowEntraModal(!showEntraModal);
  };

  const handleEntraModalSuccess = () => {
    setShowEntraModal(false);
    // Set phase to awaiting OAuth instead of immediately refetching config
    // Config will be checked when user refreshes the page
    setEntraPhase(EntraPhase.AwaitingOAuth);
  };

  const onDeleteConditionalAccess = (updatedConfig: IConfig) => {
    setConfig(updatedConfig);
  };

  const toggleOktaModal = () => {
    setShowOktaModal(!showOktaModal);
  };

  const handleOktaModalSuccess = (updatedConfig: IConfig) => {
    setShowOktaModal(false);
    setConfig(updatedConfig);
  };

  const handleEntraDelete = () => {
    setProviderToDelete("microsoft-entra");
  };

  const handleOktaDelete = () => {
    setProviderToDelete("okta");
  };

  // RENDER

  const renderOktaContent = () => {
    return (
      <SectionCard
        header={oktaConfigured ? undefined : "Okta"}
        iconName={oktaConfigured ? "success" : undefined}
        cta={
          oktaConfigured ? (
            <Button variant="text-icon" onClick={handleOktaDelete}>
              Delete
              <Icon name="trash" color="ui-fleet-black-75" />
            </Button>
          ) : (
            <Button onClick={toggleOktaModal}>Connect</Button>
          )
        }
      >
        {oktaConfigured
          ? "Okta conditional access configured"
          : "Connect Okta to enable conditional access."}
      </SectionCard>
    );
  };

  const renderEntraContent = () => {
    if (entraPhase === EntraPhase.ConfirmingConfigured) {
      return (
        <SectionCard header="Microsoft Entra">
          <Spinner />
        </SectionCard>
      );
    }

    if (entraPhase === EntraPhase.ConfirmationError) {
      return (
        <SectionCard header="Microsoft Entra">
          <DataError />
        </SectionCard>
      );
    }

    // Compute Entra card props to avoid nested ternaries
    const entraIsConfigured = entraPhase === EntraPhase.Configured;
    const entraIsAwaitingOAuth = entraPhase === EntraPhase.AwaitingOAuth;

    let entraIconName: IconNames | undefined;
    if (entraIsConfigured) {
      entraIconName = "success";
    } else if (entraIsAwaitingOAuth) {
      entraIconName = "pending-outline";
    }

    let entraCta: React.JSX.Element | undefined;
    if (entraIsConfigured) {
      entraCta = (
        <Button variant="text-icon" onClick={handleEntraDelete}>
          Delete
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      );
    } else if (!entraIsAwaitingOAuth) {
      entraCta = <Button onClick={toggleEntraModal}>Connect</Button>;
    }

    let entraContent: string;
    if (entraIsConfigured) {
      entraContent = "Microsoft Entra conditional access configured";
    } else if (entraIsAwaitingOAuth) {
      entraContent =
        "To complete your integration, follow the instructions in the other tab, then refresh this page to verify.";
    } else {
      entraContent = "Connect Entra to enable conditional access.";
    }

    return (
      <SectionCard
        header={
          entraIsConfigured || entraIsAwaitingOAuth
            ? undefined
            : "Microsoft Entra"
        }
        iconName={entraIconName}
        cta={entraCta}
      >
        {entraContent}
      </SectionCard>
    );
  };

  const renderContent = () => {
    return (
      <div className={`${baseClass}__cards`}>
        {renderOktaContent()}
        {isManagedCloud && renderEntraContent()}
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Conditional access" />
      <p className={`${baseClass}__page-description`}>
        Block hosts failing policies from logging in with single sign-on. Once
        connected, enable or disable on the{" "}
        <CustomLink url={paths.MANAGE_POLICIES} text="Policies" /> page.
      </p>
      {renderContent()}
      {showEntraModal && (
        <EntraConditionalAccessModal
          onCancel={toggleEntraModal}
          onSuccess={handleEntraModalSuccess}
        />
      )}
      {showOktaModal && (
        <OktaConditionalAccessModal
          onCancel={toggleOktaModal}
          onSuccess={handleOktaModalSuccess}
        />
      )}
      {providerToDelete && (
        <DeleteConditionalAccessModal
          onDelete={onDeleteConditionalAccess}
          toggleDeleteConditionalAccessModal={toggleDeleteModal}
          provider={providerToDelete}
          config={config}
        />
      )}
    </div>
  );
};

export default ConditionalAccess;
