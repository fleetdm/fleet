import React, { useState, useCallback, useMemo } from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import TableDataError from "components/DataError";
import EmptyState from "components/EmptyState";
import PageDescription from "components/PageDescription";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import { notify } from "components/ToastNotification";
import { IConfig } from "interfaces/config";
import { IApiError } from "interfaces/errors";
import {
  IJiraIntegration,
  IZendeskIntegration,
  ILinearIntegration,
  IIntegration,
  IIntegrationTableData,
  IGlobalIntegrations,
} from "interfaces/integration";
import SettingsSection from "pages/admin/components/SettingsSection";
import configAPI from "services/entities/config";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import AddTicketDestinationModal from "./components/AddIntegrationModal";
import DeleteIntegrationModal from "./components/DeleteIntegrationModal";
import {
  generateTableHeaders,
  combineDataSets,
} from "./IntegrationsTableConfig";

const baseClass = "integrations-management";

const VALIDATION_FAILED_ERROR =
  "There was a problem with the information you provided.";
const BAD_REQUEST_ERROR =
  "Invalid login credentials or URL. Please correct and try again.";
const LINEAR_BAD_REQUEST_ERROR =
  "Invalid API key or team ID. Please correct and try again.";
const UNKNOWN_ERROR =
  "We experienced an error when attempting to connect. Please try again later.";

const TicketDestinations = (): JSX.Element => {
  const [
    showAddTicketDestinationModal,
    setShowAddTicketDestinationModal,
  ] = useState(false);
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
  const [linearIntegrations, setLinearIntegrations] = useState<
    ILinearIntegration[]
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
          setLinearIntegrations(data.linear);
        }
      },
    }
  );

  // TODO: Cleanup useCallbacks, add missing dependencies, use state setter functions, e.g.,
  // `setShowAddTicketDestinationModal((prevState) => !prevState)`, instead of including state
  // variables as dependencies for toggles, etc.

  const toggleAddTicketDestinationModal = useCallback(() => {
    setShowAddTicketDestinationModal(!showAddTicketDestinationModal);
  }, [showAddTicketDestinationModal, setShowAddTicketDestinationModal]);

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
    (
      submitData: IIntegration[] | ILinearIntegration[],
      integrationDestination: string
    ) => {
      const isLinear = integrationDestination === "linear";
      const integrationSubmitData = submitData as IIntegration[];
      const lastLinear = isLinear
        ? (submitData as ILinearIntegration[])[submitData.length - 1]
        : undefined;
      const lastJiraZendesk =
        integrationSubmitData[integrationSubmitData.length - 1];
      const addedName = lastLinear
        ? `Linear - ${lastLinear.team_id}`
        : `${lastJiraZendesk.url} - ${
            lastJiraZendesk.project_key || lastJiraZendesk.group_id
          }`;

      // Updates integrations.jira, integrations.zendesk or integrations.linear
      const destination = () => {
        if (isLinear) {
          return { linear: submitData as ILinearIntegration[] };
        }
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
      return configAPI
        .update({ integrations: destination() })
        .then(() => {
          notify.success(
            <>
              Successfully added <b>{addedName}</b>
            </>
          );
          toggleAddTicketDestinationModal();
          refetchIntegrations();
        })
        .catch((addError: { data: IApiError }) => {
          if (addError.data?.message.includes("Validation Failed")) {
            if (
              addError.data?.errors[0].reason.includes(
                "duplicate Jira integration"
              ) ||
              addError.data?.errors[0].reason.includes(
                "duplicate Linear integration"
              )
            ) {
              notify.error(
                <>
                  Could not add <b>{addedName}</b>. This integration already
                  exists
                </>,
                { response: addError }
              );
            } else {
              notify.error(VALIDATION_FAILED_ERROR, { response: addError });
            }
          } else if (addError.data?.message.includes("Bad request")) {
            notify.error(
              isLinear ? LINEAR_BAD_REQUEST_ERROR : BAD_REQUEST_ERROR,
              { response: addError }
            );
          } else if (addError.data?.message.includes("Unknown Error")) {
            notify.error(UNKNOWN_ERROR, { response: addError });
          } else {
            notify.error(
              <>
                Could not add <b>{isLinear ? "Linear" : lastJiraZendesk.url}</b>
                . Please try again.
              </>,
              { response: addError }
            );
          }
        })
        .finally(() => {
          setTestingConnection(false);
        });
    },
    [toggleAddTicketDestinationModal]
  );

  const onDeleteSubmit = useCallback(() => {
    if (integrationEditing) {
      const deletedName =
        integrationEditing.type === "linear"
          ? `Linear - ${integrationEditing.teamId}`
          : `${integrationEditing.url} - ${
              integrationEditing.projectKey ||
              integrationEditing.groupId?.toString()
            }`;
      const deleteIntegrationDestination = () => {
        if (integrationEditing.type === "linear") {
          return configAPI.update({
            integrations: {
              linear: (integrations?.linear || []).filter(
                (_, index) => index !== integrationEditing.originalIndex
              ),
            },
          });
        }
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
          notify.success(
            <>
              Successfully deleted <b>{deletedName}</b>
            </>
          );
          refetchIntegrations();
        })
        .catch((deleteError: unknown) => {
          notify.error(
            <>
              Could not delete <b>{deletedName}</b>. Please try again.
            </>,
            { response: deleteError }
          );
        })
        .finally(() => {
          setIsUpdatingIntegration(false);
          toggleDeleteIntegrationModal();
        });
    }
  }, [
    integrationEditing,
    integrations,
    jiraIntegrations,
    zendeskIntegrations,
    toggleDeleteIntegrationModal,
  ]);

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
    () =>
      combineDataSets(
        jiraIntegrations || [],
        zendeskIntegrations || [],
        linearIntegrations || []
      ),
    [jiraIntegrations, zendeskIntegrations, linearIntegrations]
  );

  const renderTable = () => {
    if (loadingIntegrationsError) {
      return <TableDataError />;
    }
    if (isLoadingIntegrations) {
      return <Spinner />;
    }
    return (
      <TableContainer
        columnConfigs={tableHeaders}
        data={tableData}
        defaultSortHeader="name"
        defaultSortDirection="asc"
        isLoading={false}
        actionButton={{
          name: "add integration",
          buttonText: "Add integration",
          variant: "default",
          onClick: toggleAddTicketDestinationModal,
          hideButton: !tableData?.length,
        }}
        resultsTitle="integrations"
        emptyComponent={() => (
          <EmptyState
            header="No ticket destinations"
            info="Create tickets whenever Fleet detects vulnerabilities or failing policies."
            primaryButton={
              <Button onClick={toggleAddTicketDestinationModal}>
                Add ticket destination
              </Button>
            }
          />
        )}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
      />
    );
  };

  return (
    <SettingsSection title="Ticketing" className={baseClass}>
      <PageDescription
        content={
          <>
            Add or edit integrations to create tickets when Fleet detects new
            vulnerabilities.{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/policy-automations`}
              text="Learn more"
              newTab
            />
          </>
        }
        variant="right-panel"
      />
      {renderTable()}
      {showAddTicketDestinationModal && (
        <AddTicketDestinationModal
          onCancel={toggleAddTicketDestinationModal}
          onSubmit={onAddSubmit}
          integrations={integrations || { jira: [], zendesk: [], linear: [] }}
          testingConnection={testingConnection}
        />
      )}
      {showDeleteIntegrationModal && (
        <DeleteIntegrationModal
          onCancel={toggleDeleteIntegrationModal}
          onSubmit={onDeleteSubmit}
          url={
            integrationEditing?.type === "linear"
              ? "Linear"
              : integrationEditing?.url || ""
          }
          projectKey={
            integrationEditing?.projectKey ||
            integrationEditing?.groupId?.toString() ||
            integrationEditing?.teamId ||
            ""
          }
          isUpdatingIntegration={isUpdatingIntegration}
        />
      )}
    </SettingsSection>
  );
};

export default TicketDestinations;
