import React, { useContext, useCallback, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import DeleteLabelModal from "pages/hosts/ManageHostsPage/components/DeleteLabelModal";

import Button from "components/buttons/Button";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import labelsAPI, { ILabelsResponse } from "services/entities/labels";
import { ILabel } from "interfaces/label";

import LabelsTable from "./LabelsTable";

const baseClass = "manage-labels-page";

interface IManageLabelsPageProps {
  router: InjectedRouter;
  // location: {
  //   pathname: string;
  //   query: {
  //     page?: string;
  //     query?: string;
  //     order_key?: string;
  //     order_direction?: "asc" | "desc";
  //   };
  //   search: string;
  // };
}

const ManageLabelsPage = ({
  router,
}: // location,
IManageLabelsPageProps): JSX.Element => {
  const { currentUser, isGlobalAdmin, isGlobalMaintainer } = useContext(
    AppContext
  );
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

  const onClickAction = (action: string, label: ILabel): void => {
    switch (action) {
      case "view_hosts":
        router.push(`${PATHS.MANAGE_HOSTS}?label_id=${label.id}`);
        break;
      case "edit":
        router.push(`${PATHS.EDIT_LABEL(label.id)}`);
        break;
      case "delete":
        setLabelToDelete(label);
        break;
      default:
    }
  };

  // if (!isOnGlobalTeam(currentUser)) {
  //   // handling like this here since there is existing redirect logic at router level that needs to
  //   // be reconciled
  //   router.push("/404");
  //   return <></>;
  // }

  const canWriteLabels = isGlobalAdmin || isGlobalMaintainer;

  const renderTable = () => {
    if (isLoading || !currentUser || !labels) {
      return <Spinner />;
    }
    if (error) {
      return <DataError />;
    }
    return <LabelsTable labels={labels} onClickAction={onClickAction} />;
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
        {labelToDelete && (
          <DeleteLabelModal
            onSubmit={onConfirmDelete}
            onCancel={() => {
              setLabelToDelete(null);
            }}
            isUpdatingLabel={isUpdating}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManageLabelsPage;
