import React from "react";

import ReactTooltip from "react-tooltip";

import { IMDMData, IMunkiData, IDeviceUser } from "interfaces/host";
import { humanHostLastRestart, humanHostEnrolled } from "utilities/helpers";

interface IAboutProps {
  aboutData: { [key: string]: any };
  deviceMapping?: IDeviceUser[];
  macadmins?: {
    munki: IMunkiData | null;
    mobile_device_management: IMDMData | null;
  } | null;
  wrapFleetHelper: (helperFn: (value: any) => string, value: string) => string;
  deviceUser?: boolean;
}

const About = ({
  aboutData,
  deviceMapping,
  macadmins,
  wrapFleetHelper,
  deviceUser,
}: IAboutProps): JSX.Element => {
  const renderSerialAndIPs = () => {
    return (
      <>
        <div className="info-grid__block">
          <span className="info-grid__header">Serial number</span>
          <span className="info-grid__data">{aboutData.hardware_serial}</span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Private IP address</span>
          <span className="info-grid__data">{aboutData.primary_ip}</span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Public IP address</span>
          <span className="info-grid__data">{aboutData.public_ip}</span>
        </div>
      </>
    );
  };

  const renderMunkiData = () => {
    if (!macadmins) {
      return null;
    }
    const { munki } = macadmins;
    return munki ? (
      <>
        <div className="info-grid__block">
          <span className="info-grid__header">Munki version</span>
          <span className="info-grid__data">{munki.version || "---"}</span>
        </div>
      </>
    ) : null;
  };

  const renderMdmData = () => {
    if (!macadmins?.mobile_device_management) {
      return null;
    }
    const mdm = macadmins.mobile_device_management;
    return mdm.enrollment_status !== "Unenrolled" ? (
      <>
        <div className="info-grid__block">
          <span className="info-grid__header">MDM enrollment</span>
          <span className="info-grid__data">
            {mdm.enrollment_status || "---"}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">MDM server URL</span>
          <span className="info-grid__data">{mdm.server_url || "---"}</span>
        </div>
      </>
    ) : null;
  };

  const renderDeviceUser = () => {
    if (!deviceMapping) {
      return null;
    }

    const numUsers = deviceMapping.length;
    const tooltipText = deviceMapping.map((d) => (
      <span key={Math.random().toString().slice(2)}>
        {d.email}
        <br />
      </span>
    ));

    return (
      <div className="info-grid__block">
        <span className="info-grid__header">Used by</span>
        <span className="info-grid__data">
          {numUsers > 1 ? (
            <>
              <span data-tip data-for="device_mapping" className="tooltip">
                {`${numUsers} users`}
              </span>
              <ReactTooltip
                effect="solid"
                backgroundColor="#3e4771"
                id="device_mapping"
                data-html
              >
                <span className={`tooltip__tooltip-text`}>{tooltipText}</span>
              </ReactTooltip>
            </>
          ) : (
            deviceMapping[0].email || "---"
          )}
        </span>
      </div>
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
    return (
      <div className="info-grid__block">
        <span className="info-grid__header">Location</span>
        <span className="info-grid__data">{location}</span>
      </div>
    );
  };

  const renderBattery = () => {
    if (
      aboutData.batteries === null ||
      typeof aboutData.batteries !== "object"
    ) {
      return null;
    }
    return (
      <div className="info-grid__block">
        <span className="info-grid__header">Battery condition</span>
        <span className="info-grid__data">
          {aboutData.batteries?.[0]?.health}
        </span>
      </div>
    );
  };

  if (deviceUser) {
    return (
      <div className="section about">
        <p className="section__header">About</p>
        <div className="info-grid">
          <div className="info-grid__block">
            <span className="info-grid__header">Last restarted</span>
            <span className="info-grid__data">
              {humanHostLastRestart(
                aboutData.detail_updated_at,
                aboutData.uptime
              )}
            </span>
          </div>
          <div className="info-grid__block">
            <span className="info-grid__header">Hardware model</span>
            <span className="info-grid__data">{aboutData.hardware_model}</span>
          </div>
          <div className="info-grid__block">
            <span className="info-grid__header">Added to Fleet</span>
            <span className="info-grid__data">
              {wrapFleetHelper(humanHostEnrolled, aboutData.last_enrolled_at)}
            </span>
          </div>
          {renderSerialAndIPs()}
          {renderDeviceUser()}
          {renderBattery()}
        </div>
      </div>
    );
  }

  return (
    <div className="section about">
      <p className="section__header">About</p>
      <div className="info-grid">
        <div className="info-grid__block">
          <span className="info-grid__header">Added to Fleet</span>
          <span className="info-grid__data">
            {wrapFleetHelper(humanHostEnrolled, aboutData.last_enrolled_at)}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Last restarted</span>
          <span className="info-grid__data">
            {humanHostLastRestart(
              aboutData.detail_updated_at,
              aboutData.uptime
            )}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Hardware model</span>
          <span className="info-grid__data">{aboutData.hardware_model}</span>
        </div>
        {renderSerialAndIPs()}
        {renderMunkiData()}
        {renderMdmData()}
        {renderDeviceUser()}
        {renderGeolocation()}
        {renderBattery()}
      </div>
    </div>
  );
};

export default About;
