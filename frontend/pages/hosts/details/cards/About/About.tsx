import React from "react";

import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
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
  BATTERY_TOOLTIP,
} from "utilities/constants";
import DataSet from "components/DataSet";
import classnames from "classnames";

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
  const isIosOrIpadosHost =
    aboutData.platform === "ios" || aboutData.platform === "ipados";

  const renderHardwareSerialAndIPs = () => {
    if (isIosOrIpadosHost) {
      return (
        <>
          <DataSet title="Serial number" value={aboutData.hardware_serial} />
          <DataSet title="Hardware model" value={aboutData.hardware_model} />
        </>
      );
    }

    return (
      <>
        <DataSet title="Hardware model" value={aboutData.hardware_model} />
        <DataSet title="Serial number" value={aboutData.hardware_serial} />
        <DataSet title="Private IP address" value={aboutData.primary_ip} />
        <DataSet
          title={
            <TooltipWrapper tipContent="The IP address the host uses to connect to Fleet.">
              Public IP address
            </TooltipWrapper>
          }
          value={aboutData.public_ip}
        />
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
            !MDM_STATUS_TOOLTIP[mdm.enrollment_status] ? (
              mdm.enrollment_status
            ) : (
              <TooltipWrapper
                tipContent={MDM_STATUS_TOOLTIP[mdm.enrollment_status]}
              >
                {mdm.enrollment_status}
              </TooltipWrapper>
            )
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
          <span className={`${baseClass}__device-mapping__primary-user`}>
            {email}{" "}
            <span
              className={`${baseClass}__device-mapping__source`}
            >{`(${source})`}</span>
          </span>
        );
      }
    }
    return (
      <DataSet
        title="Used by"
        value={
          <div className={`${baseClass}__used-by`}>
            {newDeviceMapping.length > 1 ? (
              <>
                <span className={`${baseClass}__multiple`}>
                  <TooltipTruncatedText value={displayPrimaryUser} />
                </span>
                <TooltipWrapper
                  tipContent={getDeviceUserTipContent(newDeviceMapping)}
                >
                  <span className="device-mapping__more">{` +${
                    newDeviceMapping.length - 1
                  } more`}</span>
                </TooltipWrapper>
              </>
            ) : (
              <span className={`${baseClass}__single`}>
                <TooltipTruncatedText value={displayPrimaryUser} />
              </span>
            )}
          </div>
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
      typeof aboutData.batteries !== "object" ||
      aboutData.batteries?.[0]?.health === "Unknown"
    ) {
      return null;
    }
    return (
      <DataSet
        title="Battery condition"
        value={
          <TooltipWrapper
            tipContent={BATTERY_TOOLTIP[aboutData.batteries?.[0]?.health]}
          >
            {aboutData.batteries?.[0]?.health}
          </TooltipWrapper>
        }
      />
    );
  };

  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      paddingSize="large"
      className={baseClass}
    >
      <p className="card__header">About</p>
      <div className="info-flex">
        <DataSet
          title="Added to Fleet"
          value={
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={aboutData.last_enrolled_at ?? "Unavailable"}
            />
          }
        />
        {!isIosOrIpadosHost && (
          <DataSet
            title="Last restarted"
            value={
              <HumanTimeDiffWithFleetLaunchCutoff
                timeString={aboutData.last_restarted_at}
              />
            }
          />
        )}
        {renderHardwareSerialAndIPs()}
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
