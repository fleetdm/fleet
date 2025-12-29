import React from "react";

import { IGeoLocation } from "interfaces/host";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";

import { HostMdmDeviceStatusUIState } from "../../helpers";
import DataSet from "components/DataSet";

const baseClass = "location-modal";

const getIosLocationMessage = (
  status: HostMdmDeviceStatusUIState,
  hasLocation: boolean
): JSX.Element | null => {
  const contentClass = `${baseClass}__content`;

  switch (status) {
    case "unlocked":
      return (
        <div className={contentClass}>
          To view location, Apple requires that iOS hosts are locked (Lost Mode)
          first.
        </div>
      );
    case "locking":
      return (
        <div className={contentClass}>
          <p>
            To view location, Apple requires that iOS hosts are locked (Lost
            Mode) first.
          </p>
          <p>
            Lock is pending. Host will lock the next time it checks in to Fleet.
          </p>
        </div>
      );
    case "locating":
      return (
        <div className={contentClass}>
          Location is pending. Host will share location the next time it checks
          in to Fleet.
        </div>
      );
    default:
      if (!hasLocation) {
        return (
          <div className={contentClass}>
            Location not available. Please close this modal and select{" "}
            <strong>Refetch</strong> to fetch latest location.
          </div>
        );
      }
      return null;
  }
};

const buildGoogleMapsLinkFromGeo = (loc: IGeoLocation): string | null => {
  const { country_iso, city_name, geometry } = loc;

  // Prefer coordinates if valid
  if (
    geometry &&
    Array.isArray(geometry.coordinates) &&
    geometry.coordinates.length >= 2
  ) {
    const [lng, lat] = geometry.coordinates; // GeoJSON is [lng, lat]
    if (lat != null && lng != null) {
      return `https://www.google.com/maps?q=${lat},${lng}`;
    }
  }

  // Fallback to "City, Country"
  if (city_name && country_iso) {
    const query = encodeURIComponent(`${city_name}, ${country_iso}`);
    return `https://www.google.com/maps/search/?api=1&query=${query}`;
  }

  return null;
};

interface IIosOrIpadosDetails {
  isIosOrIpadosHost: boolean;
  hostMdmDeviceStatus: HostMdmDeviceStatusUIState;
}

interface ILocationModal {
  hostGeolocation?: IGeoLocation;
  onExit: () => void;
  onClickLock: () => void;
  iosOrIpadosDetails?: IIosOrIpadosDetails;
  detailsUpdatedAt?: string;
}

const LocationModal = ({
  hostGeolocation,
  onExit,
  onClickLock,
  iosOrIpadosDetails,
  detailsUpdatedAt,
}: ILocationModal) => {
  const googleMapsUrl = hostGeolocation
    ? buildGoogleMapsLinkFromGeo(hostGeolocation)
    : null;

  const location = hostGeolocation
    ? [hostGeolocation.city_name, hostGeolocation.country_iso]
        .filter(Boolean)
        .join(", ")
    : "";

  const lastUpdatedAt = detailsUpdatedAt ? (
    <LastUpdatedText
      lastUpdatedAt={detailsUpdatedAt}
      customTooltipText="The last time location data was updated."
    />
  ) : (
    ": unavailable"
  );

  const renderContent = () => {
    if (iosOrIpadosDetails?.isIosOrIpadosHost) {
      const iosContent = getIosLocationMessage(
        iosOrIpadosDetails.hostMdmDeviceStatus,
        Boolean(hostGeolocation)
      );
      if (iosContent) {
        return iosContent;
      }
    }

    if (!hostGeolocation) {
      return null;
    }

    return (
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__updated-at`}>
          <DataSet
            title="Last reported location"
            value={
              <>
                {location} {lastUpdatedAt}
              </>
            }
          />
        </div>
        {googleMapsUrl && (
          <div className={`${baseClass}__link`}>
            <CustomLink url={googleMapsUrl} text="Open in Google Maps" newTab />
          </div>
        )}
      </div>
    );
  };

  const renderFooter = () => {
    const isIos = iosOrIpadosDetails?.isIosOrIpadosHost;
    const status = iosOrIpadosDetails?.hostMdmDeviceStatus;

    if (isIos && status === "unlocked") {
      return (
        <ModalFooter
          primaryButtons={
            <>
              <Button type="button" onClick={onExit} variant="inverse">
                Cancel
              </Button>
              <Button type="button" onClick={onClickLock}>
                Lock
              </Button>
            </>
          }
        />
      );
    }

    return (
      <ModalFooter
        primaryButtons={
          <Button type="button" onClick={onExit}>
            Done
          </Button>
        }
      />
    );
  };

  return (
    <Modal title="Location" className={baseClass} onExit={onExit}>
      <>
        {renderContent()}
        {renderFooter()}
      </>
    </Modal>
  );
};

export default LocationModal;
