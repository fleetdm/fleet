import React, { useCallback, useEffect, useMemo } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { IDeviceSoftwareWithUiStatus } from "interfaces/software";
import { ISelfServiceCategory } from "interfaces/self_service_category";
import selfServiceCategoriesAPI, {
  ISelfServiceCategoriesResponse,
} from "services/entities/self_service_categories";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getPathWithQueryParams } from "utilities/url";

import Card from "components/Card";
import EmptyState from "components/EmptyState";
import Spinner from "components/Spinner";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import InstallAllInCategoryButton from "../components/InstallAllInCategoryButton";
import SelfServiceFilters from "../components/SelfServiceFilters";
import SelfServiceHeader from "../components/SelfServiceHeader";
import SelfServiceTable from "../components/SelfServiceTable";
import SelfServiceTiles from "../components/SelfServiceTiles";
import {
  countUninstalledForInstallAll,
  filterSoftwareByCustomCategory,
  hasInProgressInstallAllItems,
} from "../helpers";

const baseClass = "software-self-service";

export interface SelfServiceQueryParams {
  query: string;
  category_id?: number;
  order_key: string;
  order_direction: "asc" | "desc";
  page: number;
  per_page: number;
}

export interface ISelfServiceCardProps {
  contactUrl: string;
  deviceToken: string;
  queryParams: SelfServiceQueryParams;
  enhancedSoftware: IDeviceSoftwareWithUiStatus[];
  selfServiceData?: IGetDeviceSoftwareResponse;
  tableConfig?: any;
  isLoading: boolean;
  isError: boolean;
  isFetching: boolean;
  isEmpty: boolean;
  router: InjectedRouter;
  pathname: string;
  isMobileView?: boolean;
  onClickInstallAction: (softwareId: number, isScriptPackage?: boolean) => void;
  onInstallAllSuccess?: () => void;
}

const SelfServiceCard = ({
  contactUrl,
  deviceToken,
  queryParams,
  enhancedSoftware,
  selfServiceData,
  tableConfig,
  isLoading,
  isError,
  isFetching,
  isEmpty,
  router,
  pathname,
  isMobileView,
  onClickInstallAction,
  onInstallAllSuccess,
}: ISelfServiceCardProps) => {
  const initialSortHeader = queryParams.order_key || "name";
  const initialSortDirection = queryParams.order_direction || "asc";

  // Device-token-scoped categories: the BE derives the fleet from the token
  // so the dropdown reflects this host's fleet, not the global fleet_id=0 set.
  // The queryKey's second element must match the queryFn arg to avoid
  // cross-device cache bleed.
  const { data: categoriesData, isSuccess: isCategoriesSuccess } = useQuery<
    ISelfServiceCategoriesResponse,
    Error,
    ISelfServiceCategory[]
  >(
    ["device_self_service_categories", deviceToken],
    () => selfServiceCategoriesAPI.getDeviceCategories(deviceToken),
    {
      select: (response) => response.self_service_categories,
      staleTime: 60_000,
    }
  );

  const categories = useMemo(() => categoriesData ?? [], [categoriesData]);

  const softwareInSelectedCategory = useMemo(
    () =>
      filterSoftwareByCustomCategory(
        enhancedSoftware,
        categories,
        queryParams.category_id
      ),
    [enhancedSoftware, categories, queryParams.category_id]
  );

  const uninstalledCount = useMemo(
    () => countUninstalledForInstallAll(softwareInSelectedCategory),
    [softwareInSelectedCategory]
  );

  const hasInProgress = useMemo(
    () => hasInProgressInstallAllItems(softwareInSelectedCategory),
    [softwareInSelectedCategory]
  );

  const onClientSidePaginationChange = useCallback(
    (page: number) => {
      router.push(
        getPathWithQueryParams(pathname, {
          query: queryParams.query,
          category_id: queryParams.category_id,
          order_key: initialSortHeader,
          order_direction: initialSortDirection,
          page,
        })
      );
    },
    [
      pathname,
      queryParams.query,
      queryParams.category_id,
      initialSortDirection,
      initialSortHeader,
      router,
    ]
  );

  const onSearchQueryChange = (value: string) => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: value,
        category_id: queryParams.category_id,
        order_key: initialSortHeader,
        order_direction: initialSortDirection,
        page: 0,
      })
    );
  };

  const onSortChange = ({ sortHeader, sortDirection }: ITableQueryData) => {
    router.push(
      getPathWithQueryParams(pathname, {
        ...queryParams,
        order_key: sortHeader,
        order_direction: sortDirection,
        query: queryParams.query !== undefined ? queryParams.query : undefined,
        category_id:
          queryParams.category_id !== undefined
            ? queryParams.category_id
            : undefined,
        page: 0,
      })
    );
  };

  const onCategoryChange = useCallback(
    (categoryId: number | undefined) => {
      router.push(
        getPathWithQueryParams(pathname, {
          category_id: categoryId,
          query: queryParams.query,
          order_key: initialSortHeader,
          order_direction: initialSortDirection,
          page: 0,
        })
      );
    },
    [
      pathname,
      queryParams.query,
      initialSortHeader,
      initialSortDirection,
      router,
    ]
  );

  // Recover from stale links: if the URL has a category_id that doesn't match
  // any loaded category (admin deleted it, or the list resolved empty), the
  // trigger label would fall through to "All" while filterSoftwareByCustomCategory
  // returns [] — contradicting what the label promises. Drop the param so the
  // user lands back on a real "All" view.
  useEffect(() => {
    if (!isCategoriesSuccess || queryParams.category_id === undefined) return;
    const idIsKnown = categories.some((c) => c.id === queryParams.category_id);
    if (!idIsKnown) {
      onCategoryChange(undefined);
    }
  }, [
    isCategoriesSuccess,
    categories,
    queryParams.category_id,
    onCategoryChange,
  ]);

  if (isLoading)
    return <Spinner {...(isMobileView && { variant: "mobile" })} />;
  if (isError)
    return (
      <EmptyState
        header="Error loading software"
        {...(isMobileView && { variant: "list" })}
      />
    );

  if ((isEmpty || !selfServiceData) && !isFetching) {
    return (
      <EmptyState
        header="No self-service software available yet"
        info="Your organization didn’t add any self-service software."
        {...(isMobileView && { variant: "list" })}
      />
    );
  }

  // Search query filter required for mobile view only ( desktop view has filter built into TableContainer)
  const filteredSoftware = isMobileView
    ? softwareInSelectedCategory.filter((software) => {
        const query = queryParams.query?.toLowerCase().trim() ?? "";
        if (!query) return true;
        return software.name.toLowerCase().includes(query);
      })
    : softwareInSelectedCategory;

  // The button shows in all four variants (including "All"). On "All",
  // `categoryId` is undefined; the click posts to install_all without a
  // category_id query param and the BE installs every eligible (uninstalled,
  // not-in-progress) self-service item. Disabled state is driven purely by
  // hasInProgressInCategory || uninstalledCount === 0 — no special case for
  // categoryId === undefined.
  const installAllButton = !isMobileView ? (
    <InstallAllInCategoryButton
      uninstalledCount={uninstalledCount}
      hasInProgressInCategory={hasInProgress}
      deviceToken={deviceToken}
      categoryId={queryParams.category_id}
      onSuccess={() => onInstallAllSuccess?.()}
    />
  ) : null;

  if (isMobileView) {
    return (
      <>
        <SelfServiceHeader contactUrl={contactUrl} variant="mobile-header" />
        <div className={`${baseClass}__mobile-installers`}>
          <SelfServiceFilters
            query={queryParams.query}
            categoryId={queryParams.category_id}
            categories={categories}
            onSearchQueryChange={onSearchQueryChange}
            onCategoryChange={onCategoryChange}
          />
          <SelfServiceTiles
            contactUrl={contactUrl}
            enhancedSoftware={filteredSoftware}
            isFetching={isFetching}
            isEmptySearch={
              enhancedSoftware.length > 0 &&
              filteredSoftware.length === 0 &&
              !!queryParams.query
            }
            isEmptyCategory={
              enhancedSoftware.length > 0 &&
              filteredSoftware.length === 0 &&
              !queryParams.query &&
              queryParams.category_id !== undefined
            }
            onClickInstallAction={onClickInstallAction}
          />
        </div>
      </>
    );
  }
  return (
    <Card
      className={`${baseClass}__self-service-card`}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
    >
      <SelfServiceHeader contactUrl={contactUrl} />
      <div className={`${baseClass}__content`}>
        <SelfServiceFilters
          query={queryParams.query}
          categoryId={queryParams.category_id}
          categories={categories}
          onSearchQueryChange={onSearchQueryChange}
          onCategoryChange={onCategoryChange}
          installAllSlot={installAllButton}
        />
        <SelfServiceTable
          baseClass={baseClass}
          contactUrl={contactUrl}
          queryParams={queryParams}
          enhancedSoftware={filteredSoftware}
          selfServiceData={selfServiceData}
          tableConfig={tableConfig}
          isFetching={isFetching}
          onSortChange={onSortChange}
          onClientSidePaginationChange={onClientSidePaginationChange}
        />
      </div>
    </Card>
  );
};

export default SelfServiceCard;
