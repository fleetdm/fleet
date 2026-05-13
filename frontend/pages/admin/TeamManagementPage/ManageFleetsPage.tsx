import React, { useState, useCallback, useContext, useMemo } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";

import { LEARN_MORE_ABOUT_BASE_LINK, PRIMO_TOOLTIP } from "utilities/constants";
import { getGitOpsModeTipContent } from "utilities/helpers";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import { ITeam } from "interfaces/team";
import { IApiError } from "interfaces/errors";
import usersAPI, { IGetMeResponse } from "services/entities/users";
import teamsAPI, {
  ILoadTeamsResponse,
  ITeamFormData,
} from "services/entities/teams";

import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import TableCount from "components/TableContainer/TableCount";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import PageDescription from "components/PageDescription";
import TooltipWrapper from "components/TooltipWrapper";

import CreateFleetModal from "./components/CreateFleetModal";
import DeleteFleetModal from "./components/DeleteFleetModal";
import RenameFleetModal from "./components/RenameFleetModal";

import { generateTableHeaders, generateDataSet } from "./FleetTableConfig";

const baseClass = "manage-fleets";
const noFleetsClass = "no-fleets";

const ManageFleetsPage = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    currentTeam,
    setCurrentTeam,
    setCurrentUser,
    setAvailableTeams,
    setUserSettings,
    config,
  } = useContext(AppContext);

  const [isUpdatingFleets, setIsUpdatingFleets] = useState(false);
  const [showCreateFleetModal, setShowCreateFleetModal] = useState(false);
  const [showDeleteFleetModal, setShowDeleteFleetModal] = useState(false);
  const [showRenameFleetModal, setShowRenameFleetModal] = useState(false);
  const [fleetEditing, setFleetEditing] = useState<ITeam>();
  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});
  const handlePageError = useErrorHandler();

  const { refetch: refetchMe } = useQuery(["me"], () => usersAPI.me(), {
    enabled: false,
    onSuccess: ({ user, available_teams, settings }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
      setUserSettings(settings);
    },
  });

  const {
    data: fleets,
    isFetching: isFetchingFleets,
    error: loadingFleetsError,
    refetch: refetchFleets,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ILoadTeamsResponse) => data.teams,
      onError: (error) => handlePageError(error),
    }
  );

  // TODO: Cleanup useCallbacks, add missing dependencies, use state setter functions, e.g.,
  // `setShowCreateFleetModal((prevState) => !prevState)`, instead of including state
  // variables as dependencies for toggles, etc.

  const toggleCreateFleetModal = useCallback(() => {
    setShowCreateFleetModal(!showCreateFleetModal);
    setBackendValidators({});
  }, [showCreateFleetModal, setShowCreateFleetModal, setBackendValidators]);

  const toggleDeleteFleetModal = useCallback(
    (fleet?: ITeam) => {
      setShowDeleteFleetModal(!showDeleteFleetModal);
      fleet ? setFleetEditing(fleet) : setFleetEditing(undefined);
    },
    [showDeleteFleetModal, setShowDeleteFleetModal, setFleetEditing]
  );

  const toggleRenameFleetModal = useCallback(
    (fleet?: ITeam) => {
      setShowRenameFleetModal(!showRenameFleetModal);
      setBackendValidators({});
      fleet ? setFleetEditing(fleet) : setFleetEditing(undefined);
    },
    [
      showRenameFleetModal,
      setShowRenameFleetModal,
      setFleetEditing,
      setBackendValidators,
    ]
  );

  const onCreateSubmit = useCallback(
    (formData: ITeamFormData) => {
      setIsUpdatingFleets(true);
      teamsAPI
        .create(formData)
        .then(() => {
          renderFlash("success", `Successfully created ${formData.name}.`);
          setBackendValidators({});
          toggleCreateFleetModal();
          refetchMe();
          refetchFleets();
        })
        .catch((createError: { data: IApiError }) => {
          const rawReason = createError.data.errors[0].reason;
          const errMsg = rawReason.toLowerCase();
          if (errMsg.includes("must differ")) {
            setBackendValidators({ name: rawReason });
          } else if (errMsg.includes("duplicate")) {
            setBackendValidators({
              name: "A fleet with this name already exists",
            });
          } else if (
            errMsg.includes("all teams") ||
            errMsg.includes("all fleets") ||
            errMsg.includes("no team") ||
            errMsg.includes("unassigned")
          ) {
            setBackendValidators({
              name: `"${formData.name}" is a reserved fleet name. Please try another name.`,
            });
          } else {
            renderFlash("error", "Could not create fleet. Please try again.");
            toggleCreateFleetModal();
          }
        })
        .finally(() => {
          setIsUpdatingFleets(false);
        });
    },
    [toggleCreateFleetModal, refetchMe, refetchFleets, renderFlash]
  );

  const onDeleteSubmit = useCallback(() => {
    if (fleetEditing) {
      setIsUpdatingFleets(true);
      teamsAPI
        .destroy(fleetEditing.id)
        .then(() => {
          renderFlash("success", `Successfully deleted ${fleetEditing.name}.`);
          if (currentTeam?.id === fleetEditing.id) {
            setCurrentTeam(undefined);
          }
        })
        .catch(() => {
          renderFlash(
            "error",
            `Could not delete ${fleetEditing.name}. Please try again.`
          );
        })
        .finally(() => {
          setIsUpdatingFleets(false);
          refetchMe();
          refetchFleets();
          toggleDeleteFleetModal();
        });
    }
  }, [
    currentTeam,
    fleetEditing,
    refetchMe,
    refetchFleets,
    renderFlash,
    setCurrentTeam,
    toggleDeleteFleetModal,
  ]);

  const onRenameSubmit = useCallback(
    (formData: ITeamFormData) => {
      if (formData.name === fleetEditing?.name) {
        toggleRenameFleetModal();
      } else if (fleetEditing) {
        setIsUpdatingFleets(true);
        teamsAPI
          .update(formData, fleetEditing.id)
          .then(() => {
            renderFlash(
              "success",
              `Successfully updated fleet name to ${formData.name}.`
            );
            setBackendValidators({});
            toggleRenameFleetModal();
            refetchFleets();
          })
          .catch((updateError: { data: IApiError }) => {
            console.error(updateError);
            const rawReason = updateError.data.errors[0].reason;
            const errMsg = rawReason.toLowerCase();
            if (errMsg.includes("must differ")) {
              setBackendValidators({ name: rawReason });
            } else if (errMsg.includes("duplicate")) {
              setBackendValidators({
                name: "A fleet with this name already exists",
              });
            } else if (
              errMsg.includes("all teams") ||
              errMsg.includes("all fleets")
            ) {
              setBackendValidators({
                name: `"All fleets" is a reserved fleet name.`,
              });
            } else if (
              errMsg.includes("no team") ||
              errMsg.includes("unassigned")
            ) {
              setBackendValidators({
                name: `"Unassigned" is a reserved fleet name. Please try another name.`,
              });
            } else {
              renderFlash(
                "error",
                `Could not rename ${fleetEditing.name}. Please try again.`
              );
            }
          })
          .finally(() => {
            setIsUpdatingFleets(false);
          });
      }
    },
    [fleetEditing, toggleRenameFleetModal, refetchFleets, renderFlash]
  );

  const onActionSelection = useCallback(
    (action: string, fleet: ITeam): void => {
      switch (action) {
        case "rename":
          toggleRenameFleetModal(fleet);
          break;
        case "delete":
          toggleDeleteFleetModal(fleet);
          break;
        default:
      }
    },
    [toggleRenameFleetModal, toggleDeleteFleetModal]
  );

  const tableHeaders = useMemo(() => generateTableHeaders(onActionSelection), [
    onActionSelection,
  ]);
  const tableData = useMemo(() => (fleets ? generateDataSet(fleets) : []), [
    fleets,
  ]);

  const renderFleetCount = useCallback(() => {
    return <TableCount name="fleets" count={fleets?.length} />;
  }, [fleets]);

  const disabledPrimaryActionTooltip = (() => {
    if (config?.partnerships?.enable_primo) {
      return PRIMO_TOOLTIP;
    }
    if (config?.gitops?.gitops_mode_enabled && config?.gitops?.repository_url) {
      return getGitOpsModeTipContent(config.gitops.repository_url);
    }
    return null;
  })();

  return (
    <div className={`${baseClass}`}>
      <PageDescription
        content={
          <>
            Use fleets to group hosts together with their own controls, reports,
            and policies.{" "}
            <CustomLink
              text="Learn more"
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/fleets`}
              newTab
            />
          </>
        }
      />
      {loadingFleetsError ? (
        <TableDataError />
      ) : (
        <TableContainer
          columnConfigs={tableHeaders}
          data={tableData}
          isLoading={isFetchingFleets}
          defaultSortHeader="name"
          defaultSortDirection="asc"
          actionButton={{
            name: "create fleet",
            buttonText: "Create fleet",
            variant: "default",
            onClick: toggleCreateFleetModal,
            hideButton: false,
            disabledTooltipContent: disabledPrimaryActionTooltip,
          }}
          resultsTitle="fleets"
          emptyComponent={() => {
            const rawButton = (
              <Button
                disabled={!!disabledPrimaryActionTooltip}
                onClick={toggleCreateFleetModal}
                className={`${noFleetsClass}__create-button`}
              >
                Create fleet
              </Button>
            );
            const primaryButton = disabledPrimaryActionTooltip ? (
              <TooltipWrapper
                tipContent={disabledPrimaryActionTooltip}
                position="top"
                underline={false}
                showArrow
                tipOffset={8}
              >
                {rawButton}
              </TooltipWrapper>
            ) : (
              rawButton
            );
            return (
              <EmptyState
                header="No fleets yet"
                info="Create a fleet to add hosts and assign users."
                primaryButton={primaryButton}
              />
            );
          }}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          isClientSidePagination
          renderCount={renderFleetCount}
        />
      )}
      {showCreateFleetModal && (
        <CreateFleetModal
          onCancel={toggleCreateFleetModal}
          onSubmit={onCreateSubmit}
          backendValidators={backendValidators}
          isUpdatingFleets={isUpdatingFleets}
        />
      )}
      {showDeleteFleetModal && (
        <DeleteFleetModal
          onCancel={toggleDeleteFleetModal}
          onSubmit={onDeleteSubmit}
          name={fleetEditing?.name || ""}
          isUpdatingFleets={isUpdatingFleets}
        />
      )}
      {showRenameFleetModal && (
        <RenameFleetModal
          onCancel={toggleRenameFleetModal}
          onSubmit={onRenameSubmit}
          defaultName={fleetEditing?.name || ""}
          backendValidators={backendValidators}
          isUpdatingFleets={isUpdatingFleets}
        />
      )}
    </div>
  );
};

export default ManageFleetsPage;
