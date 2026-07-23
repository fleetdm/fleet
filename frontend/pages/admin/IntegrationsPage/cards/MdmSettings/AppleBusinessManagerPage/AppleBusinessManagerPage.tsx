import React, { useCallback, useContext, useRef, useState } from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { IMdmAbToken } from "interfaces/mdm";
import mdmAbmAPI, {
  IGetAbTokensResponse,
} from "services/entities/mdm_apple_bm";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import EmptyState from "components/EmptyState";
import { getEarliestExpiry } from "components/App/App";

import AppleBusinessManagerTable from "./components/AppleBusinessManagerTable";
import AddAbmModal from "./components/AddAbmModal";
import RenewAbmModal from "./components/RenewAbmModal";
import DeleteAbmModal from "./components/DeleteAbmModal";
import EditTeamsAbmModal from "./components/EditTeamsAbmModal";

const baseClass = "apple-business-manager-page";

interface IAddAbmMessageProps {
  onAddAbm: () => void;
}

const AddAbmMessage = ({ onAddAbm }: IAddAbmMessageProps) => {
  return (
    <EmptyState
      header="Add your AB"
      info="Automatically enroll newly purchased Apple hosts when they're first unboxed and set up by your end users."
      primaryButton={<Button onClick={onAddAbm}>Add AB</Button>}
    />
  );
};

const AppleBusinessManagerPage = ({ router }: { router: InjectedRouter }) => {
  const { config, isPremiumTier, setABMExpiry } = useContext(AppContext);

  const [showRenewModal, setShowRenewModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddAbmModal, setShowAddAbmModal] = useState(false);
  const [showEditTeamsModal, setShowEditTeamsModal] = useState(false);

  const selectedToken = useRef<IMdmAbToken | null>(null);

  const {
    data: abTokens,
    error: errorAbmTokens,
    isLoading,
    isRefetching,
    refetch,
  } = useQuery<IGetAbTokensResponse, AxiosError, IMdmAbToken[]>(
    ["abTokens"],
    () => mdmAbmAPI.getTokens(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
      select: (data) => data?.ab_tokens,
      onSuccess: (data) => {
        // we need to call setABMExpiry here to update the expiry info so the terms banner
        // displays correctly
        if (data.length === 0) {
          setABMExpiry({
            earliestExpiry: "",
            needsAbmTermsRenewal: false,
            hasAbmTokenInvalid: false,
            invalidAbmTokenOrgNames: [],
          });
        } else {
          setABMExpiry({
            earliestExpiry: getEarliestExpiry(data),
            needsAbmTermsRenewal: data.some((token) => token.terms_expired),
            hasAbmTokenInvalid: data.some((token) => token.token_invalid),
            invalidAbmTokenOrgNames: data
              .filter((token) => token.token_invalid)
              .map((token) => token.org_name),
          });
        }
      },
      enabled: isPremiumTier,
    }
  );

  const onEditTokenTeam = (abmToken: IMdmAbToken) => {
    selectedToken.current = abmToken;
    setShowEditTeamsModal(true);
  };

  const onCancelEditTeam = useCallback(() => {
    selectedToken.current = null;
    setShowEditTeamsModal(false);
  }, []);

  const onEditedTeam = useCallback(() => {
    selectedToken.current = null;
    refetch();
    setShowEditTeamsModal(false);
  }, [refetch]);

  const onAddAbm = () => {
    setShowAddAbmModal(true);
  };

  const onAdded = () => {
    refetch();
    setShowAddAbmModal(false);
  };

  const onRenewToken = (abmToken: IMdmAbToken) => {
    selectedToken.current = abmToken;
    setShowRenewModal(true);
  };

  const onCancelRenewToken = useCallback(() => {
    selectedToken.current = null;
    setShowRenewModal(false);
  }, []);

  const onRenewed = useCallback(() => {
    selectedToken.current = null;
    refetch();
    setShowRenewModal(false);
  }, [refetch]);

  const onDeleteToken = (abmToken: IMdmAbToken) => {
    selectedToken.current = abmToken;
    setShowDeleteModal(true);
  };

  const onCancelDeleteToken = useCallback(() => {
    selectedToken.current = null;
    setShowDeleteModal(false);
  }, []);

  const onDeleted = useCallback(() => {
    selectedToken.current = null;
    refetch();
    setShowDeleteModal(false);
  }, [refetch]);

  if (isLoading || isRefetching) {
    return <Spinner />;
  }

  const showDataError = errorAbmTokens && errorAbmTokens.status !== 404;

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (!config?.mdm.enabled_and_configured) {
      return (
        <EmptyState
          header="Turn on Apple MDM"
          info="To add your AB and enable automatic enrollment for macOS, iOS, and iPadOS hosts, first turn on Apple MDM."
          primaryButton={
            <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
              Turn on
            </Button>
          }
        />
      );
    }

    if (isLoading) {
      return <Spinner />;
    }

    // TODO: error UI
    if (showDataError) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
    }

    if (abTokens?.length === 0) {
      return <AddAbmMessage onAddAbm={onAddAbm} />;
    }

    if (abTokens) {
      return (
        <>
          <p>
            Add your AB to enable automatic enrollment for company-owned hosts
            and enrollment, via a Managed Apple Account, for BYOD hosts.
          </p>
          <AppleBusinessManagerTable
            abTokens={abTokens}
            onEditTokenTeam={onEditTokenTeam}
            onRenewToken={onRenewToken}
            onDeleteToken={onDeleteToken}
          />
        </>
      );
    }

    return null;
  };

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton
            text="Back to MDM"
            path={PATHS.ADMIN_INTEGRATIONS_MDM}
            className={`${baseClass}__back-to-mdm`}
          />
        </div>
        <div className={`${baseClass}__page-content`}>
          <div className={`${baseClass}__page-header-section`}>
            <h1>Apple Business (AB)</h1>
            {isPremiumTier &&
              abTokens?.length !== 0 &&
              !!config?.mdm.enabled_and_configured && (
                <Button onClick={onAddAbm}>Add AB</Button>
              )}
          </div>
          <>{renderContent()}</>
        </div>
      </>
      {showAddAbmModal && (
        <AddAbmModal
          onAdded={onAdded}
          onCancel={() => setShowAddAbmModal(false)}
        />
      )}
      {showRenewModal && selectedToken.current && (
        <RenewAbmModal
          tokenId={selectedToken.current.id}
          onCancel={onCancelRenewToken}
          onRenewedToken={onRenewed}
        />
      )}
      {showDeleteModal && selectedToken.current && (
        <DeleteAbmModal
          tokenOrgName={selectedToken.current.org_name}
          tokenId={selectedToken.current.id}
          onCancel={onCancelDeleteToken}
          onDeletedToken={onDeleted}
        />
      )}
      {showEditTeamsModal && selectedToken.current && (
        <EditTeamsAbmModal
          token={selectedToken.current}
          onCancel={onCancelEditTeam}
          onSuccess={onEditedTeam}
        />
      )}
    </MainContent>
  );
};

export default AppleBusinessManagerPage;
