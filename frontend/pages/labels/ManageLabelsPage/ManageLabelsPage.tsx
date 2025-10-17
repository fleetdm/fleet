import React, { useContext, useCallback, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import labelsAPI, { ILabelsResponse } from "services/entities/labels";

import { ILabel } from "interfaces/label";

import DeleteLabelModal from "pages/hosts/ManageHostsPage/components/DeleteLabelModal";

import Button from "components/buttons/Button";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import PageDescription from "components/PageDescription";

import LabelsTable from "./LabelsTable";

const baseClass = "manage-labels-page";

interface IManageLabelsPageProps {
  router: InjectedRouter;
}

const ManageLabelsPage = ({ router }: IManageLabelsPageProps): JSX.Element => {
  const {
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [labelToDelete, setLabelToDelete] = useState<ILabel | null>(null);
  const [isUpdating, setIsUpdating] = useState(false);

  const { data: labels, isLoading, error, refetch } = useQuery<
    ILabelsResponse,
    Error,
    ILabel[]
  >(["labels"], () => labelsAPI.loadAll(), {
    select: (data: ILabelsResponse) => data.labels,
  });

  const onCreateLabelClick = useCallback(() => {
    router.push(PATHS.NEW_LABEL);
  }, [router]);

  const onConfirmDelete = useCallback(async () => {
    if (labelToDelete) {
      // there will always be one at this point
      try {
        setIsUpdating(true);
        await labelsAPI.destroy(labelToDelete);
        renderFlash("success", `Successfully deleted ${labelToDelete.name}.`);
        refetch();
      } catch {
        renderFlash(
          "error",
          `Could not delete ${labelToDelete.name}. Please try again.`
        );
      } finally {
        setLabelToDelete(null);
        setIsUpdating(false);
      }
    }
  }, [labelToDelete, refetch, renderFlash]);

  const onClickAction = useCallback(
    (action: string, label: ILabel): void => {
      switch (action) {
        case "view_hosts":
          router.push(PATHS.MANAGE_HOSTS_LABEL(label.id));
          break;
        case "edit":
          router.push(PATHS.EDIT_LABEL(label.id));
          break;
        case "delete":
          setLabelToDelete(label);
          break;
        default:
      }
    },
    [router]
  );

  const canAddLabel =
    isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin;

  const renderTable = useCallback(() => {
    if (isLoading || !currentUser || !labels) {
      return <Spinner />;
    }
    if (error) {
      return <DataError />;
    }
    return (
      <LabelsTable
        currentUser={currentUser}
        labels={labels}
        onClickAction={onClickAction}
      />
    );
  }, [currentUser, error, isLoading, labels, onClickAction]);

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__header-wrap`}>
        <div className={`${baseClass}__header`}>
          <div className={`${baseClass}__text`}>
            <div className={`${baseClass}__title`}>
              <h1>Labels</h1>
            </div>
          </div>
          {canAddLabel && (
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
        <PageDescription content="Group hosts for targeting and filtering." />
      </div>
      {renderTable()}
      {labelToDelete && (
        <DeleteLabelModal
          onSubmit={onConfirmDelete}
          onCancel={() => {
            setLabelToDelete(null);
          }}
          isUpdatingLabel={isUpdating}
        />
      )}
    </MainContent>
  );
};

export default ManageLabelsPage;
