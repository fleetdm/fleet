import React, { useContext, useEffect, useState } from "react";
import { useQuery, useQueryClient } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import useTeamIdParam from "hooks/useTeamIdParam";
import { getPathWithQueryParams } from "utilities/url";
import selfServiceCategoriesAPI, {
  ISelfServiceCategoriesResponse,
} from "services/entities/self_service_categories";
import { ISelfServiceCategory } from "interfaces/self_service_category";

import { notify } from "components/ToastNotification";
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
import TooltipTruncatedText from "components/TooltipTruncatedText";
import UploadList from "components/UploadList";

import AddCategoryModal from "./AddCategoryModal";
import EditCategoryModal from "./EditCategoryModal";
import DeleteCategoryModal from "./DeleteCategoryModal";

const baseClass = "self-service-categories-page";

interface ISelfServiceCategoriesPageProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    search: string;
    query: { fleet_id?: string; team_id?: string; add_category?: string };
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
  } = useContext(AppContext);
  const isPrimoMode = config?.partnerships?.enable_primo || false;
  const queryClient = useQueryClient();

  const {
    currentTeamId,
    teamIdForApi,
    userTeams,
    handleTeamChange,
    isRouteOk,
    isTeamAdmin,
    isTeamMaintainer,
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

  // Open the Add category modal via deep-link (e.g. from the command
  // palette). Gate on the same predicate the in-page button uses so the
  // param can't bypass admin/maintainer-only authoring. Strip the param
  // either way so refreshes don't keep reopening the modal.
  useEffect(() => {
    if (location.query.add_category !== "1") return;
    if (canManage) {
      setShowAddModal(true);
    }
    const { add_category, ...rest } = location.query;
    router.replace({ pathname: location.pathname, query: rest });
  }, [location.query, location.pathname, router, canManage]);
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
    ["selfServiceCategories", teamIdForApi],
    () => selfServiceCategoriesAPI.getCategories(teamIdForApi as number),
    {
      enabled: !!isPremiumTier && isRouteOk && teamIdForApi !== undefined,
      refetchOnWindowFocus: false,
    }
  );

  const invalidateList = () => {
    queryClient.invalidateQueries(["selfServiceCategories", teamIdForApi]);
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

  const renderHeader = () => (
    <>
      <BackButton text="Back to software library" path={backToLibraryPath} />
      {isPremiumTier && !isPrimoMode ? (
        <div className={`${baseClass}__fleet-row`}>
          <TeamsDropdown
            currentUserTeams={userTeams ?? []}
            selectedTeamId={currentTeamId}
            onChange={handleTeamChange}
            includeAllTeams={false}
            includeNoTeams
          />
        </div>
      ) : (
        <h1>Self-service categories</h1>
      )}
      <PageDescription
        content={
          <>
            Manage self-service categories.{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/self-service-software-categories"
              text="Learn more"
              newTab
            />
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
              <Button variant="inverse" onClick={() => setShowAddModal(true)}>
                <Icon name="plus" />
                Add category
              </Button>
            )}
          </div>
        )}
        ListItemComponent={({ listItem }) => (
          <div className={`${baseClass}__row`}>
            <div className={`${baseClass}__row-name`}>
              <TooltipTruncatedText value={listItem.name} />
            </div>
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
