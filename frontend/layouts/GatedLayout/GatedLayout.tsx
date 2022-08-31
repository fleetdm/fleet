import React, { useContext } from "react";
import { NotificationContext } from "context/notification";
import FlashMessage from "components/FlashMessage";

interface IGatedLayoutProps {
  children: React.ReactNode;
}

const GatedLayout = ({ children }: IGatedLayoutProps): JSX.Element => {
  const { notification, hideFlash } = useContext(NotificationContext);

  return (
    <div className="gated-layout">
      <FlashMessage
        fullWidth
        notification={notification}
        onRemoveFlash={hideFlash}
      />
      {children}
    </div>
  );
};

export default GatedLayout;
