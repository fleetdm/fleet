import React from "react";

import ReactTooltip from "react-tooltip";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import Card from "components/Card";

import {
  IHostMdmData,
  IMunkiData,
  IDeviceUser,
  mapDeviceUsersForDisplay,
} from "interfaces/host";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  MDM_STATUS_TOOLTIP,
} from "utilities/constants";
import { COLORS } from "styles/var/colors";
import DataSet from "components/DataSet";

const getDeviceUserTipContent = (deviceMapping: IDeviceUser[]) => {
  if (deviceMapping.length === 0) {
    return [];
  }
  const format = (d: IDeviceUser) =>
    d.source ? `${d.email} (${d.source})` : d.email;

  return deviceMapping.slice(1).map((d) => (
    <span key={format(d)}>
      {format(d)}
      <br />
    </span>
  ));
};

interface IAboutProps {
  aboutData: { [key: string]: any };
  deviceMapping?: IDeviceUser[];
  munki?: IMunkiData | null;
  mdm?: IHostMdmData;
}

const baseClass = "about-card";

const About = ({
  aboutData,
  deviceMapping,
  munki,
  mdm,
}: IAboutProps): JSX.Element => {
  const renderPublicIp = () => {
    if (aboutData.public_ip !== DEFAULT_EMPTY_CELL_VALUE) {
      return aboutData.public_ip;
    }
    return (
      <>
        <span
          className="text-cell text-muted tooltip"
          data-tip
          data-for="public-ip-tooltip"
        >
          {aboutData.public_ip}
        </span>
        <ReactTooltip
          place="bottom"
          effect="solid"
          backgroundColor={COLORS["tooltip-bg"]}
          id="public-ip-tooltip"
          data-html
          clickable
          delayHide={200} // need delay set to hover using clickable
        >
          Public IP address could not be
          <br /> determined.{" "}
          <CustomLink
            url="https://fleetdm.com/docs/deploying/configuration#public-i-ps-of-devices"
            text="Learn more"
            newTab
            iconColor="core-fleet-white"
          />
        </ReactTooltip>
      </>
    );
  };

  const renderSerialAndIPs = () => {
    return (
      <>
        <DataSet title="Serial number" value={aboutData.hardware_serial} />
        <DataSet title="Private IP address" value={aboutData.primary_ip} />
        <DataSet title="Public IP address" value={renderPublicIp()} />
      </>
    );
  };

  const renderMunkiData = () => {
    return munki ? (
      <>
        <DataSet
          title="Munki version"
          value={munki.version || DEFAULT_EMPTY_CELL_VALUE}
        />
      </>
    ) : null;
  };

  const renderMdmData = () => {
    if (!mdm?.enrollment_status) {
      return null;
    }
    return (
      <>
        <DataSet
          title="MDM status"
          value={
            <TooltipWrapper
              tipContent={MDM_STATUS_TOOLTIP[mdm.enrollment_status]}
            >
              {mdm.enrollment_status}
            </TooltipWrapper>
          }
        />
        <DataSet
          title="MDM server URL"
          value={mdm.server_url || DEFAULT_EMPTY_CELL_VALUE}
        />
      </>
    );
  };

  const renderDeviceUser = () => {
    if (!deviceMapping) {
      return null;
    }

    let displayPrimaryUser: React.ReactNode = DEFAULT_EMPTY_CELL_VALUE;

    const newDeviceMapping = mapDeviceUsersForDisplay(deviceMapping);
    if (newDeviceMapping[0]) {
      const { email, source } = newDeviceMapping[0];
      if (!source) {
        displayPrimaryUser = email;
      } else {
        displayPrimaryUser = (
          <>
            {email}{" "}
            <span className="device-mapping__source">{`(${source})`}</span>
          </>
        );
      }
    }
    return (
      <DataSet
        title="Used by"
        value={
          newDeviceMapping.length > 1 ? (
            <TooltipWrapper
              tipContent={getDeviceUserTipContent(newDeviceMapping)}
            >
              {displayPrimaryUser}
              <span className="device-mapping__more">{` +${
                newDeviceMapping.length - 1
              } more`}</span>
            </TooltipWrapper>
          ) : (
            displayPrimaryUser
          )
        }
      />
    );
  };

  const renderGeolocation = () => {
    const geolocation = aboutData.geolocation;

    if (!geolocation) {
      return null;
    }

    const location = [geolocation?.city_name, geolocation?.country_iso]
      .filter(Boolean)
      .join(", ");
    return <DataSet title="Location" value={location} />;
  };

  const renderBattery = () => {
    if (
      aboutData.batteries === null ||
      typeof aboutData.batteries !== "object"
    ) {
      return null;
    }
    return (
      <DataSet
        title="Battery condition"
        value={aboutData.batteries?.[0]?.health}
      />
    );
  };

  return (
    <Card
      borderRadiusSize="large"
      includeShadow
      largePadding
      className={baseClass}
    >
      <p className="card__header">About</p>
      <div className="info-grid">
        <DataSet
          title="Added to Fleet"
          value={
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={aboutData.last_enrolled_at ?? "Unavailable"}
            />
          }
        />
        <DataSet
          title="Last restarted"
          value={
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={aboutData.last_restarted_at}
            />
          }
        />
        <DataSet title="Hardware model" value={aboutData.hardware_model} />
        {renderSerialAndIPs()}
        {renderMunkiData()}
        {renderMdmData()}
        {renderDeviceUser()}
        {renderGeolocation()}
        {renderBattery()}
      </div>
    </Card>
  );
};

export default About;
