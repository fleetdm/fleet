import React, { useState, useContext, useCallback } from "react";
import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import { IConfigNested } from "interfaces/config";
import {
  IJiraIntegration,
  IJiraIntegrationIndexed,
  IJiraIntegrationFormErrors,
} from "interfaces/integration";
import { IApiError } from "interfaces/errors";

import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import { DEFAULT_CREATE_INTEGRATION_ERRORS } from "utilities/constants";

import configAPI from "services/entities/config";

import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
import AddIntegrationModal from "./components/CreateIntegrationModal";
import DeleteIntegrationModal from "./components/DeleteIntegrationModal";
import EditIntegrationModal from "./components/EditIntegrationModal";

import {
  generateTableHeaders,
  generateDataSet,
} from "./IntegrationsTableConfig";

const baseClass = "integrations-management";
const noIntegrationsClass = "no-integrations";

const IntegrationsPage = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [showAddIntegrationModal, setShowAddIntegrationModal] = useState(false);
  const [showDeleteIntegrationModal, setShowDeleteIntegrationModal] = useState(
    false
  );
  const [showEditIntegrationModal, setShowEditIntegrationModal] = useState(
    false
  );
  const [
    integrationEditing,
    setIntegrationEditing,
  ] = useState<IJiraIntegrationIndexed>();
  const [integrationsIndexed, setIntegrationsIndexed] = useState<
    IJiraIntegrationIndexed[]
  >();
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const [
    createIntegrationError,
    setCreateIntegrationError,
  ] = useState<IJiraIntegrationFormErrors>(DEFAULT_CREATE_INTEGRATION_ERRORS);
  const [testingConnection, setTestingConnection] = useState<boolean>(false);

  const {
    data: integrations,
    isLoading: isLoadingIntegrations,
    error: loadingIntegrationsError,
    refetch: refetchIntegrations,
  } = useQuery<IConfigNested, Error, IJiraIntegration[]>(
    ["integrations"],
    () => configAPI.loadAll(),
    {
      select: (data: IConfigNested) => {
        return data.integrations.jira;
      },
      onSuccess: (data) => {
        if (data) {
          const addIndex = data.map((integration, index) => {
            return { ...integration, integrationIndex: index };
          });
          setIntegrationsIndexed(addIndex);
        }
      },
    }
  );

  const toggleAddIntegrationModal = useCallback(() => {
    setShowAddIntegrationModal(!showAddIntegrationModal);
    setBackendValidators({});
  }, [
    showAddIntegrationModal,
    setShowAddIntegrationModal,
    setBackendValidators,
  ]);

  const toggleDeleteIntegrationModal = useCallback(
    (integration?: IJiraIntegrationIndexed) => {
      setShowDeleteIntegrationModal(!showDeleteIntegrationModal);
      integration
        ? setIntegrationEditing(integration)
        : setIntegrationEditing(undefined);
    },
    [
      showDeleteIntegrationModal,
      setShowDeleteIntegrationModal,
      setIntegrationEditing,
    ]
  );

  const toggleEditIntegrationModal = useCallback(
    (integration?: IJiraIntegrationIndexed) => {
      setShowEditIntegrationModal(!showEditIntegrationModal);
      setBackendValidators({});
      integration
        ? setIntegrationEditing(integration)
        : setIntegrationEditing(undefined);
    },
    [
      showEditIntegrationModal,
      setShowEditIntegrationModal,
      setIntegrationEditing,
      setBackendValidators,
    ]
  );

  const onCreateSubmit = useCallback(
    (jiraIntegrationSubmitData: IJiraIntegration[]) => {
      setTestingConnection(true);
      configAPI
        .update({ integrations: { jira: jiraIntegrationSubmitData } })
        .then(() => {
          renderFlash(
            "success",
            <>
              Successfully added{" "}
              <b>
                {
                  jiraIntegrationSubmitData[
                    jiraIntegrationSubmitData.length - 1
                  ].url
                }
              </b>
            </>
          );
          setBackendValidators({});
          toggleAddIntegrationModal();
          refetchIntegrations();
        })
        .catch((createError: { data: IApiError }) => {
          if (createError.data.errors[0].reason.includes("Duplicate")) {
            setBackendValidators({
              name: "A team with this name already exists", // TODO: Any backend errors here
            });
          } else {
            renderFlash(
              "error",
              <>
                Could not add{" "}
                <b>
                  {
                    jiraIntegrationSubmitData[
                      jiraIntegrationSubmitData.length - 1
                    ].url
                  }
                </b>
                . Please try again.
              </>
            );
            toggleAddIntegrationModal();
          }
        })
        .finally(() => {
          setTestingConnection(false);
        });
    },
    [toggleAddIntegrationModal]
  );

  const onDeleteSubmit = useCallback(() => {
    if (integrationEditing) {
      integrations?.splice(integrationEditing.integrationIndex, 1);
      configAPI
        .update({ integrations: { jira: integrations } })
        .then(() => {
          renderFlash(
            "success",
            <>
              Successfully deleted <b>{integrationEditing.url}</b>
            </>
          );
        })
        .catch(() => {
          renderFlash(
            "error",
            <>
              Could not delete <b>{integrationEditing.url}</b>. Please try
              again.
            </>
          );
        })
        .finally(() => {
          refetchIntegrations();
          toggleDeleteIntegrationModal();
        });
    }
  }, [integrationEditing, toggleDeleteIntegrationModal]);

  const onEditSubmit = useCallback(
    (jiraIntegrationSubmitData: IJiraIntegration[]) => {
      console.log(
        "onEditSubmit data \njiraIntegrationSubmitData:",
        jiraIntegrationSubmitData
      );

      if (integrationEditing) {
        setTestingConnection(true);
        configAPI
          .update({ integrations: { jira: jiraIntegrationSubmitData } })
          .then(() => {
            renderFlash(
              "success",
              <>
                Successfully edited{" "}
                <b>
                  {
                    jiraIntegrationSubmitData[
                      integrationEditing?.integrationIndex
                    ].url
                  }
                </b>
              </>
            );
            setBackendValidators({});
            setTestingConnection(false);
            setShowEditIntegrationModal(false);
            refetchIntegrations();
          })
          .catch((updateError: { data: IApiError }) => {
            console.error(updateError);
            if (updateError.data.errors[0].reason.includes("Duplicate")) {
              setBackendValidators({
                name: "A team with this name already exists", // TODO: Any backend errors here
              });
            } else {
              renderFlash(
                "error",
                <>
                  Could not edit <b>{integrationEditing?.url}</b>. Please try
                  again.
                </>
              );
            }
          })
          .finally(() => {
            setTestingConnection(false);
          });
      }
    },
    [integrationEditing, toggleEditIntegrationModal]
  );

  const onActionSelection = (
    action: string,
    integration: IJiraIntegrationIndexed
  ): void => {
    console.log(
      "\nonActionSelection in Table:\naction:",
      action,
      "\nintegration",
      integration
    );
    switch (action) {
      case "edit":
        toggleEditIntegrationModal(integration);
        break;
      case "delete":
        toggleDeleteIntegrationModal(integration);
        break;
      default:
    }
  };

  const NoIntegrationsComponent = () => {
    return (
      <div className={`${noIntegrationsClass}`}>
        <div className={`${noIntegrationsClass}__inner`}>
          <div className={`${noIntegrationsClass}__inner-text`}>
            <h1>Set up integrations</h1>
            <p>
              Create tickets automatically when Fleet detects new
              vulnerabilities.
            </p>
            <p>
              Want to learn more?&nbsp;
              <a
                href="https://fleetdm.com/docs/using-fleet/automations"
                target="_blank"
                rel="noopener noreferrer"
              >
                Read about automations&nbsp;
                <FleetIcon name="external-link" />
              </a>
            </p>
            <Button
              variant="brand"
              className={`${noIntegrationsClass}__create-button`}
              onClick={toggleAddIntegrationModal}
            >
              Add integration
            </Button>
          </div>
        </div>
      </div>
    );
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const tableData = integrationsIndexed
    ? generateDataSet(integrationsIndexed)
    : [];

  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Add or edit integrations to create tickets when Fleet detects new
        vulnerabilities.
      </p>
      {loadingIntegrationsError ? (
        <TableDataError />
      ) : (
        <TableContainer
          columns={tableHeaders}
          data={tableData}
          isLoading={isLoadingIntegrations}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          actionButtonText={"Add integration"}
          actionButtonVariant={"brand"}
          hideActionButton={!integrations || integrations.length === 0}
          onActionButtonClick={toggleAddIntegrationModal}
          resultsTitle={"integrations"}
          emptyComponent={NoIntegrationsComponent}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disablePagination
        />
      )}
      {showAddIntegrationModal && (
        <AddIntegrationModal
          onCancel={toggleAddIntegrationModal}
          onSubmit={onCreateSubmit}
          backendValidators={backendValidators}
          integrations={integrations || []}
          testingConnection={testingConnection}
        />
      )}
      {showDeleteIntegrationModal && (
        <DeleteIntegrationModal
          onCancel={toggleDeleteIntegrationModal}
          onSubmit={onDeleteSubmit}
          url={integrationEditing?.url || ""}
        />
      )}
      {showEditIntegrationModal && (
        <EditIntegrationModal
          onCancel={toggleEditIntegrationModal}
          onSubmit={onEditSubmit}
          backendValidators={backendValidators}
          integrations={integrations || []}
          integrationEditing={integrationEditing}
          testingConnection={testingConnection}
        />
      )}
    </div>
  );
};

export default IntegrationsPage;
