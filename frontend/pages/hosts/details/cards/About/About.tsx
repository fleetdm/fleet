import React from "react";
import classnames from "classnames";

import { IHostMdmData, IMunkiData } from "interfaces/host";
import { isAndroid, isIPadOrIPhone } from "interfaces/platform";
import { MDM_ENROLLMENT_STATUS_UI_MAP } from "interfaces/mdm";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  MDM_STATUS_TOOLTIP,
  BATTERY_TOOLTIP,
} from "utilities/constants";

import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Card from "components/Card";
import DataSet from "components/DataSet";
import CardHeader from "components/CardHeader";

interface IAboutProps {
  aboutData: { [key: string]: any };
  munki?: IMunkiData | null;
  mdm?: IHostMdmData;
  className?: string;
}

const baseClass = "about-card";

const About = ({
  aboutData,
  munki,
  mdm,
  className,
}: IAboutProps): JSX.Element => {
  const isIosOrIpadosHost = isIPadOrIPhone(aboutData.platform);
  const isAndroidHost = isAndroid(aboutData.platform);

  const renderHardwareSerialAndIPs = () => {
    if (isIosOrIpadosHost) {
      return (
        <>
          <DataSet
            title="Serial number"
            value={<TooltipTruncatedText value={aboutData.hardware_serial} />}
          />
          <DataSet title="Hardware model" value={aboutData.hardware_model} />
        </>
      );
    }

    if (isAndroidHost) {
      return (
        <DataSet title="Hardware model" value={aboutData.hardware_model} />
      );
    }

    return (
      <>
        <DataSet title="Hardware model" value={aboutData.hardware_model} />
        <DataSet
          title="Serial number"
          value={<TooltipTruncatedText value={aboutData.hardware_serial} />}
        />
        <DataSet
          title="Private IP address"
          value={<TooltipTruncatedText value={aboutData.primary_ip} />}
        />
        <DataSet
          title={
            <TooltipWrapper tipContent="The IP address the host uses to connect to Fleet.">
              Public IP address
            </TooltipWrapper>
          }
          value={<TooltipTruncatedText value={aboutData.public_ip} />}
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
            <TooltipWrapper
              tipContent={MDM_STATUS_TOOLTIP[mdm.enrollment_status]}
              underline={mdm.enrollment_status !== "Off"}
            >
              {MDM_ENROLLMENT_STATUS_UI_MAP[mdm.enrollment_status].displayName}
            </TooltipWrapper>
          }
        />
        <DataSet
          title="MDM server URL"
          value={
            <TooltipTruncatedText
              value={mdm.server_url || DEFAULT_EMPTY_CELL_VALUE}
            />
          }
        />
      </>
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

  // TODO(android): confirm visible fields using actual android device data

  const classNames = classnames(baseClass, className);

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <CardHeader header="About" />
      <div className={`${baseClass}__info-grid`}>
        <DataSet
          title="Added to Fleet"
          value={
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={aboutData.last_enrolled_at ?? "Unavailable"}
            />
          }
        />
        {!isIosOrIpadosHost && !isAndroidHost && (
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
        {renderGeolocation()}
        {renderBattery()}
      </div>
    </Card>
  );
};

export default About;
