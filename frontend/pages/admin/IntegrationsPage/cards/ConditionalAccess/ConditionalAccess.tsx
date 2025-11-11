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

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import Button from "components/buttons/Button";
import { AppContext } from "context/app";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import { useQuery } from "react-query";
import DataError from "components/DataError";
import Modal from "components/Modal";
import { IConfig } from "interfaces/config";

import SectionCard from "../MdmSettings/components/SectionCard";
import EntraConditionalAccessModal from "./components/EntraConditionalAccessModal";
import OktaConditionalAccessModal from "./components/OktaConditionalAccessModal";

const baseClass = "conditional-access";

interface IDeleteConditionalAccessModal {
  toggleDeleteConditionalAccessModal: () => void;
  onDelete: (config: IConfig) => void;
  provider: "microsoft-entra" | "okta";
}

const DeleteConditionalAccessModal = ({
  toggleDeleteConditionalAccessModal,
  onDelete,
  provider,
}: IDeleteConditionalAccessModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);
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

  return (
    <Modal
      title="Delete"
      onExit={toggleDeleteConditionalAccessModal}
      onEnter={handleDelete}
    >
      <>
        <p>
          Fleet will be disconnected from {providerName} and will stop blocking
          end users from logging in with single sign-on.
        </p>
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

enum Phase {
  Loading = "loading",
  ConfirmingConfigured = "confirming-configured",
  ConfirmationError = "confirmation-error",
  Configured = "configured",
  NotConfigured = "not-configured",
}

const ConditionalAccess = () => {
  // HOOKS
  const { renderFlash } = useContext(NotificationContext);

  const { isPremiumTier, setConfig, config: contextConfig } = useContext(
    AppContext
  );

  const [entraPhase, setEntraPhase] = useState<Phase>(Phase.Loading);
  const [isUpdating, setIsUpdating] = useState(false);

  // Modal states
  const [showEntraModal, setShowEntraModal] = useState(false);
  const [showOktaModal, setShowOktaModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteProvider, setDeleteProvider] = useState<
    "microsoft-entra" | "okta"
  >("microsoft-entra");

  // Banner state - shows after form submission, before page refresh
  const [showAwaitingOAuthBanner, setShowAwaitingOAuthBanner] = useState(false);

  // this page is unique in that it triggers a server process that will result in an update to
  // config, but via an endpoint (conditional access) other than the usual PATCH config, so we want
  // to both reference config context AND conditionally (when `isUpdating` from the Configured
  // phase) access `refetchConfig` and associated useQuery capability

  // see frontend/docs/patterns.md > ### Reading and updating configs for why this is atypical

  useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    enabled: isUpdating,
    onSuccess: (_config) => {
      setConfig(_config);
      setIsUpdating(false);
    },
    ...DEFAULT_USE_QUERY_OPTIONS,
  });

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
    enabled: entraPhase === Phase.ConfirmingConfigured && isPremiumTier,
    onSuccess: ({ configuration_completed, setup_error }) => {
      if (configuration_completed) {
        setEntraPhase(Phase.Configured);
        renderFlash(
          "success",
          "Successfully verified conditional access integration"
        );
      } else {
        setEntraPhase(Phase.NotConfigured);

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
      setEntraPhase(Phase.ConfirmationError);
    },
  });

  const {
    microsoft_entra_tenant_id: entraTenantId,
    microsoft_entra_connection_configured: entraConfigured,
    okta_idp_id: oktaIdpId,
  } = contextConfig?.conditional_access || {};

  // Determine if Okta is configured (all 4 fields must be present)
  const oktaConfigured = !!(
    oktaIdpId &&
    contextConfig?.conditional_access?.okta_assertion_consumer_service_url &&
    contextConfig?.conditional_access?.okta_audience_uri &&
    contextConfig?.conditional_access?.okta_certificate
  );

  // Check if this is a managed cloud deployment (Microsoft Entra requires proxy infrastructure)
  const isManagedCloud = contextConfig?.license?.managed_cloud || false;

  // Check Entra configuration state
  useEffect(() => {
    // Don't check config if we're showing the awaiting OAuth banner
    if (showAwaitingOAuthBanner) {
      setEntraPhase(Phase.NotConfigured);
      return;
    }

    if (entraTenantId) {
      if (!entraConfigured) {
        setEntraPhase(Phase.ConfirmingConfigured);
      } else {
        // tenant id is present and connection is configured
        setEntraPhase(Phase.Configured);
      }
    } else {
      setEntraPhase(Phase.NotConfigured);
    }
  }, [entraTenantId, entraConfigured, showAwaitingOAuthBanner]);

  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
  }

  // HANDLERS

  const toggleDeleteModal = () => {
    setShowDeleteModal(!showDeleteModal);
  };

  const handleEntraConnect = () => {
    setShowEntraModal(true);
  };

  const handleEntraModalClose = () => {
    setShowEntraModal(false);
  };

  const handleEntraModalSuccess = () => {
    setShowEntraModal(false);
    // Show banner instead of immediately refetching config
    // Config will be checked when user refreshes the page
    setShowAwaitingOAuthBanner(true);
  };

  const onDeleteConditionalAccess = (updatedConfig: IConfig) => {
    setConfig(updatedConfig);
    setIsUpdating(false);
  };

  const handleOktaConnect = () => {
    setShowOktaModal(true);
  };

  const handleOktaModalClose = () => {
    setShowOktaModal(false);
  };

  const handleOktaModalSuccess = (updatedConfig: IConfig) => {
    setShowOktaModal(false);
    setConfig(updatedConfig);
  };

  const handleEntraDelete = () => {
    setDeleteProvider("microsoft-entra");
    setShowDeleteModal(true);
  };

  const handleOktaDelete = () => {
    setDeleteProvider("okta");
    setShowDeleteModal(true);
  };

  // RENDER

  const renderContent = () => {
    if (entraPhase === Phase.ConfirmingConfigured) {
      return <Spinner />;
    }

    if (entraPhase === Phase.ConfirmationError) {
      return <DataError />;
    }

    // Compute Entra card props to avoid nested ternaries
    const entraIsConfigured = entraPhase === Phase.Configured;

    let entraIconName: IconNames | undefined;
    if (entraIsConfigured) {
      entraIconName = "success";
    } else if (showAwaitingOAuthBanner) {
      entraIconName = "pending-outline";
    }

    let entraCta: JSX.Element | undefined;
    if (entraIsConfigured) {
      entraCta = (
        <Button variant="text-icon" onClick={handleEntraDelete}>
          Delete
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      );
    } else if (!showAwaitingOAuthBanner) {
      entraCta = (
        <Button onClick={handleEntraConnect} isLoading={isUpdating}>
          Connect
        </Button>
      );
    }

    let entraContent: string;
    if (entraIsConfigured) {
      entraContent = "Microsoft Entra conditional access configured";
    } else if (showAwaitingOAuthBanner) {
      entraContent =
        "To complete your integration, follow the instructions in the other tab, then refresh this page to verify.";
    } else {
      entraContent = "Connect Entra to enable conditional access.";
    }

    return (
      <div className={`${baseClass}__cards`}>
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
              <Button onClick={handleOktaConnect}>Connect</Button>
            )
          }
        >
          {oktaConfigured
            ? "Okta conditional access configured"
            : "Connect Okta to enable conditional access."}
        </SectionCard>
        {isManagedCloud && (
          <SectionCard
            header={
              entraIsConfigured || showAwaitingOAuthBanner
                ? undefined
                : "Microsoft Entra"
            }
            iconName={entraIconName}
            cta={entraCta}
          >
            {entraContent}
          </SectionCard>
        )}
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
          onCancel={handleEntraModalClose}
          onSuccess={handleEntraModalSuccess}
        />
      )}
      {showOktaModal && (
        <OktaConditionalAccessModal
          onCancel={handleOktaModalClose}
          onSuccess={handleOktaModalSuccess}
        />
      )}
      {showDeleteModal && (
        <DeleteConditionalAccessModal
          onDelete={onDeleteConditionalAccess}
          toggleDeleteConditionalAccessModal={toggleDeleteModal}
          provider={deleteProvider}
        />
      )}
    </div>
  );
};

export default ConditionalAccess;
