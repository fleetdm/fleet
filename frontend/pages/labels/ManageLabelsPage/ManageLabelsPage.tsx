import React, { useContext, useCallback, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import Button from "components/buttons/Button";
import MainContent from "components/MainContent";
import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import TableCount from "components/TableContainer/TableCount";
import PATHS from "router/paths";
import {
  isGlobalAdmin,
  isGlobalMaintainer,
  isOnGlobalTeam,
} from "utilities/permissions/permissions";
import labelsAPI, { ILabelsResponse } from "services/entities/labels";
import { ILabel } from "interfaces/label";

import { generateTableHeaders, generateDataSet } from "./LabelsTableConfig";
import "./ManageLabelsPage.scss";

const baseClass = "manage-labels-page";

interface IManageLabelsPageProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: {
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
    };
    search: string;
  };
}

const ManageLabelsPage = ({
  router,
  location,
}: IManageLabelsPageProps): JSX.Element => {
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const handlePageError = useErrorHandler();
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [labelToDelete, setLabelToDelete] = useState<ILabel | undefined>();

  const onCreateLabelClick = useCallback(() => {
    router.push(PATHS.NEW_LABEL);
  }, [router]);

  const {
    data: labels,
    isFetching: isFetchingLabels,
    error: loadingLabelsError,
    refetch: refetchLabels,
  } = useQuery<ILabelsResponse, Error, ILabel[]>(
    ["labels"],
    () => labelsAPI.loadAll(),
    {
      select: (data: ILabelsResponse) => data.labels,
      onError: (error) => handlePageError(error),
    }
  );

  const onActionSelection = useCallback(
    (action: string, label: ILabel): void => {
      switch (action) {
        case "view_hosts":
          // Navigate to hosts page filtered by this label
          router.push(`${PATHS.MANAGE_HOSTS}?label_id=${label.id}`);
          break;
        case "edit":
          // Navigate to edit label page
          router.push(`${PATHS.EDIT_LABEL(label.id)}`);
          break;
        case "delete":
          // Open delete confirmation modal
          setLabelToDelete(label);
          setShowDeleteModal(true);
          break;
        default:
      }
    },
    [router]
  );

  const toggleDeleteModal = useCallback(() => {
    setShowDeleteModal(!showDeleteModal);
    if (showDeleteModal) {
      setLabelToDelete(undefined);
    }
  }, [showDeleteModal]);

  const onDeleteSubmit = useCallback(() => {
    if (labelToDelete) {
      labelsAPI
        .destroy(labelToDelete)
        .then(() => {
          renderFlash("success", `Successfully deleted ${labelToDelete.name}.`);
          refetchLabels();
        })
        .catch(() => {
          renderFlash(
            "error",
            `Could not delete ${labelToDelete.name}. Please try again.`
          );
        })
        .finally(() => {
          toggleDeleteModal();
        });
    }
  }, [labelToDelete, refetchLabels, renderFlash, toggleDeleteModal]);

  if (!currentUser) {
    return <></>;
  }
  if (!isOnGlobalTeam(currentUser)) {
    // handling like this here since there is existing redirect logic at router level that needs to
    // be reconciled
    router.push("/404");
    return <></>;
  }

  const canWriteLabels =
    isGlobalAdmin(currentUser) || isGlobalMaintainer(currentUser);

  const tableHeaders = generateTableHeaders(onActionSelection, currentUser);
  const tableData = labels ? generateDataSet(labels, currentUser) : [];

  const renderLabelCount = () => {
    if (!labels || labels.length === 0) {
      return <></>;
    }

    return <TableCount name="labels" count={labels.length} />;
  };

  // Placeholder for delete modal - in a real implementation, you would create a proper DeleteLabelModal component
  const renderDeleteModal = () => {
    if (!showDeleteModal || !labelToDelete) {
      return null;
    }

    return (
      <div className="modal">
        <div className="modal-content">
          <h2>Delete Label</h2>
          <p>
            Are you sure you want to delete the label &quot;{labelToDelete.name}
            &quot;?
          </p>
          <div className="modal-actions">
            <Button onClick={toggleDeleteModal} variant="inverse">
              Cancel
            </Button>
            <Button onClick={onDeleteSubmit} variant="alert">
              Delete
            </Button>
          </div>
        </div>
      </div>
    );
  };

  const renderTable = () => {
    if (loadingLabelsError) {
      return <TableDataError />;
    }

    return (
      <TableContainer
        columnConfigs={tableHeaders}
        data={tableData}
        isLoading={isFetchingLabels}
        defaultSortHeader="name"
        defaultSortDirection="asc"
        resultsTitle="labels"
        showMarkAllPages={false}
        isAllPagesSelected={false}
        isClientSidePagination
        renderCount={renderLabelCount}
        emptyComponent={() => (
          <div className={`${baseClass}__empty-state`}>
            <p>No labels found.</p>
          </div>
        )}
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                <h1>Labels</h1>
              </div>
            </div>
          </div>
          {canWriteLabels && (
            <div className={`${baseClass}__action-button-container`}>
              <Button
                className={`${baseClass}__create-button`}
                onClick={onCreateLabelClick}
              >
                Add label
              </Button>
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          <p>Group hosts for targeting and filtering</p>
        </div>
        {renderTable()}
        {renderDeleteModal()}
      </div>
    </MainContent>
  );
};

export default ManageLabelsPage;
