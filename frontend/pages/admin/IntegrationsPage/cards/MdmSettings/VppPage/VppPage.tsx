import React, { useCallback, useContext, useRef, useState } from "react";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { IMdmVppToken } from "interfaces/mdm";
import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";

import AddVppModal from "./components/AddVppModal";
import RenewVppModal from "./components/RenewVppModal";
import DeleteVppModal from "./components/DeleteVppModal";
import VppTable from "./components/VppTable";
import EditTeamsVppModal from "./components/EditTeamsVppModal";

const baseClass = "vpp-page";

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
  const { config, isPremiumTier } = useContext(AppContext);

  const [showRenewModal, setShowRenewModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddVppModal, setShowAddVppModal] = useState(false);
  const [showEditTeamsModal, setShowEditTeamsModal] = useState(false);

  const selectedToken = useRef<IMdmVppToken | null>(null);

  const {
    data: vppTokens,
    error: errorVppTokens,
    isLoading,
    refetch,
  } = useQuery<IGetVppTokensResponse, AxiosError, IMdmVppToken[]>(
    ["vpp_tokens"],
    () => mdmAppleAPI.getVppTokens(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
      select: (data) => data.vpp_tokens,
      enabled: isPremiumTier,
    }
  );

  const onEditTokenTeams = (token: IMdmVppToken) => {
    selectedToken.current = token;
    setShowEditTeamsModal(true);
  };

  const onCancelEditTokenTeams = useCallback(() => {
    selectedToken.current = null;
    setShowEditTeamsModal(false);
  }, []);

  const onEditedTeams = useCallback(() => {
    selectedToken.current = null;
    refetch();
    setShowEditTeamsModal(false);
  }, [refetch]);

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
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (!config?.mdm.enabled_and_configured) {
      return (
        <TurnOnMdmMessage
          router={router}
          header="Turn on Apple MDM"
          info=" To install Apple App Store apps purchased through Apple Business
        Manager, first turn on Apple MDM."
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
            onEditTokenTeam={onEditTokenTeams}
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
            {isPremiumTier &&
              vppTokens?.length !== 0 &&
              !!config?.mdm.enabled_and_configured && (
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
      {showEditTeamsModal && selectedToken.current && (
        <EditTeamsVppModal
          currentToken={selectedToken.current}
          tokens={vppTokens || []}
          onCancel={onCancelEditTokenTeams}
          onSuccess={onEditedTeams}
        />
      )}
    </MainContent>
  );
};

export default VppPage;
