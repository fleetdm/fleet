import React from "react";

import {
  IDeviceSoftware,
  IDeviceSoftwareWithUiStatus,
} from "interfaces/software";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import DeviceUserError from "components/DeviceUserError";
import Pagination from "components/Pagination";

import UpdateSoftwareItem from "./UpdateSoftwareItem";

interface IUpdatesCardProps {
  contactUrl: string;
  disableUpdateAllButton: boolean;
  onClickUpdateAll: () => void;
  paginatedUpdates: IDeviceSoftwareWithUiStatus[];
  isLoading: boolean;
  isError: boolean;
  onClickUpdateAction: (id: number) => void;
  onClickFailedUpdateStatus: (s: IDeviceSoftware) => void;
  updatesPage: number;
  totalUpdatesPages: number;
  onNextUpdatesPage: () => void;
  onPreviousUpdatesPage: () => void;
}

const baseClass = "updates-card";

const UpdatesCard = ({
  contactUrl,
  disableUpdateAllButton,
  onClickUpdateAll,
  paginatedUpdates,
  isLoading,
  isError,
  onClickUpdateAction,
  onClickFailedUpdateStatus,
  updatesPage,
  totalUpdatesPages,
  onNextUpdatesPage,
  onPreviousUpdatesPage,
}: IUpdatesCardProps) => {
  if (paginatedUpdates.length === 0) return null;

  const renderCardContent = () => {
    if (isLoading) {
      return <Spinner />;
    } else if (isError) {
      return <DeviceUserError />;
    }
    return (
      <>
        <div className={`${baseClass}__items`}>
          {paginatedUpdates.map((s) => (
            <UpdateSoftwareItem
              key={s.id}
              software={s}
              onClickUpdateAction={onClickUpdateAction}
              onShowInstallerDetails={() => onClickFailedUpdateStatus(s)}
            />
          ))}
        </div>
        <Pagination
          disableNext={updatesPage >= totalUpdatesPages - 1}
          disablePrev={updatesPage === 0}
          hidePagination={
            updatesPage >= totalUpdatesPages - 1 && updatesPage === 0
          }
          onNextPage={onNextUpdatesPage}
          onPrevPage={onPreviousUpdatesPage}
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
      <div className={`${baseClass}__header`}>
        <CardHeader
          header="Updates"
          subheader={
            <>
              The following app require updating.{" "}
              {contactUrl && (
                <span>
                  If you need help,{" "}
                  <CustomLink url={contactUrl} text="reach out to IT" newTab />
                </span>
              )}
            </>
          }
        />
        <Button disabled={disableUpdateAllButton} onClick={onClickUpdateAll}>
          Update all
        </Button>
      </div>
      {renderCardContent()}
    </Card>
  );
};

export default UpdatesCard;
