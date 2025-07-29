import React from "react";
import { SingleValue } from "react-select-5";

import { IDeviceSoftware } from "interfaces/software";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";
import TableContainer from "components/TableContainer";
import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import EmptyTable from "components/EmptyTable";
import Spinner from "components/Spinner";
import DeviceUserError from "components/DeviceUserError";
import Pagination from "components/Pagination";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import SearchField from "components/forms/fields/SearchField";
import CategoriesMenu from "./CategoriesMenu";
import {
  ICategory,
  filterSoftwareByCategory,
  CATEGORIES_NAV_ITEMS,
} from "../helpers";

interface ISelfServiceCardProps {
  isLoading: boolean;
  isError: boolean;
  isFetching: boolean;
  selfServiceData?: IGetDeviceSoftwareResponse;
  enhancedSoftware: IDeviceSoftware[];
  tableConfig: any;
  queryParams: any;
  contactUrl: string;
  onSearchQueryChange: (value: string) => void;
  onCategoriesDropdownChange: (option: SingleValue<any>) => void;
  onNextPage: () => void;
  onPrevPage: () => void;
  isEmpty: boolean;
  isEmptySearch: boolean;
}

const baseClass = "self-service-card";

const SelfServiceCard = ({
  isLoading,
  isError,
  isFetching,
  selfServiceData,
  enhancedSoftware,
  tableConfig,
  queryParams,
  contactUrl,
  onSearchQueryChange,
  onCategoriesDropdownChange,
  onNextPage,
  onPrevPage,
  isEmpty,
  isEmptySearch,
}: ISelfServiceCardProps) => {
  const renderCardBody = () => {
    if (isLoading) {
      return <Spinner />;
    }
    if (isError) {
      return <DeviceUserError />;
    }
    if ((isEmpty || !selfServiceData) && !isFetching) {
      return (
        <EmptyTable
          graphicName="empty-software"
          header="No self-service software available yet"
          info="Your organization didn't add any self-service software. If you need any, reach out to your IT department."
        />
      );
    }

    return (
      <>
        <div className={`${baseClass}__header-filters`}>
          <SearchField
            placeholder="Search by name"
            onChange={onSearchQueryChange}
            defaultValue={queryParams.query}
          />
          <DropdownWrapper
            options={CATEGORIES_NAV_ITEMS.map((category: ICategory) => ({
              ...category,
              value: String(category.id),
            }))}
            value={String(queryParams.category_id || 0)}
            onChange={onCategoriesDropdownChange}
            name="categories-dropdown"
            className={`${baseClass}__categories-dropdown`}
          />
        </div>
        <div className={`${baseClass}__table`}>
          <CategoriesMenu
            queryParams={queryParams}
            categories={CATEGORIES_NAV_ITEMS}
          />
          <TableContainer
            columnConfigs={tableConfig}
            data={filterSoftwareByCategory(
              enhancedSoftware || [],
              queryParams.category_id
            )}
            isLoading={isFetching}
            defaultSortHeader="name"
            defaultSortDirection="asc"
            pageIndex={0}
            disableNextPage={selfServiceData?.meta.has_next_results === false}
            pageSize={20}
            searchQuery={queryParams.query}
            searchQueryColumn="name"
            isClientSideFilter
            isClientSidePagination
            emptyComponent={() => {
              return isEmptySearch ? (
                <EmptyTable
                  graphicName="empty-search-question"
                  header="No items match the current search criteria"
                  info={
                    <>
                      Not finding what you&apos;re looking for?{" "}
                      <CustomLink
                        url={contactUrl}
                        text="Reach out to IT"
                        newTab
                      />
                    </>
                  }
                />
              ) : (
                <EmptySoftwareTable />
              );
            }}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableTableHeader
            disableCount
          />
        </div>
        <Pagination
          disableNext={selfServiceData?.meta.has_next_results === false}
          disablePrev={selfServiceData?.meta.has_previous_results === false}
          hidePagination={
            selfServiceData?.meta.has_next_results === false &&
            selfServiceData?.meta.has_previous_results === false
          }
          onNextPage={onNextPage}
          onPrevPage={onPrevPage}
          className={`${baseClass}__pagination`}
        />
      </>
    );
  };

  return (
    <Card
      className={baseClass}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <CardHeader
        header="Self-service"
        subheader={
          <>
            Install organization-approved apps provided by your IT department.{" "}
            {contactUrl && (
              <span>
                If you need help,{" "}
                <CustomLink url={contactUrl} text="reach out to IT" newTab />
              </span>
            )}
          </>
        }
      />
      {renderCardBody()}
    </Card>
  );
};

export default SelfServiceCard;
