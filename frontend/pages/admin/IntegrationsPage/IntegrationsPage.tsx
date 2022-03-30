import React, { useState, useCallback } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";

import { ITeam } from "interfaces/team";
import { IConfig, IConfigNested } from "interfaces/config";
import {
  IIntegrations,
  IJiraIntegration,
  IJiraIntegrationFormData,
  IJiraIntegrationFormErrors,
} from "interfaces/integration";
import { IApiError } from "interfaces/errors";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

import configAPI from "services/entities/config";
import teamsAPI from "services/entities/teams";

import MOCKS from "services/mock_service/mocks/responses";

import TableContainer from "components/TableContainer";
// @ts-ignore
import TableDataError from "components/TableDataError";
import AddIntegrationModal from "./components/CreateIntegrationModal";
import DeleteIntegrationModal from "./components/DeleteIntegrationModal";
import EditIntegrationModal from "./components/EditIntegrationModal";

import { DEFAULT_CREATE_INTEGRATION_ERRORS } from "utilities/constants";

import {
  generateTableHeaders,
  generateDataSet,
} from "./IntegrationsTableConfig";

interface ITeamsResponse {
  teams: ITeam[];
}

const baseClass = "integrations-management";
const noIntegrationsClass = "no-integrations";

const IntegrationsPage = (): JSX.Element => {
  const dispatch = useDispatch();
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
  ] = useState<IJiraIntegration>();
  const [integrationsIndexed, setIntegrationsIndexed] = useState<
    IJiraIntegration[]
  >();
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const [
    createIntegrationError,
    setCreateIntegrationError,
  ] = useState<IJiraIntegrationFormErrors>(DEFAULT_CREATE_INTEGRATION_ERRORS);

  const {
    data: teams,
    isLoading: isLoadingTeams,
    error: loadingTeamsError,
    refetch: refetchTeams,
  } = useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ITeamsResponse) => data.teams,
    }
  );

  const {
    data: integrations,
    isLoading: isLoadingIntegrations,
    error: loadingIntegrationsError,
    refetch: refetchIntegrations,
  } = useQuery<IConfigNested, Error, IJiraIntegration[]>(
    ["integrations"],
    () => configAPI.loadIntegrations(),
    {
      select: (data: IConfigNested) => {
        return data.integrations.jira;
      },
      onSuccess: (data) => {
        const addIndex = data.map((integration, index) => {
          return { ...integration, integrationIndex: index };
        });
        console.log("addIndex", addIndex);
        setIntegrationsIndexed(addIndex);
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
    (integration?: IJiraIntegration) => {
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
    (integration?: IJiraIntegration) => {
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
      console.log("jiraIntegrationSubmitData", jiraIntegrationSubmitData);
      // replace with .update when we have the API
      configAPI
        .updateIntegrations(MOCKS.configAdd2)
        .then(() => {
          dispatch(
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
            )
          );
          setBackendValidators({});
          toggleAddIntegrationModal();
          refetchIntegrations();
        })
        .catch((createError: { data: IApiError }) => {
          if (createError.data.errors[0].reason.includes("Duplicate")) {
            setBackendValidators({
              name: "A team with this name already exists",
            });
          } else {
            dispatch(
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
              )
            );
            toggleAddIntegrationModal();
          }
        });
    },
    [dispatch, toggleAddIntegrationModal]
  );

  const onDeleteSubmit = useCallback(() => {
    if (integrationEditing) {
      const removeIntegration =
        integrationEditing.integrationIndex &&
        integrations?.splice(integrationEditing.integrationIndex, 1);

      // replace with .update(removeIntegration) when we have the APIs
      configAPI
        .updateIntegrations(MOCKS.config1)
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              <>
                Successfully deleted <b>{integrationEditing.url}</b>
              </>
            )
          );
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              <>
                Could not delete <b>{integrationEditing.url}</b>. Please try
                again.
              </>
            )
          );
        })
        .finally(() => {
          refetchIntegrations();
          toggleDeleteIntegrationModal();
        });
    }
  }, [dispatch, integrationEditing, toggleDeleteIntegrationModal]);

  const onEditSubmit = useCallback(
    (formData: IJiraIntegration) => {
      const replaceIntegration =
        integrationEditing?.integrationIndex &&
        integrations?.splice(integrationEditing.integrationIndex, 1, formData);

      // replace with .update(replaceIntegration) when we have the API
      configAPI
        .updateIntegrations(MOCKS.config2)
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              <>
                Successfully edited <b>{formData.url}</b>
              </>
            )
          );
          setBackendValidators({});
          toggleEditIntegrationModal();
          refetchIntegrations();
        })
        .catch((updateError: { data: IApiError }) => {
          console.error(updateError);
          if (updateError.data.errors[0].reason.includes("Duplicate")) {
            setBackendValidators({
              name: "A team with this name already exists",
            });
          } else {
            dispatch(
              renderFlash(
                "error",
                <>
                  Could not edit <b>{integrationEditing?.url}</b>. Please try
                  again.
                </>
              )
            );
          }
        });
    },
    [dispatch, integrationEditing, toggleEditIntegrationModal]
  );

  const onActionSelection = (
    action: string,
    integration: IJiraIntegration
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
          hideActionButton={integrations && integrations.length === 0}
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
          createIntegrationErrors={createIntegrationError}
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
          defaultName={integrationEditing?.url || ""}
          backendValidators={backendValidators}
        />
      )}
    </div>
  );
};

export default IntegrationsPage;
