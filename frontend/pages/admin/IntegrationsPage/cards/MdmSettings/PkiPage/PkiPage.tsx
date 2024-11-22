import React, { useCallback, useContext, useRef, useState } from "react";
import { noop } from "lodash";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";

import { AppContext } from "context/app";

import { IPkiConfig } from "interfaces/pki";

import BackLink from "components/BackLink";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";

import AddPkiModal from "./components/AddPkiModal";
import DeletePkiModal from "./components/DeletePkiModal";
import EditTemplateModal from "./components/EditTemplateModal";
import PkiTable from "./components/PkiTable";

const baseClass = "pki-page";

interface IAddPkiMessageProps {
  onAddPki: () => void;
}

const AddPkiMessage = ({ onAddPki }: IAddPkiMessageProps) => {
  return (
    <div className={`${baseClass}__add-message`}>
      <h2>Add your PKI</h2>
      <p>Help your end users connect to Wi-Fi</p>
      <Button variant="brand" onClick={onAddPki}>
        Add PKI
      </Button>
    </div>
  );
};

const PkiPage = ({ router }: { router: InjectedRouter }) => {
  const { config, isPremiumTier } = useContext(AppContext);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddPkiModal, setShowAddPkiModal] = useState(false);
  const [showEditTemplateModal, setShowEditTemplateModal] = useState(false);

  const selectedPki = useRef<IPkiConfig | null>(null);

  const {
    data: pkiConfigs,
    error: errorPkiConfigs,
    isLoading,
    isRefetching,
    refetch,
  } = useQuery<IPkiConfig[], AxiosError>(
    ["pkiConfigs"],
    () =>
      Promise.resolve([
        {
          name: "test_config",
          templates: [
            {
              profile_id: 1,
              name: "test",
              common_name: "test",
              san: "test",
              seat_id: "test",
            },
          ],
        },
      ]),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
      enabled: isPremiumTier,
    }
  );

  const onAdd = () => {
    setShowAddPkiModal(true);
  };

  const onAdded = () => {
    refetch();
    setShowAddPkiModal(false);
  };

  const onEditTemplate = (pkiConfig: IPkiConfig) => {
    selectedPki.current = pkiConfig;
    setShowEditTemplateModal(true);
  };

  const onCancelEditTemplate = useCallback(() => {
    selectedPki.current = null;
    setShowEditTemplateModal(false);
  }, []);

  const onEditedTemplate = useCallback(() => {
    selectedPki.current = null;
    refetch();
    setShowEditTemplateModal(false);
  }, [refetch]);

  const onDelete = (pkiConfig: IPkiConfig) => {
    selectedPki.current = pkiConfig;
    setShowDeleteModal(true);
  };

  const onCancelDelete = useCallback(() => {
    selectedPki.current = null;
    setShowDeleteModal(false);
  }, []);

  const onDeleted = useCallback(() => {
    selectedPki.current = null;
    refetch();
    setShowDeleteModal(false);
  }, [refetch]);

  if (isLoading || isRefetching) {
    return <Spinner />;
  }

  const showDataError = errorPkiConfigs && errorPkiConfigs.status !== 404;

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

    if (!pkiConfigs?.length) {
      return <AddPkiMessage onAddPki={onAdd} />;
    }

    return (
      <>
        <p>To help your end users connect to Wi-Fi, you can add your PKI.</p>
        <PkiTable
          data={pkiConfigs}
          onEdit={onEditTemplate}
          onDelete={onDelete}
        />
      </>
    );
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
            <h1>Public key infrastructure (PKI)</h1>
            {isPremiumTier &&
              pkiConfigs?.length !== 0 &&
              !!config?.mdm.enabled_and_configured && (
                <Button variant="brand" onClick={onAdd}>
                  Add PKI
                </Button>
              )}
          </div>
          <>{renderContent()}</>
        </div>
      </>
      {showAddPkiModal && (
        <AddPkiModal
          onAdded={onAdded}
          onCancel={() => setShowAddPkiModal(false)}
        />
      )}
      {showDeleteModal && selectedPki.current && (
        <DeletePkiModal
          pkiConfig={selectedPki.current}
          onCancel={onCancelDelete}
          onDeleted={onDeleted}
        />
      )}
      {showEditTemplateModal && selectedPki.current && (
        <EditTemplateModal
          pkiConfig={selectedPki.current}
          onCancel={onCancelEditTemplate}
          onSuccess={onEditedTemplate}
        />
      )}
    </MainContent>
  );
};

export default PkiPage;
