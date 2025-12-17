/** Mobile preview modal uses a screenshot
 * Non-mobile now uses HTML/CSS instead for
 * maintainability as the self-selvice UI changes
 *
 * Currently only shown from the edit UI, though wired through the Add UI
 * Users currently can set categories only when editing a curent installer
 */

import React, { useContext } from "react";

import { Column } from "react-table";
import { noop } from "lodash";
import { AppContext } from "context/app";
import { IHeaderProps } from "interfaces/datatable_config";

import TableContainer from "components/TableContainer";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Card from "components/Card";
import SelfServiceHeader from "pages/hosts/details/cards/Software/SelfService/components/SelfServiceHeader";
import SearchField from "components/forms/fields/SearchField";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import CategoriesMenu from "pages/hosts/details/cards/Software/SelfService/components/CategoriesMenu";
import { CATEGORIES_NAV_ITEMS } from "pages/hosts/details/cards/Software/SelfService/helpers";

import CategoriesEndUserExperiencePreviewMobile from "../../../../../../assets/images/categories-end-user-experience-preview-mobile@2x.png";

const baseClass = "categories-end-user-experience-preview-modal";

interface ISoftwareRow {
  name: React.ReactNode;
}

type ITableHeaderProps = IHeaderProps<ISoftwareRow>;

const columns: Column<ISoftwareRow>[] = [
  {
    Header: (cellProps: ITableHeaderProps) => (
      <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "name",
    disableSortBy: true,
  },
];

const getData = (
  name: string,
  displayName: string,
  iconUrl: string | null,
  source?: string
): ISoftwareRow[] => [
  {
    name: (
      <SoftwareNameCell
        name={name}
        display_name={displayName}
        source={source}
        iconUrl={iconUrl}
        pageContext="deviceUser"
        isSelfService
      />
    ),
  },
  {
    name: (
      <SoftwareNameCell
        name="1Password"
        source="apps"
        pageContext="deviceUser"
        isSelfService
      />
    ),
  },
  {
    name: (
      <SoftwareNameCell
        name="Adobe Acrobat Reader"
        source="apps"
        pageContext="deviceUser"
        isSelfService
      />
    ),
  },
  {
    name: (
      <SoftwareNameCell
        name="Box Drive"
        source="apps"
        pageContext="deviceUser"
        isSelfService
      />
    ),
  },
];

const EmptyState = () => <div>No software found</div>;

interface BasicSoftwareTableProps {
  name: string;
  displayName: string;
  source?: string;
  iconUrl?: string | null;
}

const BasicSoftwareTable = ({
  name,
  displayName,
  source,
  iconUrl = null,
}: BasicSoftwareTableProps) => {
  return (
    <TableContainer<ISoftwareRow>
      columnConfigs={columns}
      data={getData(name, displayName, iconUrl, source)}
      isLoading={false}
      emptyComponent={EmptyState}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      searchable={false}
      disablePagination
      disableCount
      disableTableHeader
    />
  );
};

interface ICategoriesEndUserExperienceModal {
  onCancel: () => void;
  isIosOrIpadosApp?: boolean;
  name?: string;
  displayName?: string;
  iconUrl?: string;
  source?: string;
}

const CategoriesEndUserExperienceModal = ({
  onCancel,
  isIosOrIpadosApp = false,
  name = "Software name",
  displayName = "Software name",
  iconUrl,
  source,
}: ICategoriesEndUserExperienceModal): JSX.Element => {
  const { config } = useContext(AppContext);
  return (
    <Modal title="End user experience" onExit={onCancel} className={baseClass}>
      <>
        <span>What end users see:</span>

        {isIosOrIpadosApp ? (
          <div className={`${baseClass}__preview`}>
            <img
              src={CategoriesEndUserExperiencePreviewMobile}
              alt="Categories end user experience preview"
            />
          </div>
        ) : (
          <Card
            borderRadiusSize="medium"
            color="grey"
            className={`${baseClass}__preview-card`}
            paddingSize="xlarge"
          >
            <div className={`${baseClass}__disabled-overlay`} />
            <Card
              className={`${baseClass}__preview-card__self-service`}
              borderRadiusSize="xxlarge"
            >
              <SelfServiceHeader
                contactUrl={config?.org_info.contact_url || ""}
                variant="preview"
              />
              <SearchField
                placeholder="Search by name"
                onChange={noop}
                disabled
              />
              <div className={`${baseClass}__table`}>
                <CategoriesMenu
                  categories={CATEGORIES_NAV_ITEMS}
                  queryParams={{
                    query: "",
                    order_direction: "asc",
                    order_key: "name",
                    page: 0,
                    per_page: 100,
                  }}
                  readOnly
                />
                <BasicSoftwareTable
                  name={name}
                  displayName={displayName}
                  source={source}
                  iconUrl={iconUrl}
                />
              </div>
            </Card>
          </Card>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default CategoriesEndUserExperienceModal;
