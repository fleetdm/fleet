import React, { useCallback, useContext, useRef, useState } from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { IMdmAbmToken } from "interfaces/mdm";
import mdmAbmAPI, {
  IGetAbmTokensResponse,
} from "services/entities/mdm_apple_bm";

import BackLink from "components/BackLink";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";

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
    <div className={`${baseClass}__add-adm-message`}>
      <h2>Add your ABM</h2>
      <p>
        Automatically enroll newly purchased Apple hosts when they&apos;re first
        unboxed and set up by your end users.
      </p>
      <Button variant="brand" onClick={onAddAbm}>
        Add ABM
      </Button>
    </div>
  );
};

const AppleBusinessManagerPage = ({ router }: { router: InjectedRouter }) => {
  const { config, isPremiumTier } = useContext(AppContext);

  const [showRenewModal, setShowRenewModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddAbmModal, setShowAddAbmModal] = useState(false);
  const [showEditTeamsModal, setShowEditTeamsModal] = useState(false);

  const selectedToken = useRef<IMdmAbmToken | null>(null);

  const {
    data: abmTokens,
    error: errorAbmTokens,
    isLoading,
    isRefetching,
    refetch,
  } = useQuery<IGetAbmTokensResponse, AxiosError, IMdmAbmToken[]>(
    ["abmTokens"],
    () => mdmAbmAPI.getTokens(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
      select: (data) => data?.abm_tokens,
      enabled: isPremiumTier,
    }
  );

  const onEditTokenTeam = (abmToken: IMdmAbmToken) => {
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

  const onRenewToken = (abmToken: IMdmAbmToken) => {
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

  const onDeleteToken = (abmToken: IMdmAbmToken) => {
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
        <TurnOnMdmMessage
          router={router}
          header="Turn on Apple MDM"
          info="To add your ABM and enable automatic enrollment for macOS, iOS, and
        iPadOS hosts, first turn on Apple MDM."
        />
      );
    }

    if (isLoading) {
      return <Spinner />;
    }

    // TODO: error UI
    if (showDataError) {
      return (
        <div>
          <DataError />
        </div>
      );
    }

    if (abmTokens?.length === 0) {
      return <AddAbmMessage onAddAbm={onAddAbm} />;
    }

    if (abmTokens) {
      return (
        <>
          <p>
            Add your ABM to automatically enroll newly purchased Apple hosts
            when they&apos;re first unboxed and set up by your end users.
          </p>
          <AppleBusinessManagerTable
            abmTokens={abmTokens}
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
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
        <div className={`${baseClass}__page-content`}>
          <div className={`${baseClass}__page-header-section`}>
            <h1>Apple Business Manager (ABM)</h1>
            {isPremiumTier &&
              abmTokens?.length !== 0 &&
              !!config?.mdm.enabled_and_configured && (
                <Button variant="brand" onClick={onAddAbm}>
                  Add ABM
                </Button>
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
          orgName={selectedToken.current.org_name}
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
