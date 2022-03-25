import React from "react";

import ReactTooltip from "react-tooltip";

import { IMDMData, IMunkiData, IDeviceUser } from "interfaces/host";
import { humanHostUptime, humanHostEnrolled } from "fleet/helpers";

const baseClass = "host-summary";

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
  const renderDeviceUser = () => {
    const numUsers = deviceMapping?.length;
    if (numUsers) {
      return (
        <div className="info-grid__block">
          <span className="info-grid__header">Used by</span>
          <span className="info-grid__data">
            {numUsers === 1 && deviceMapping ? (
              deviceMapping[0].email || "---"
            ) : (
              <span className={`${baseClass}__device-mapping`}>
                <span
                  className="device-user"
                  data-tip
                  data-for="device-user-tooltip"
                >
                  {`${numUsers} users`}
                </span>
                <ReactTooltip
                  place="top"
                  type="dark"
                  effect="solid"
                  id="device-user-tooltip"
                  backgroundColor="#3e4771"
                >
                  <div
                    className={`${baseClass}__tooltip-text device-user-tooltip`}
                  >
                    {deviceMapping &&
                      deviceMapping.map((user: any, i: number, arr: any) => (
                        <span key={user.email}>{`${user.email}${
                          i < arr.length - 1 ? ", " : ""
                        }`}</span>
                      ))}
                  </div>
                </ReactTooltip>
              </span>
            )}
          </span>
        </div>
      );
    }
    return null;
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

  if (deviceUser) {
    return (
      <div className="section about">
        <p className="section__header">About</p>
        <div className="info-grid">
          <div className="info-grid__block">
            <span className="info-grid__header">Last restarted</span>
            <span className="info-grid__data">
              {wrapFleetHelper(humanHostUptime, aboutData.uptime)}
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
          <div className="info-grid__block">
            <span className="info-grid__header">Serial number</span>
            <span className="info-grid__data">{aboutData.hardware_serial}</span>
          </div>
          <div className="info-grid__block">
            <span className="info-grid__header">Internal IP address</span>
            <span className="info-grid__data">{aboutData.primary_ip}</span>
          </div>
          <div className="info-grid__block">
            <span className="info-grid__header">Public IP address</span>
            <span className="info-grid__data">{aboutData.public_ip}</span>
          </div>
          {renderDeviceUser()}
        </div>
      </div>
    );
  }

  return (
    <div className="section about">
      <p className="section__header">About</p>
      <div className="info-grid">
        <div className="info-grid__block">
          <span className="info-grid__header">First enrolled</span>
          <span className="info-grid__data">
            {wrapFleetHelper(humanHostEnrolled, aboutData.last_enrolled_at)}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Last restarted</span>
          <span className="info-grid__data">
            {wrapFleetHelper(humanHostUptime, aboutData.uptime)}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Hardware model</span>
          <span className="info-grid__data">{aboutData.hardware_model}</span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Serial number</span>
          <span className="info-grid__data">{aboutData.hardware_serial}</span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Internal IP address</span>
          <span className="info-grid__data">{aboutData.primary_ip}</span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Public IP address</span>
          <span className="info-grid__data">{aboutData.public_ip}</span>
        </div>
        {renderMunkiData()}
        {renderMdmData()}
        {renderDeviceUser()}
        {renderGeolocation()}
      </div>
    </div>
  );
};

export default About;
