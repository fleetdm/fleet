import React, { useState, useEffect, useMemo } from "react";

import {
  IDeviceSoftware,
  IDeviceSoftwareWithUiStatus,
} from "interfaces/software";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import Button from "components/buttons/Button";
import Pagination from "components/Pagination";
import Spinner from "components/Spinner";
import DeviceUserError from "components/DeviceUserError";
import UpdateSoftwareItem from "./UpdateSoftwareItem";

const getUpdatesPageSize = (width: number): number => {
  if (width >= 1400) return 4;
  if (width >= 880) return 3;
  return 2;
};

const baseClass = "updates-card";

interface IUpdatesCardProps {
  enhancedSoftware: IDeviceSoftwareWithUiStatus[];
  isLoading: boolean;
  isError: boolean;
  onClickUpdateAll: () => void;
  onClickUpdateAction: (id: number) => void;
  onClickFailedUpdateStatus: (software: IDeviceSoftware) => void;
}

const UpdatesCard = ({
  enhancedSoftware,
  isLoading,
  isError,
  onClickUpdateAll,
  onClickUpdateAction,
  onClickFailedUpdateStatus,
}: IUpdatesCardProps) => {
  const [updatesPage, setUpdatesPage] = useState(0);
  const [updatesPageSize, setUpdatesPageSize] = useState(() =>
    getUpdatesPageSize(window.innerWidth)
  );

  const updateSoftware = enhancedSoftware.filter(
    (software) =>
      software.ui_status === "updating" ||
      software.ui_status === "recently_updated" ||
      software.ui_status === "pending_update" || // Should never show as self-service = host online
      software.ui_status === "update_available" ||
      software.ui_status === "failed_install_update_available" ||
      software.ui_status === "failed_uninstall_update_available"
  );

  // The page size only changes at 2 breakpoints and state update is very lightweight.
  // Page size controls the number of shown cards and current page; no API calls or
  // expensive UI updates occur so debouncing the resize handler isn’t necessary.
  useEffect(() => {
    const handleResize = () => {
      const newPageSize = getUpdatesPageSize(window.innerWidth);
      setUpdatesPageSize(() => {
        const newTotalPages = Math.ceil(updateSoftware.length / newPageSize);
        setUpdatesPage((prevPage) => {
          // If the current page is now out of range, go to the last valid page
          return Math.min(prevPage, Math.max(0, newTotalPages - 1));
        });
        return newPageSize;
      });
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [updateSoftware.length]);

  const paginatedUpdates = useMemo(() => {
    const start = updatesPage * updatesPageSize;
    return updateSoftware.slice(start, start + updatesPageSize);
  }, [updateSoftware, updatesPage, updatesPageSize]);

  const totalUpdatesPages = Math.ceil(updateSoftware.length / updatesPageSize);

  const onNextUpdatesPage = () => {
    setUpdatesPage((prev) => Math.min(prev + 1, totalUpdatesPages - 1));
  };

  const onPreviousUpdatesPage = () => {
    setUpdatesPage((prev) => Math.max(prev - 1, 0));
  };

  const disableUpdateAllButton = useMemo(() => {
    // Disable if all statuses are "updating" or "recently_updated"
    return (
      updateSoftware.length > 0 &&
      updateSoftware.every(
        (software) =>
          software.ui_status === "updating" ||
          software.ui_status === "recently_updated"
      )
    );
  }, [updateSoftware]);

  if (paginatedUpdates.length === 0) {
    return null;
  }

  const renderUpdatesContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
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
      className={`${baseClass}__updates-card`}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <div className={`${baseClass}__header`}>
        <CardHeader
          header="Updates"
          subheader={
            <>
              Your device has outdated software. Update to address potential
              security vulnerabilities or compatibility issues.
            </>
          }
        />
        <Button disabled={disableUpdateAllButton} onClick={onClickUpdateAll}>
          Update all
        </Button>
      </div>
      {renderUpdatesContent()}
    </Card>
  );
};

export default UpdatesCard;
