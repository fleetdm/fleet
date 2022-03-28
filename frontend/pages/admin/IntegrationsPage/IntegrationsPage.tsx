import React, { useState, useCallback } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";

import { ITeam } from "interfaces/team";
import { IApiError } from "interfaces/errors";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import Button from "components/buttons/Button";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

import teamsAPI from "services/entities/teams";

import TableContainer from "components/TableContainer";
// @ts-ignore
import TableDataError from "components/TableDataError";
import AddIntegrationModal from "./components/CreateIntegrationModal";
import DeleteIntegrationModal from "./components/DeleteIntegrationModal";
import EditIntegrationModal from "./components/EditIntegrationModal";
import { ICreateIntegrationFormData } from "./components/CreateIntegrationModal/CreateIntegrationModal";
import { IEditTeamFormData } from "./components/EditIntegrationModal/EditIntegrationModal";
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
  const [integrationEditing, setIntegrationEditing] = useState<ITeam>(); // TODO: Change to IIntegration
  const [searchString, setSearchString] = useState<string>("");
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});

  // TODO: Change to integration useQuery
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

  const toggleAddIntegrationModal = useCallback(() => {
    setShowAddIntegrationModal(!showAddIntegrationModal);
    setBackendValidators({});
  }, [
    showAddIntegrationModal,
    setShowAddIntegrationModal,
    setBackendValidators,
  ]);

  const toggleDeleteIntegrationModal = useCallback(
    (team?: ITeam) => {
      setShowDeleteIntegrationModal(!showDeleteIntegrationModal);
      team ? setIntegrationEditing(team) : setIntegrationEditing(undefined);
    },
    [
      showDeleteIntegrationModal,
      setShowDeleteIntegrationModal,
      setIntegrationEditing,
    ]
  );

  const toggleEditIntegrationModal = useCallback(
    (team?: ITeam) => {
      setShowEditIntegrationModal(!showEditIntegrationModal);
      setBackendValidators({});
      team ? setIntegrationEditing(team) : setIntegrationEditing(undefined);
    },
    [
      showEditIntegrationModal,
      setShowEditIntegrationModal,
      setIntegrationEditing,
      setBackendValidators,
    ]
  );

  const onQueryChange = useCallback(
    (queryData) => {
      if (teams) {
        setSearchString(queryData.searchQuery);
        const { pageIndex, pageSize, searchQuery } = queryData;
        teamsAPI.loadAll({
          page: pageIndex,
          perPage: pageSize,
          globalFilter: searchQuery,
        });
      }
    },
    [dispatch, setSearchString]
  );

  const onCreateSubmit = useCallback(
    (formData: ICreateIntegrationFormData) => {
      teamsAPI
        .create(formData)
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully added ${formData.name}.`) // TODO: formData.URL
          );
          setBackendValidators({});
          toggleAddIntegrationModal();
          refetchTeams();
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
                `Could not add ${formData.name}. Please try again.` // TODO: formData.URL
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
      teamsAPI
        .destroy(integrationEditing.id)
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              `Successfully deleted ${integrationEditing.name}.`
            )
          );
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not delete ${integrationEditing.name}. Please try again.`
            )
          );
        })
        .finally(() => {
          refetchTeams();
          toggleDeleteIntegrationModal();
        });
    }
  }, [dispatch, integrationEditing, toggleDeleteIntegrationModal]);

  const onEditSubmit = useCallback(
    (formData: IEditTeamFormData) => {
      if (formData.name === integrationEditing?.name) {
        toggleEditIntegrationModal();
      } else if (integrationEditing) {
        teamsAPI
          .update(integrationEditing.id, formData)
          .then(() => {
            dispatch(
              renderFlash(
                "success",
                `Successfully edited ${formData.name}.` // TODO: formData.URL
              )
            );
            setBackendValidators({});
            toggleEditIntegrationModal();
            refetchTeams();
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
                  `Could not edit ${integrationEditing.name}. Please try again.` // TODO: integrationEditing.URL
                )
              );
            }
          });
      }
    },
    [dispatch, integrationEditing, toggleEditIntegrationModal]
  );

  const onActionSelection = (action: string, team: ITeam): void => {
    switch (action) {
      case "edit":
        toggleEditIntegrationModal(team);
        break;
      case "delete":
        toggleDeleteIntegrationModal(team);
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
  const tableData = teams ? generateDataSet(teams) : [];

  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Add or edit integrations to create tickets when Fleet detects new
        vulnerabilities.
      </p>
      {loadingTeamsError ? (
        <TableDataError />
      ) : (
        <TableContainer
          columns={tableHeaders}
          data={tableData}
          isLoading={isLoadingTeams}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search"}
          actionButtonText={"Add integration"}
          actionButtonVariant={"brand"}
          hideActionButton={teams && teams.length === 0 && searchString === ""}
          onActionButtonClick={toggleAddIntegrationModal}
          onQueryChange={onQueryChange}
          resultsTitle={"teams"}
          emptyComponent={NoIntegrationsComponent}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable={teams && teams.length > 0 && searchString !== ""}
          disablePagination
        />
      )}
      {showAddIntegrationModal ? (
        <AddIntegrationModal
          onCancel={toggleAddIntegrationModal}
          onSubmit={onCreateSubmit}
          backendValidators={backendValidators}
        />
      ) : null}
      {showDeleteIntegrationModal ? (
        <DeleteIntegrationModal
          onCancel={toggleDeleteIntegrationModal}
          onSubmit={onDeleteSubmit}
          name={integrationEditing?.name || ""}
        />
      ) : null}
      {showEditIntegrationModal ? (
        <EditIntegrationModal
          onCancel={toggleEditIntegrationModal}
          onSubmit={onEditSubmit}
          defaultName={integrationEditing?.name || ""}
          backendValidators={backendValidators}
        />
      ) : null}
    </div>
  );
};

export default IntegrationsPage;
