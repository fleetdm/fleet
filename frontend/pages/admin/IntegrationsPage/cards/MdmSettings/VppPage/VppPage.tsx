import React, { useCallback, useContext, useRef, useState } from "react";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { IMdmVppToken } from "interfaces/mdm";
import mdmAppleAPI from "services/entities/mdm_apple";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import AddVppModal from "./components/AddVppModal";
import RenewVppModal from "./components/RenewVppModal";
import DeleteVppModal from "./components/DeleteVppModal";
import VppTable from "./components/VppTable";

const baseClass = "vpp-page";

interface ITurnOnMdmMessageProps {
  router: InjectedRouter;
}

const TurnOnMdmMessage = ({ router }: ITurnOnMdmMessageProps) => {
  return (
    <div className={`${baseClass}__turn-on-mdm-message`}>
      <h2>Turn on Apple MDM</h2>
      <p>
        To install Apple App Store apps purchased through Apple Business
        Manager, first turn on Apple MDM.
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

interface IAddVppMessageProps {
  onAddVpp: () => void;
}

const AddVppMessage = ({ onAddVpp }: IAddVppMessageProps) => {
  return (
    <div className={`${baseClass}__add-vpp-message`}>
      <h2>Add your VPP</h2>
      <p>
        Install Apple App Store apps purchased through Apple Business Manager.
      </p>
      <Button variant="brand" onClick={onAddVpp}>
        Add VPP
      </Button>
    </div>
  );
};

interface IVppPageProps {
  router: InjectedRouter;
}

const VppPage = ({ router }: IVppPageProps) => {
  const { config } = useContext(AppContext);

  const [showRenewModal, setShowRenewModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddVppModal, setShowAddVppModal] = useState(false);

  const selectedToken = useRef<IMdmVppToken | null>(null);

  const {
    data: vppTokens,
    error: errorVppTokens,
    isLoading,
    isRefetching,
    refetch,
  } = useQuery<IMdmVppToken[], AxiosError>(
    ["abmTokens"],
    () => mdmAppleAPI.getVppTokens(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
    }
  );

  const onEditTokenTeam = (token: IMdmVppToken) => {
    console.log(token);
  };

  const onAddVpp = () => {
    setShowAddVppModal(true);
  };

  const onAdded = () => {
    refetch();
    setShowAddVppModal(false);
  };

  const onRenewToken = (vppToken: IMdmVppToken) => {
    selectedToken.current = vppToken;
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

  const onDeleteToken = (vppToken: IMdmVppToken) => {
    selectedToken.current = vppToken;
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

  const showDataError = errorVppTokens && errorVppTokens.status !== 404;

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

    if (vppTokens?.length === 0) {
      return <AddVppMessage onAddVpp={onAddVpp} />;
    }

    if (vppTokens) {
      return (
        <>
          <p>
            Add your VPP to install Apple App Store apps purchased through Apple
            Business Manager.
          </p>
          <VppTable
            vppTokens={vppTokens}
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
            <h1>Volume Purchasing Program (VPP)</h1>
            {vppTokens?.length !== 0 && !!config?.mdm.enabled_and_configured && (
              <Button variant="brand" onClick={onAddVpp}>
                Add VPP
              </Button>
            )}
          </div>
          <>{renderContent()}</>
        </div>
      </>
      {showAddVppModal && (
        <AddVppModal
          onAdded={onAdded}
          onCancel={() => setShowAddVppModal(false)}
        />
      )}
      {showRenewModal && selectedToken.current && (
        <RenewVppModal
          tokenId={selectedToken.current.id}
          orgName={selectedToken.current.org_name}
          onCancel={onCancelRenewToken}
          onRenewedToken={onRenewed}
        />
      )}
      {showDeleteModal && selectedToken.current && (
        <DeleteVppModal
          orgName={selectedToken.current.org_name}
          tokenId={selectedToken.current.id}
          onCancel={onCancelDeleteToken}
          onDeletedToken={onDeleted}
        />
      )}
    </MainContent>
  );
};

export default VppPage;
