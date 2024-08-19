import React, { useCallback, useContext, useRef, useState } from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import { IMdmAbmToken, IMdmAppleBm } from "interfaces/mdm";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";

import BackLink from "components/BackLink";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";

import DisableAutomaticEnrollmentModal from "./modals/DisableAutomaticEnrollmentModal";
import RenewTokenModal from "./modals/RenewTokenModal";
import AppleBusinessManagerTable from "./components/AppleBusinessManagerTable";
import AddAbmModal from "./components/AddAbmModal";
import DeleteAbmModal from "./components/DeleteAbmModal";

const baseClass = "apple-business-manager-page";

interface ITurnOnMdmMessageProps {
  router: InjectedRouter;
}

const TurnOnMdmMessage = ({ router }: ITurnOnMdmMessageProps) => {
  return (
    <div className={`${baseClass}__turn-on-mdm-message`}>
      <h2>Turn on Apple MDM</h2>
      <p>
        To add your ABM and enable automatic enrollment for macOS, iOS, and
        iPadOS hosts, first turn on Apple MDM.
      </p>
      <Button
        variant="brand"
        onClick={() => {
          router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
        }}
      >
        Turn on
      </Button>
    </div>
  );
};

const AddAbmMessage = () => {
  return (
    <div className={`${baseClass}__add-adm-message`}>
      <h2>Add your ABM</h2>
      <p>
        Automatically enroll newly purchased Apple hosts when they&apos;re first
        unboxed and set up by your end users.
      </p>
      <Button
        variant="brand"
        onClick={() => {
          console.log("click add abm");
        }}
      >
        Add ABM
      </Button>
    </div>
  );
};

const AppleBusinessManagerPage = ({ router }: { router: InjectedRouter }) => {
  const { config } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [showDisableModal, setShowDisableModal] = useState(false);
  const [showRenewModal, setShowRenewModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddAbmModal, setShowAddAbmModal] = useState(false);

  const selectedToken = useRef<IMdmAbmToken | null>(null);

  const {
    data: abmTokens,
    error: errorAbmTokens,
    isLoading,
    isRefetching,
    refetch,
  } = useQuery<IMdmAbmToken[], AxiosError>(
    ["abmTokens"],
    () => mdmAppleBmAPI.getTokens(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
    }
  );

  const onEditTokenTeam = (abmToken: IMdmAbmToken) => {
    selectedToken.current = abmToken;
    // TODO: Show edit team modal
  };

  const onRenewToken = (abmToken: IMdmAbmToken) => {
    selectedToken.current = abmToken;
    setShowRenewModal(true);
  };

  const onDeleteToken = (abmToken: IMdmAbmToken) => {
    selectedToken.current = abmToken;
    setShowDeleteModal(true);
  };

  const onRenewed = useCallback(() => {
    refetch();
    setShowRenewModal(false);
  }, [refetch]);

  const onCancelRenewToken = useCallback(() => {
    selectedToken.current = null;
    setShowRenewModal(false);
  }, []);

  const onCancelDeleteToken = useCallback(() => {
    selectedToken.current = null;
    setShowDeleteModal(false);
  }, []);

  const onDeleted = useCallback(() => {
    selectedToken.current = null;
    setShowDeleteModal(false);
  }, []);

  if (isLoading || isRefetching) {
    return <Spinner />;
  }

  const showDataError = errorAbmTokens && errorAbmTokens.status !== 404;
  const showConnectAbm = !abmTokens;

  const renderContent = () => {
    if (!config?.mdm.enabled_and_configured) {
      return <TurnOnMdmMessage router={router} />;
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
      return <AddAbmMessage />;
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
            {abmTokens?.length !== 0 && !!config?.mdm.enabled_and_configured && (
              <Button variant="brand" onClick={() => setShowAddAbmModal(true)}>
                Add ABM
              </Button>
            )}
          </div>
          <>{renderContent()}</>
        </div>
      </>
      {showAddAbmModal && (
        <AddAbmModal onCancel={() => setShowAddAbmModal(false)} />
      )}
      {showRenewModal && selectedToken.current && (
        <RenewTokenModal
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
    </MainContent>
  );
};

export default AppleBusinessManagerPage;
