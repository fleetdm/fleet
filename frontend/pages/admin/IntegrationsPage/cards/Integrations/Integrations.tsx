import React, { useState, useContext, useCallback } from "react";
import { useQuery } from "react-query";
import memoize from "memoize-one";

import { NotificationContext } from "context/notification";
import { IConfig } from "interfaces/config";
import {
  IJiraIntegration,
  IZendeskIntegration,
  IIntegration,
  IIntegrationTableData,
  IIntegrations,
} from "interfaces/integration";
import { IApiError } from "interfaces/errors";
import { IEmptyTableProps } from "interfaces/empty_table";

import Button from "components/buttons/Button";
// @ts-ignore

import configAPI from "services/entities/config";

import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import AddIntegrationModal from "./components/AddIntegrationModal";
import DeleteIntegrationModal from "./components/DeleteIntegrationModal";
import EditIntegrationModal from "./components/EditIntegrationModal";

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
  const [showEditIntegrationModal, setShowEditIntegrationModal] = useState(
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
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const [testingConnection, setTestingConnection] = useState(false);

  const {
    data: integrations,
    isLoading: isLoadingIntegrations,
    error: loadingIntegrationsError,
    refetch: refetchIntegrations,
  } = useQuery<IConfig, Error, IIntegrations>(
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

  const combineJiraAndZendesk = memoize(() => {
    return combineDataSets(jiraIntegrations || [], zendeskIntegrations || []);
  });

  const toggleAddIntegrationModal = useCallback(() => {
    setShowAddIntegrationModal(!showAddIntegrationModal);
    setBackendValidators({});
  }, [
    showAddIntegrationModal,
    setShowAddIntegrationModal,
    setBackendValidators,
  ]);

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

  const toggleEditIntegrationModal = useCallback(
    (integration?: IIntegrationTableData) => {
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

  const onAddSubmit = useCallback(
    (integrationSubmitData: IIntegration[], integrationDestination: string) => {
      // Updates either integrations.jira or integrations.zendesk
      const destination = () => {
        if (integrationDestination === "jira") {
          return { jira: integrationSubmitData, zendesk: zendeskIntegrations };
        }
        return { zendesk: integrationSubmitData, jira: jiraIntegrations };
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
          setBackendValidators({});
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

  const onEditSubmit = useCallback(
    (integrationSubmitData: IIntegration[]) => {
      if (integrationEditing) {
        setTestingConnection(true);

        const editIntegrationDestination = () => {
          if (integrationEditing.type === "jira") {
            return configAPI.update({
              integrations: {
                jira: integrationSubmitData,
                zendesk: zendeskIntegrations,
              },
            });
          }
          return configAPI.update({
            integrations: {
              zendesk: integrationSubmitData,
              jira: jiraIntegrations,
            },
          });
        };

        editIntegrationDestination()
          .then(() => {
            renderFlash(
              "success",
              <>
                Successfully edited{" "}
                <b>
                  {integrationSubmitData[integrationEditing?.originalIndex].url}{" "}
                  -{" "}
                  {integrationSubmitData[integrationEditing?.originalIndex]
                    .project_key ||
                    integrationSubmitData[integrationEditing?.originalIndex]
                      .group_id}
                </b>
              </>
            );
            setBackendValidators({});
            setTestingConnection(false);
            setShowEditIntegrationModal(false);
            refetchIntegrations();
          })
          .catch((editError: { data: IApiError }) => {
            if (editError.data.message.includes("Validation Failed")) {
              renderFlash("error", VALIDATION_FAILED_ERROR);
            }
            if (editError.data.message.includes("Bad request")) {
              renderFlash("error", BAD_REQUEST_ERROR);
            }
            if (editError.data.message.includes("Unknown Error")) {
              renderFlash("error", UNKNOWN_ERROR);
            } else {
              renderFlash(
                "error",
                <>
                  Could not edit{" "}
                  <b>
                    {integrationEditing?.url} -{" "}
                    {integrationEditing?.projectKey ||
                      integrationEditing?.groupId?.toString()}
                  </b>
                  . Please try again.
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
    integration: IIntegrationTableData
  ): void => {
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

  const emptyState = () => {
    const emptyIntegrations: IEmptyTableProps = {
      graphicName: "empty-integrations",
      header: "Set up integrations",
      info:
        "Create tickets automatically when Fleet detects new software vulnerabilities or hosts failing policies.",
      additionalInfo: (
        <>
          Want to learn more?&nbsp;
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/automations"
            text="Read about automations"
            newTab
          />
        </>
      ),
      primaryButton: (
        <Button
          variant="brand"
          className={`${noIntegrationsClass}__add-button`}
          onClick={toggleAddIntegrationModal}
        >
          Add integration
        </Button>
      ),
    };
    return emptyIntegrations;
  };

  const tableHeaders = generateTableHeaders(onActionSelection);

  const tableData = combineJiraAndZendesk();

  return (
    <div className={`${baseClass}`}>
      <h2 className={`${baseClass}__title`}>Ticket destinations</h2>
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
          actionButton={{
            name: "add integration",
            buttonText: "Add integration",
            variant: "brand",
            onActionButtonClick: toggleAddIntegrationModal,
            hideButton: !tableData?.length,
          }}
          resultsTitle={"integrations"}
          emptyComponent={() =>
            EmptyTable({
              graphicName: emptyState().graphicName,
              header: emptyState().header,
              info: emptyState().info,
              additionalInfo: emptyState().additionalInfo,
              primaryButton: emptyState().primaryButton,
            })
          }
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disablePagination
        />
      )}
      {showAddIntegrationModal && (
        <AddIntegrationModal
          onCancel={toggleAddIntegrationModal}
          onSubmit={onAddSubmit}
          backendValidators={backendValidators}
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
      {showEditIntegrationModal && integrations && (
        <EditIntegrationModal
          onCancel={toggleEditIntegrationModal}
          onSubmit={onEditSubmit}
          backendValidators={backendValidators}
          integrations={integrations}
          integrationEditing={integrationEditing}
          testingConnection={testingConnection}
        />
      )}
    </div>
  );
};

export default Integrations;
