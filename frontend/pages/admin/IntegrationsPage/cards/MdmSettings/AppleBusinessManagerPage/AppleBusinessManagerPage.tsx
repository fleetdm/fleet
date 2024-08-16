import React, { useCallback, useContext, useState } from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";

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

const baseClass = "apple-business-manager-page";

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

const ButtonWrap = ({
  onClickDisable,
  onClickRenew,
}: {
  onClickDisable: () => void;
  onClickRenew: () => void;
}) => {
  return (
    <div className={`${baseClass}__button-wrap`}>
      <Button variant="inverse" onClick={onClickDisable}>
        Disable automatic enrollment
      </Button>
      <Button variant="brand" onClick={onClickRenew}>
        Renew token
      </Button>
    </div>
  );
};

const AppleBusinessManagerPage = ({ router }: { router: InjectedRouter }) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isUploading, setIsUploading] = useState(false);
  const [showDisableModal, setShowDisableModal] = useState(false);
  const [showRenewModal, setShowRenewModal] = useState(false);
  const [showAddAbmModal, setShowAddAbmModal] = useState(false);

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

  const uploadToken = useCallback(
    async (data: FileList | null) => {
      setIsUploading(true);
      const token = data?.[0];
      if (!token) {
        setIsUploading(false);
        renderFlash("error", "No token selected.");
        return;
      }

      try {
        await mdmAppleBmAPI.uploadToken(token);
        renderFlash(
          "success",
          "Automatic enrollment for macOS hosts is enabled."
        );
        router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
      } catch (e) {
        const msg = getErrorReason(e);
        if (msg.toLowerCase().includes("valid token")) {
          renderFlash("error", msg);
        } else {
          renderFlash("error", "Couldn’t enable. Please try again.");
        }
      } finally {
        setIsUploading(false);
      }
    },
    [renderFlash, router]
  );

  const onClickDisable = useCallback(() => {
    setShowDisableModal(true);
  }, []);

  const onClickRenew = useCallback(() => {
    setShowRenewModal(true);
  }, []);

  const disableAutomaticEnrollment = useCallback(async () => {
    try {
      await mdmAppleBmAPI.disableAutomaticEnrollment();
      renderFlash("success", "Automatic enrollment disabled successfully.");
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
    } catch (e) {
      renderFlash(
        "error",
        "Couldn’t disable automatic enrollment. Please try again."
      );
      setShowDisableModal(false);
    }
  }, [renderFlash, router]);

  const onCancelDisable = useCallback(() => {
    setShowDisableModal(false);
  }, []);

  const onRenew = useCallback(() => {
    refetch();
    setShowRenewModal(false);
  }, [refetch]);

  const onCancelRenew = useCallback(() => {
    setShowRenewModal(false);
  }, []);

  if (isLoading || isRefetching) {
    return <Spinner />;
  }

  const showDataError = errorAbmTokens && errorAbmTokens.status !== 404;
  const showConnectAbm = !abmTokens;

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (showDataError) {
      return (
        <div>
          <DataError />
          <ButtonWrap
            onClickDisable={onClickDisable}
            onClickRenew={onClickRenew}
          />
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
          <AppleBusinessManagerTable abmTokens={abmTokens} />
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
            {abmTokens?.length !== 0 && (
              <Button
                variant="brand"
                onClick={() => {
                  console.log("click add abm");
                }}
              >
                Add ABM
              </Button>
            )}
          </div>
          <>{renderContent()}</>
        </div>
      </>
      {showDisableModal && (
        <DisableAutomaticEnrollmentModal
          onCancel={onCancelDisable}
          onConfirm={disableAutomaticEnrollment}
        />
      )}
      {showRenewModal && (
        <RenewTokenModal onCancel={onCancelRenew} onRenew={onRenew} />
      )}
    </MainContent>
  );
};

export default AppleBusinessManagerPage;
