import React, { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import UploadList from "components/UploadList";
import { ISelfServiceCategory } from "interfaces/self_service_category";
import selfServiceCategoriesAPI, {
  ISelfServiceCategoriesResponse,
} from "services/entities/self_service_categories";

import { ILibrarySettingsCardProps } from "../../types";

import AddCategoryModal from "./AddCategoryModal";
import DeleteCategoryModal from "./DeleteCategoryModal";
import EditCategoryModal from "./EditCategoryModal";

const baseClass = "self-service-categories-section";

const SelfServiceCategoriesSection = ({
  router,
  location,
  fleetId,
  isRouteOk,
  canManage,
  gitOpsModeEnabled,
}: ILibrarySettingsCardProps) => {
  const queryClient = useQueryClient();

  const [showAddModal, setShowAddModal] = useState(false);

  // Open the Add category modal via deep-link (e.g. from the command
  // palette). Gate on the same predicates the in-page button uses so the
  // param can't bypass admin/maintainer-only authoring or GitOps mode.
  // Strip the param either way so refreshes don't keep reopening the modal.
  useEffect(() => {
    if (location.query.add_category !== "1") return;
    if (canManage && !gitOpsModeEnabled) {
      setShowAddModal(true);
    }
    const { add_category, ...rest } = location.query;
    router.replace({ pathname: location.pathname, query: rest });
  }, [location.query, location.pathname, router, canManage, gitOpsModeEnabled]);

  const [
    categoryToEdit,
    setCategoryToEdit,
  ] = useState<ISelfServiceCategory | null>(null);
  const [
    categoryToDelete,
    setCategoryToDelete,
  ] = useState<ISelfServiceCategory | null>(null);

  const { data: categoriesData, isLoading, isError } = useQuery<
    ISelfServiceCategoriesResponse,
    Error
  >(
    ["selfServiceCategories", fleetId],
    () => selfServiceCategoriesAPI.getCategories(fleetId),
    {
      enabled: isRouteOk,
      refetchOnWindowFocus: false,
    }
  );

  const invalidateList = () => {
    queryClient.invalidateQueries(["selfServiceCategories", fleetId]);
  };

  const onAddSuccess = () => {
    invalidateList();
    setShowAddModal(false);
    notify.success("Successfully added self-service category.");
  };

  const onEditSuccess = () => {
    invalidateList();
    setCategoryToEdit(null);
    notify.success("Successfully updated self-service category.");
  };

  const onDeleteSuccess = () => {
    invalidateList();
    setCategoryToDelete(null);
    notify.success("Successfully deleted self-service category.");
  };

  const renderBody = () => {
    if (!isRouteOk || isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError verticalPaddingSize="pad-xxxlarge" selfCenter />;
    }

    const categories = categoriesData?.self_service_categories ?? [];
    const hasCategories = categories.length > 0;

    if (!hasCategories) {
      return (
        <EmptyState
          variant="header-list"
          header="No self-service categories"
          info={
            canManage
              ? "Add category to group your software and scripts in self service."
              : "No self-service categories are available."
          }
        />
      );
    }

    return (
      <UploadList<ISelfServiceCategory>
        className={`${baseClass}__list`}
        keyAttribute="id"
        listItems={categories}
        HeadingComponent={() => (
          <span className={`${baseClass}__list-title`}>
            Self-service categories
          </span>
        )}
        ListItemComponent={({ listItem }) => (
          <div className={`${baseClass}__row`}>
            <div className={`${baseClass}__row-name`}>
              <TooltipTruncatedText value={listItem.name} />
            </div>
            {canManage && (
              <div className={`${baseClass}__row-actions`}>
                <GitOpsModeTooltipWrapper
                  entityType="software"
                  position="top"
                  tipOffset={8}
                  renderChildren={(disableChildren) => (
                    <Button
                      variant="secondary"
                      onClick={() => setCategoryToEdit(listItem)}
                      ariaLabel={`Edit ${listItem.name}`}
                      title="Edit"
                      icon="pencil"
                      disabled={disableChildren}
                    />
                  )}
                />
                <GitOpsModeTooltipWrapper
                  entityType="software"
                  position="top"
                  tipOffset={8}
                  renderChildren={(disableChildren) => (
                    <Button
                      variant="secondary"
                      onClick={() => setCategoryToDelete(listItem)}
                      ariaLabel={`Delete ${listItem.name}`}
                      title="Delete"
                      icon="trash"
                      disabled={disableChildren}
                    />
                  )}
                />
              </div>
            )}
          </div>
        )}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        wrapperCustomClass={`${baseClass}__header`}
        title="Self-service categories"
        details={
          canManage ? (
            <GitOpsModeTooltipWrapper
              entityType="software"
              position="top"
              tipOffset={8}
              renderChildren={(disableChildren) => (
                <Button
                  variant="secondary"
                  size="small"
                  onClick={() => setShowAddModal(true)}
                  icon="plus"
                  disabled={disableChildren}
                >
                  Add category
                </Button>
              )}
            />
          ) : undefined
        }
      />
      {renderBody()}

      {showAddModal && (
        <AddCategoryModal
          fleetId={fleetId}
          onExit={() => setShowAddModal(false)}
          onSuccess={onAddSuccess}
        />
      )}

      {categoryToEdit && (
        <EditCategoryModal
          category={categoryToEdit}
          onExit={() => setCategoryToEdit(null)}
          onSuccess={onEditSuccess}
        />
      )}

      {categoryToDelete && (
        <DeleteCategoryModal
          category={categoryToDelete}
          onExit={() => setCategoryToDelete(null)}
          onSuccess={onDeleteSuccess}
        />
      )}
    </div>
  );
};

export default SelfServiceCategoriesSection;
