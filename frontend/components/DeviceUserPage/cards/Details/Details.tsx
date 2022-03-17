import React from "react";

interface IDetailsProps {
  onlineCount?: string | undefined;
  offlineCount?: string | undefined;
  isLoadingHosts?: boolean;
  showHostsUI?: boolean;
}

const Details = ({}: IDetailsProps): JSX.Element => {
  return (
    <div className="section about">
      <p className="section__header">About</p>
      <div className="info-grid">
        <div className="info-grid__block">
          <span className="info-grid__header">Last restarted</span>
          <span className="info-grid__data">
            {/* {wrapFleetHelper(humanHostUptime, aboutData.uptime)} */}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Hardware model</span>
          {/* <span className="info-grid__data">{aboutData.hardware_model}</span> */}
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Added to Fleet</span>
          <span className="info-grid__data">
            {/* {wrapFleetHelper(humanHostEnrolled, aboutData.last_enrolled_at)} */}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">Serial number</span>
          {/* <span className="info-grid__data">{aboutData.hardware_serial}</span> */}
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">IP address</span>
          {/* <span className="info-grid__data">{aboutData.primary_ip}</span> */}
        </div>
        {/* {renderDeviceUser()} */}
      </div>
    </div>
  );
};

export default Details;
