import React, { useState, useContext, useCallback, useMemo } from "react";
import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import { IConfig } from "interfaces/config";
import {
  IJiraIntegration,
  IZendeskIntegration,
  IIntegration,
  IIntegrationTableData,
  IGlobalIntegrations,
} from "interfaces/integration";
import { IApiError } from "interfaces/errors";

import configAPI from "services/entities/config";

import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import SectionHeader from "components/SectionHeader";

import AddIntegrationModal from "./components/AddIntegrationModal";
import DeleteIntegrationModal from "./components/DeleteIntegrationModal";
import EmptyIntegrationsTable from "./components/EmptyIntegrationsTable";

import {
  generateTableHeaders,
  combineDataSets,
} from "./IntegrationsTableConfig";

const baseClass = "integrations-management";
const noIntegrationsClass = "no-integrations";

const VALIDATION_FAILED_ERROR =
  "There was a problem with the information you provided.";
const BAD_REQUEST_ERROR =
  "Invalid login credentials or URL. Please correct and try again.";
const UNKNOWN_ERROR =
  "We experienced an error when attempting to connect. Please try again later.";

const Integrations = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [showAddIntegrationModal, setShowAddIntegrationModal] = useState(false);
  const [showDeleteIntegrationModal, setShowDeleteIntegrationModal] = useState(
    false
  );
  const [
    integrationEditing,
    setIntegrationEditing,
  ] = useState<IIntegrationTableData>();
  const [isUpdatingIntegration, setIsUpdatingIntegration] = useState(false);
  const [jiraIntegrations, setJiraIntegrations] = useState<
    IJiraIntegration[]
  >();
  const [zendeskIntegrations, setZendeskIntegrations] = useState<
    IZendeskIntegration[]
  >();
  const [testingConnection, setTestingConnection] = useState(false);

  const {
    data: integrations,
    isLoading: isLoadingIntegrations,
    error: loadingIntegrationsError,
    refetch: refetchIntegrations,
  } = useQuery<IConfig, Error, IGlobalIntegrations>(
    ["integrations"],
    () => configAPI.loadAll(),
    {
      select: (data: IConfig) => {
        return data.integrations;
      },
      onSuccess: (data) => {
        if (data) {
          setJiraIntegrations(data.jira);
          setZendeskIntegrations(data.zendesk);
        }
      },
    }
  );

  // TODO: Cleanup useCallbacks, add missing dependencies, use state setter functions, e.g.,
  // `setShowAddIntegrationModal((prevState) => !prevState)`, instead of including state
  // variables as dependencies for toggles, etc.

  const toggleAddIntegrationModal = useCallback(() => {
    setShowAddIntegrationModal(!showAddIntegrationModal);
  }, [showAddIntegrationModal, setShowAddIntegrationModal]);

  const toggleDeleteIntegrationModal = useCallback(
    (integration?: IIntegrationTableData) => {
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

  const onAddSubmit = useCallback(
    (integrationSubmitData: IIntegration[], integrationDestination: string) => {
      // Updates either integrations.jira or integrations.zendesk
      const destination = () => {
        if (integrationDestination === "jira") {
          return {
            jira: integrationSubmitData,
            zendesk: zendeskIntegrations,
          };
        }
        return {
          zendesk: integrationSubmitData,
          jira: jiraIntegrations,
        };
      };

      setTestingConnection(true);
      configAPI
        .update({ integrations: destination() })
        .then(() => {
          renderFlash(
            "success",
            <>
              Successfully added{" "}
              <b>
                {integrationSubmitData[integrationSubmitData.length - 1].url} -{" "}
                {integrationSubmitData[integrationSubmitData.length - 1]
                  .project_key ||
                  integrationSubmitData[integrationSubmitData.length - 1]
                    .group_id}
              </b>
            </>
          );
          toggleAddIntegrationModal();
          refetchIntegrations();
        })
        .catch((addError: { data: IApiError }) => {
          if (addError.data?.message.includes("Validation Failed")) {
            if (
              addError.data?.errors[0].reason.includes(
                "duplicate Jira integration"
              )
            ) {
              renderFlash(
                "error",
                <>
                  Could not add{" "}
                  <b>
                    {
                      integrationSubmitData[integrationSubmitData.length - 1]
                        .url
                    }{" "}
                    -{" "}
                    {integrationSubmitData[integrationSubmitData.length - 1]
                      .project_key ||
                      integrationSubmitData[integrationSubmitData.length - 1]
                        .group_id}
                  </b>
                  . This integration already exists
                </>
              );
            } else {
              renderFlash("error", VALIDATION_FAILED_ERROR);
            }
          } else if (addError.data?.message.includes("Bad request")) {
            renderFlash("error", BAD_REQUEST_ERROR);
          } else if (addError.data?.message.includes("Unknown Error")) {
            renderFlash("error", UNKNOWN_ERROR);
          } else {
            renderFlash(
              "error",
              <>
                Could not add{" "}
                <b>
                  {integrationSubmitData[integrationSubmitData.length - 1].url}
                </b>
                . Please try again.
              </>
            );
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
      const deleteIntegrationDestination = () => {
        if (integrationEditing.type === "jira") {
          integrations?.jira.splice(integrationEditing.originalIndex, 1);
          return configAPI.update({
            integrations: {
              jira: integrations?.jira,
              zendesk: zendeskIntegrations,
            },
          });
        }
        integrations?.zendesk.splice(integrationEditing.originalIndex, 1);
        return configAPI.update({
          integrations: {
            zendesk: integrations?.zendesk,
            jira: jiraIntegrations,
          },
        });
      };
      setIsUpdatingIntegration(true);
      deleteIntegrationDestination()
        .then(() => {
          renderFlash(
            "success",
            <>
              Successfully deleted{" "}
              <b>
                {integrationEditing.url} -{" "}
                {integrationEditing.projectKey ||
                  integrationEditing.groupId?.toString()}
              </b>
            </>
          );
          refetchIntegrations();
        })
        .catch(() => {
          renderFlash(
            "error",
            <>
              Could not delete{" "}
              <b>
                {integrationEditing.url} -{" "}
                {integrationEditing.projectKey ||
                  integrationEditing.groupId?.toString()}
              </b>
              . Please try again.
            </>
          );
        })
        .finally(() => {
          setIsUpdatingIntegration(false);
          toggleDeleteIntegrationModal();
        });
    }
  }, [integrationEditing, toggleDeleteIntegrationModal]);

  const onActionSelection = useCallback(
    (action: string, integration: IIntegrationTableData): void => {
      switch (action) {
        case "delete":
          toggleDeleteIntegrationModal(integration);
          break;
        default:
        // do nothing
      }
    },
    [toggleDeleteIntegrationModal]
  );

  const tableHeaders = useMemo(() => generateTableHeaders(onActionSelection), [
    onActionSelection,
  ]);

  const tableData = useMemo(
    () => combineDataSets(jiraIntegrations || [], zendeskIntegrations || []),
    [jiraIntegrations, zendeskIntegrations]
  );

  return (
    <div className={`${baseClass}`}>
      <SectionHeader title="Ticket destinations" />
      <p className={`${baseClass}__page-description`}>
        Add or edit integrations to create tickets when Fleet detects new
        vulnerabilities.
      </p>
      {loadingIntegrationsError ? (
        <TableDataError />
      ) : (
        <TableContainer
          columnConfigs={tableHeaders}
          data={tableData}
          isLoading={isLoadingIntegrations}
          defaultSortHeader="name"
          defaultSortDirection="asc"
          actionButton={{
            name: "add integration",
            buttonText: "Add integration",
            variant: "default",
            onActionButtonClick: toggleAddIntegrationModal,
            hideButton: !tableData?.length,
          }}
          resultsTitle="integrations"
          emptyComponent={() => (
            <EmptyIntegrationsTable
              className={noIntegrationsClass}
              onActionButtonClick={toggleAddIntegrationModal}
            />
          )}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disablePagination
        />
      )}
      {showAddIntegrationModal && (
        <AddIntegrationModal
          onCancel={toggleAddIntegrationModal}
          onSubmit={onAddSubmit}
          integrations={integrations || { jira: [], zendesk: [] }}
          testingConnection={testingConnection}
        />
      )}
      {showDeleteIntegrationModal && (
        <DeleteIntegrationModal
          onCancel={toggleDeleteIntegrationModal}
          onSubmit={onDeleteSubmit}
          url={integrationEditing?.url || ""}
          projectKey={
            integrationEditing?.projectKey ||
            integrationEditing?.groupId?.toString() ||
            ""
          }
          isUpdatingIntegration={isUpdatingIntegration}
        />
      )}
    </div>
  );
};

export default Integrations;
