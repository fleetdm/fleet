import React, { useContext, useState } from "react";
import { useQuery, useQueryClient } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { getPathWithQueryParams } from "utilities/url";
import selfServiceCategoriesAPI, {
  ISelfServiceCategoriesResponse,
} from "services/entities/self_service_categories";
import { ISelfServiceCategory } from "interfaces/self_service_category";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import TeamsDropdown from "components/TeamsDropdown";
import UploadList from "components/UploadList";

import AddCategoryModal from "./AddCategoryModal";
import EditCategoryModal from "./EditCategoryModal";
import DeleteCategoryModal from "./DeleteCategoryModal";

const baseClass = "self-service-categories-page";

const LEARN_MORE_URL =
  "https://fleetdm.com/guides/self-service-software-categories";

interface ISelfServiceCategoriesPageProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    search: string;
    query: { fleet_id?: string; team_id?: string };
    hash?: string;
  };
}

const SelfServiceCategoriesPage = ({
  router,
  location,
}: ISelfServiceCategoriesPageProps) => {
  const {
    config,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);
  const isPrimoMode = config?.partnerships?.enable_primo || false;
  const { renderFlash } = useContext(NotificationContext);
  const queryClient = useQueryClient();

  const {
    currentTeamId,
    teamIdForApi,
    userTeams,
    handleTeamChange,
    isRouteOk,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: true,
  });

  const fleetId = teamIdForApi ?? 0;
  const backToLibraryPath = getPathWithQueryParams(PATHS.SOFTWARE_LIBRARY, {
    fleet_id: teamIdForApi,
  });

  const canManage =
    !!isGlobalAdmin ||
    !!isGlobalMaintainer ||
    !!isTeamAdmin ||
    !!isTeamMaintainer;

  const [showAddModal, setShowAddModal] = useState(false);
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
    () => selfServiceCategoriesAPI.list(fleetId),
    {
      enabled: !!isPremiumTier && isRouteOk,
      refetchOnWindowFocus: false,
    }
  );

  const invalidateList = () => {
    queryClient.invalidateQueries(["selfServiceCategories", fleetId]);
  };

  const onAddSuccess = () => {
    invalidateList();
    setShowAddModal(false);
    renderFlash("success", "Successfully added self-service category.");
  };

  const onEditSuccess = () => {
    invalidateList();
    setCategoryToEdit(null);
    renderFlash("success", "Successfully updated self-service category.");
  };

  const onDeleteSuccess = () => {
    invalidateList();
    setCategoryToDelete(null);
    renderFlash("success", "Successfully deleted self-service category.");
  };

  const renderHeader = () => (
    <>
      <BackButton text="Back to software library" path={backToLibraryPath} />
      {!isPrimoMode && (
        <div className={`${baseClass}__fleet-row`}>
          <TeamsDropdown
            currentUserTeams={userTeams ?? []}
            selectedTeamId={currentTeamId}
            onChange={handleTeamChange}
            includeAllTeams={false}
            includeNoTeams
          />
        </div>
      )}
      <PageDescription
        content={
          <>
            Manage self-service categories.{" "}
            <CustomLink url={LEARN_MORE_URL} text="Learn more" newTab />
          </>
        }
      />
    </>
  );

  const renderBody = () => {
    if (!isPremiumTier) {
      return (
        <div className={`${baseClass}__premium-card`}>
          <PremiumFeatureMessage />
        </div>
      );
    }

    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
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
              ? "Add category to group your software and scripts in self-service."
              : "No self-service categories are available."
          }
          primaryButton={
            canManage ? (
              <Button onClick={() => setShowAddModal(true)}>
                Add category
              </Button>
            ) : undefined
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
          <div className={`${baseClass}__list-header`}>
            <span className={`${baseClass}__list-title`}>
              Self-service categories
            </span>
            {canManage && (
              <Button variant="text-icon" onClick={() => setShowAddModal(true)}>
                <Icon name="plus" />
                Add category
              </Button>
            )}
          </div>
        )}
        ListItemComponent={({ listItem }) => (
          <div className={`${baseClass}__row`}>
            <span className={`${baseClass}__row-name`}>{listItem.name}</span>
            {canManage && (
              <div className={`${baseClass}__row-actions`}>
                <Button
                  variant="icon"
                  onClick={() => setCategoryToEdit(listItem)}
                  ariaLabel={`Edit ${listItem.name}`}
                  title="Edit"
                >
                  <Icon name="pencil" />
                </Button>
                <Button
                  variant="icon"
                  onClick={() => setCategoryToDelete(listItem)}
                  ariaLabel={`Delete ${listItem.name}`}
                  title="Delete"
                >
                  <Icon name="trash" />
                </Button>
              </div>
            )}
          </div>
        )}
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <>
        {renderHeader()}
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
      </>
    </MainContent>
  );
};

export default SelfServiceCategoriesPage;
