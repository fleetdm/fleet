import React from "react";
import {
  IDeviceSoftwareWithUiStatus,
  SCRIPT_PACKAGE_SOURCES,
} from "interfaces/software";
import Card from "components/Card";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Spinner from "components/Spinner";
import TileActionStatus from "../TileActionStatus";

const baseClass = "self-service-tiles-list";

const tileBaseClass = "self-service-tile";

interface SelfServiceTilesProps {
  enhancedSoftware: IDeviceSoftwareWithUiStatus[];
  contactUrl: string;
  onClickInstallAction: (
    softwareId: number,
    isSoftwarePackage?: boolean
  ) => void;
  isEmptySearch?: boolean;
  isFetching?: boolean;
}

const SelfServiceTiles = ({
  enhancedSoftware,
  contactUrl,
  onClickInstallAction,
  isEmptySearch,
  isFetching,
}: SelfServiceTilesProps) => {
  if (isFetching) {
    return <Spinner />;
  }

  if (isEmptySearch) {
    return (
      <EmptyTable
        graphicName="empty-search-question"
        header="No items match your search"
        info={
          <>
            Not finding what you&apos;re looking for?{" "}
            <CustomLink url={contactUrl} text="Reach out to IT" newTab />
          </>
        }
      />
    );
  }

  return (
    <div className={baseClass}>
      {enhancedSoftware.map((software) => (
        <Card className={tileBaseClass} key={software.id}>
          <div className={`${tileBaseClass}__item`}>
            <div className={`${tileBaseClass}__item-icon`}>
              <SoftwareIcon
                url={software.icon_url}
                name={software.name}
                source={software.source}
                size="large"
              />
            </div>
            <div className={`${tileBaseClass}__item-name-version`}>
              <div className={`${tileBaseClass}__item-name`}>
                <TooltipTruncatedText isMobileView value={software.name} />
              </div>
              <div className={`${tileBaseClass}__item-version`}>
                {software.software_package?.version ||
                  software.app_store_app?.version}
              </div>
            </div>
          </div>
          <TileActionStatus
            software={software}
            onActionClick={() =>
              onClickInstallAction(
                software.id,
                SCRIPT_PACKAGE_SOURCES.includes(software.source)
              )
            }
          />
        </Card>
      ))}
    </div>
  );
};

export default SelfServiceTiles;
