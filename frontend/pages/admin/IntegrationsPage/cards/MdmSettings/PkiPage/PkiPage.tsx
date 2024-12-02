import React, {
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import { noop } from "lodash";

import PATHS from "router/paths";

import { AppContext } from "context/app";

import { IConfig } from "interfaces/config";
import { IPkiCert, IPkiConfig } from "interfaces/pki";

import configApi from "services/entities/config";
import pkiApi, { IPkiListCertsResponse } from "services/entities/pki";

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

  // const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddPkiModal, setShowAddPkiModal] = useState(false);
  const [showEditTemplateModal, setShowEditTemplateModal] = useState(false);

  const selectedPki = useRef<IPkiConfig | null>(null);

  const {
    data: pkiCerts,
    error: errorCerts,
    isLoading,
    isRefetching,
    refetch: refetchCerts,
  } = useQuery<IPkiListCertsResponse, AxiosError, IPkiCert[]>(
    ["pki_certs"],
    () => pkiApi.listCerts(),
    {
      refetchOnWindowFocus: false,
      select: (data) => data.certificates,
      enabled: isPremiumTier,
    }
  );

  const {
    data: pkiConfigs,
    error: errorConfigs,
    isLoading: isLoadingConfigs,
    isRefetching: isRefetchingConfigs,
    refetch: refetchConfigs,
  } = useQuery<IConfig, AxiosError, IPkiConfig[]>(
    ["digicert_pki"],
    () => configApi.loadAll(),
    {
      refetchOnWindowFocus: false,
      select: (data) => data.integrations.digicert_pki || [], // TODO: handle no value
      enabled: isPremiumTier,
    }
  );

  const byPkiName = useMemo(() => {
    const dict: Record<string, IPkiConfig> = {};
    pkiConfigs?.forEach((pki) => {
      dict[pki.pki_name] = pki;
    });
    pkiCerts?.forEach((pki) => {
      if (!dict[pki.name]) {
        dict[pki.name] = { pki_name: pki.name, templates: [] };
      }
    });
    return dict;
  }, [pkiConfigs, pkiCerts]);

  const tableData = useMemo(() => {
    return Object.values(byPkiName);
  }, [byPkiName]);

  const onAdd = () => {
    setShowAddPkiModal(true);
  };

  const onAdded = () => {
    refetchCerts();
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
    refetchConfigs();
    setShowEditTemplateModal(false);
  }, [refetchConfigs]);

  const onDelete = noop;

  // const onDelete = (pkiConfig: IPkiConfig) => {
  //   selectedPki.current = pkiConfig;
  //   setShowDeleteModal(true);
  // };

  // const onCancelDelete = useCallback(() => {
  //   selectedPki.current = null;
  //   setShowDeleteModal(false);
  // }, []);

  // const onDeleted = useCallback(() => {
  //   selectedPki.current = null;
  //   refetchCerts();
  //   refetchConfigs();
  //   setShowDeleteModal(false);
  // }, [refetchCerts, refetchConfigs]);

  // if (isLoading || isRefetching || isLoadingConfigs || isRefetchingConfigs) {
  //   return <Spinner />;
  // }

  const showDataError = errorCerts || errorConfigs;

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

    if (isLoading || isRefetching || isLoadingConfigs || isRefetchingConfigs) {
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

    if (!pkiCerts?.length) {
      return <AddPkiMessage onAddPki={onAdd} />;
    }

    return (
      <>
        <p>To help your end users connect to Wi-Fi, you can add your PKI.</p>
        <PkiTable
          data={tableData}
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
              !!pkiCerts?.length &&
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
      {/* {showDeleteModal && selectedPki.current && (
        <DeletePkiModal
          pkiConfig={selectedPki.current}
          onCancel={onCancelDelete}
          onDeleted={onDeleted}
        />
      )} */}
      {showEditTemplateModal && selectedPki.current && (
        <EditTemplateModal
          byPkiName={byPkiName}
          selectedConfig={selectedPki.current}
          onCancel={onCancelEditTemplate}
          onSuccess={onEditedTemplate}
        />
      )}
    </MainContent>
  );
};

export default PkiPage;
